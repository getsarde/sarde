package deploy

import (
	"context"
	"crypto/sha1" //nolint:gosec // Netlify and Vercel address files by SHA1; not a security use.
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/workers"
)

// File is one file of the built site as a deployer sees it.
type File struct {
	// Rel is the path relative to the output directory, with forward
	// slashes and no leading slash ("blog/index.html"). It is the only path
	// form that ever reaches a provider manifest.
	Rel  string
	Abs  string
	Size int64
}

// collectFiles walks distDir and returns every regular file, sorted by Rel.
// It skips the build lock file at the root (.sarde.lock, which the build
// leaves on disk on purpose), symlinks and other non-regular files.
func collectFiles(distDir string, rep Reporter) ([]File, error) {
	rep = orNop(rep)
	root, err := filepath.Abs(distDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output directory: %w", err)
	}
	var files []File
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(rel, consts.FileOutputLock) {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			rep.Log(LevelWarn, fmt.Sprintf("skipping symlink %s", rel))
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, File{Rel: rel, Abs: p, Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading output directory: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Rel < files[j].Rel })
	return files, nil
}

// totalSize sums the sizes of files.
func totalSize(files []File) int64 {
	var n int64
	for _, f := range files {
		n += f.Size
	}
	return n
}

// sha1File returns the lowercase hex SHA1 of a file's contents.
func sha1File(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New() //nolint:gosec // content address, see import comment
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashAll hashes every file in parallel and returns the digests in the same
// order as files, reporting StepHash progress.
func hashAll(ctx context.Context, files []File, hash func(File) (string, error), rep Reporter) ([]string, error) {
	rep = orNop(rep)
	out := make([]string, len(files))
	var done atomic.Int64
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workers.IOLimit(len(files)))
	for i, f := range files {
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			sum, err := hash(f)
			if err != nil {
				return fmt.Errorf("hashing %s: %w", f.Rel, err)
			}
			out[i] = sum
			rep.Progress(Progress{Step: StepHash, Done: int(done.Add(1)), Total: len(files)})
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// webContentTypes pins the types that matter for a static site. The system
// MIME table is not trusted for these: on Windows it comes from the registry,
// where .js is sometimes text/plain.
var webContentTypes = map[string]string{
	".html":        "text/html; charset=utf-8",
	".htm":         "text/html; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".mjs":         "text/javascript; charset=utf-8",
	".json":        "application/json",
	".map":         "application/json",
	".webmanifest": "application/manifest+json",
	".xml":         "application/xml",
	".txt":         "text/plain; charset=utf-8",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".jpg":         "image/jpeg",
	".jpeg":        "image/jpeg",
	".gif":         "image/gif",
	".webp":        "image/webp",
	".avif":        "image/avif",
	".ico":         "image/x-icon",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".ttf":         "font/ttf",
	".otf":         "font/otf",
	".pdf":         "application/pdf",
	".wasm":        "application/wasm",
	".mp4":         "video/mp4",
	".webm":        "video/webm",
	".mp3":         "audio/mpeg",
}

// contentTypeFor returns the Content-Type for a file by extension: the pinned
// web table first, then the system table, then application/octet-stream.
func contentTypeFor(rel string) string {
	ext := strings.ToLower(path.Ext(rel))
	if t, ok := webContentTypes[ext]; ok {
		return t
	}
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "application/octet-stream"
}

// plainDir strips a Windows verbatim prefix (`\\?\C:\...`, `\\?\UNC\host\share`)
// from a directory. Callers such as Sarde Studio pass canonicalized paths,
// and cmd.exe refuses a verbatim path as its working directory ("UNC paths
// are not supported"), silently running a custom deploy command in C:\Windows.
func plainDir(p string) string {
	if strings.HasPrefix(p, `\\?\UNC\`) {
		return `\\` + p[len(`\\?\UNC\`):]
	}
	return strings.TrimPrefix(p, `\\?\`)
}

// nodeExtname mirrors Node's path.extname for a forward-slash path: the
// extension starts at the last dot of the base name, a dot at position 0 does
// not count, and a trailing dot yields ".". Cloudflare's asset hash is built
// from this value, so it must match Node exactly (Go's path.Ext returns
// ".nojekyll" for ".nojekyll"; Node returns "").
func nodeExtname(p string) string {
	base := p
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		base = p[i+1:]
	}
	dot := strings.LastIndexByte(base, '.')
	if dot <= 0 {
		return ""
	}
	return base[dot:]
}
