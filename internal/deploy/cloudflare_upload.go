package deploy

// Cloudflare Pages asset upload. These endpoints are not in Cloudflare's
// public API reference; they follow wrangler's implementation
// (packages/wrangler/src/pages/upload.ts and
// packages/deploy-helpers/src/deploy/helpers/hash.ts in
// github.com/cloudflare/workers-sdk). If Cloudflare changes them, the
// fallback is provider: custom with `wrangler pages deploy`.

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/zeebo/blake3"
)

const (
	cfBucketBytes  = 40 << 20
	cfBucketFiles  = 2000
	cfUploadsAtOne = 3
	cfJWTExpired   = "8000013"
)

// cloudflareHashFile computes the asset key wrangler uses:
// blake3(base64(content) + extension without its dot), hex, first 32
// characters. The extension follows Node's path.extname (nodeExtname).
func cloudflareHashFile(f File) (string, error) {
	data, err := os.ReadFile(f.Abs)
	if err != nil {
		return "", err
	}
	return cloudflareHash(data, f.Rel), nil
}

func cloudflareHash(data []byte, rel string) string {
	h := blake3.New()
	enc := base64.NewEncoder(base64.StdEncoding, h)
	_, _ = enc.Write(data)
	_ = enc.Close()
	ext := nodeExtname(rel)
	if ext != "" {
		_, _ = io.WriteString(h, ext[1:])
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// cloudflareUpload holds the upload JWT, refreshing it when it expires.
type cloudflareUpload struct {
	d        *CloudflareDeployer
	c        *apiClient
	rep      Reporter
	maxFiles int

	mu  sync.Mutex
	jwt string
}

func newCloudflareUpload(ctx context.Context, d *CloudflareDeployer, c *apiClient, rep Reporter) (*cloudflareUpload, error) {
	rep.Step(StepPrepare, "Requesting an upload token")
	jwt, err := d.uploadToken(ctx, c)
	if err != nil {
		return nil, err
	}
	return &cloudflareUpload{d: d, c: c, rep: rep, maxFiles: jwtMaxFiles(jwt), jwt: jwt}, nil
}

func (u *cloudflareUpload) token() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.jwt
}

func (u *cloudflareUpload) refresh(ctx context.Context, stale string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.jwt != stale {
		return nil // another upload already refreshed it
	}
	jwt, err := u.d.uploadToken(ctx, u.c)
	if err != nil {
		return err
	}
	u.jwt = jwt
	return nil
}

// withToken runs fn with the current JWT and retries once with a fresh one
// when the asset API reports it expired or unauthorised.
func (u *cloudflareUpload) withToken(ctx context.Context, fn func(jwt string) error) error {
	jwt := u.token()
	err := fn(jwt)
	var apiErr *APIError
	if err == nil || !errors.As(err, &apiErr) || (apiErr.Status != 401 && apiErr.Code != cfJWTExpired) {
		return err
	}
	if rerr := u.refresh(ctx, jwt); rerr != nil {
		return rerr
	}
	return fn(u.token())
}

// uploadMissing asks which hashes Cloudflare lacks, uploads those files in
// size-bounded buckets, then registers every hash with the project. It
// returns how many files and bytes were sent.
func (u *cloudflareUpload) uploadMissing(ctx context.Context, files []File, hashes []string) (int, int64, error) {
	u.rep.Step(StepPrepare, "Checking which files Cloudflare already has")
	unique := dedupe(hashes)
	var missing []string
	err := u.withToken(ctx, func(jwt string) error {
		return u.d.call(ctx, u.c, apiRequest{
			Method:      "POST",
			Path:        "/pages/assets/check-missing",
			Endpoint:    "POST /pages/assets/check-missing",
			Token:       jwt,
			ContentType: "application/json",
			Body:        jsonBody(map[string]any{"hashes": unique}),
		}, &missing)
	})
	if err != nil {
		return 0, 0, err
	}

	byHash := make(map[string]File, len(files))
	for i, f := range files {
		if _, ok := byHash[hashes[i]]; !ok {
			byHash[hashes[i]] = f
		}
	}
	var todo []cfAsset
	var bytesTotal int64
	for _, h := range missing {
		f, ok := byHash[h]
		if !ok {
			continue
		}
		todo = append(todo, cfAsset{hash: h, file: f})
		bytesTotal += f.Size
	}
	buckets := bucketAssets(todo)

	u.rep.Step(StepUpload, fmt.Sprintf("Uploading %d of %d files", len(todo), len(files)))
	u.rep.Progress(Progress{Step: StepUpload, Total: len(todo), BytesTotal: bytesTotal})
	var done, sent atomic.Int64
	err = uploadAll(ctx, len(buckets), cfUploadsAtOne, func(ctx context.Context, i int) error {
		b := buckets[i]
		err := u.withToken(ctx, func(jwt string) error {
			return u.d.call(ctx, u.c, apiRequest{
				Method:      "POST",
				Path:        "/pages/assets/upload",
				Endpoint:    "POST /pages/assets/upload",
				Token:       jwt,
				ContentType: "application/json",
				Body:        bucketBody(b),
				Timeout:     uploadTimeout,
			}, nil)
		})
		if err != nil {
			return err
		}
		var size int64
		for _, a := range b {
			size += a.file.Size
		}
		u.rep.Progress(Progress{Step: StepUpload, Done: int(done.Add(int64(len(b)))), Total: len(todo), Bytes: sent.Add(size), BytesTotal: bytesTotal})
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	// Registering the hashes lets later deploys skip these files; a failure
	// only costs a re-upload next time, so it is a warning.
	err = u.withToken(ctx, func(jwt string) error {
		return u.d.call(ctx, u.c, apiRequest{
			Method:      "POST",
			Path:        "/pages/assets/upsert-hashes",
			Endpoint:    "POST /pages/assets/upsert-hashes",
			Token:       jwt,
			ContentType: "application/json",
			Body:        jsonBody(map[string]any{"hashes": unique}),
		}, nil)
	})
	if err != nil {
		if ctx.Err() != nil {
			return 0, 0, ctx.Err()
		}
		u.rep.Log(LevelWarn, "could not register uploaded files with Cloudflare; the next deploy may upload them again: "+err.Error())
	}
	return len(todo), bytesTotal, nil
}

type cfAsset struct {
	hash string
	file File
}

// bucketAssets groups assets, largest first, into batches of at most
// cfBucketBytes of file content and cfBucketFiles files.
func bucketAssets(assets []cfAsset) [][]cfAsset {
	sorted := append([]cfAsset(nil), assets...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].file.Size > sorted[j].file.Size })
	var buckets [][]cfAsset
	var cur []cfAsset
	var curBytes int64
	for _, a := range sorted {
		if len(cur) > 0 && (curBytes+a.file.Size > cfBucketBytes || len(cur) >= cfBucketFiles) {
			buckets = append(buckets, cur)
			cur, curBytes = nil, 0
		}
		cur = append(cur, a)
		curBytes += a.file.Size
	}
	if len(cur) > 0 {
		buckets = append(buckets, cur)
	}
	return buckets
}

// bucketBody streams a bucket as the JSON array the upload endpoint takes,
// base64-encoding each file on the fly so memory stays near one file.
func bucketBody(bucket []cfAsset) bodyFunc {
	return func() (io.ReadCloser, int64, error) {
		pr, pw := io.Pipe()
		go func() {
			pw.CloseWithError(writeBucket(pw, bucket))
		}()
		return pr, -1, nil
	}
}

func writeBucket(w io.Writer, bucket []cfAsset) error {
	if _, err := io.WriteString(w, "["); err != nil {
		return err
	}
	for i, a := range bucket {
		if i > 0 {
			if _, err := io.WriteString(w, ","); err != nil {
				return err
			}
		}
		ct, _ := json.Marshal(contentTypeFor(a.file.Rel))
		if _, err := fmt.Fprintf(w, `{"key":%q,"value":"`, a.hash); err != nil {
			return err
		}
		f, err := os.Open(a.file.Abs)
		if err != nil {
			return err
		}
		enc := base64.NewEncoder(base64.StdEncoding, w)
		_, err = io.Copy(enc, f)
		f.Close()
		if err != nil {
			return err
		}
		if err := enc.Close(); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, `","metadata":{"contentType":%s},"base64":true}`, ct); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "]")
	return err
}

func dedupe(vals []string) []string {
	seen := make(map[string]bool, len(vals))
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
