package video

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type videoRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &videoRenderer{} }

func (r *videoRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindVideoBlock, r.render)
}

func (r *videoRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	v := node.(*VideoBlock)
	raw := v.URL

	iframeAttrs := ` frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen loading="lazy"`
	if id, ok := extractYouTubeID(raw); ok {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-video-embed\"><div class=\"sarde-video-wrapper\"><iframe src=\"https://www.youtube.com/embed/%s\"%s></iframe></div></div>\n", htmlutil.EscapeHTML(id), iframeAttrs)
	} else if id, ok := extractVimeoID(raw); ok {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-video-embed\"><div class=\"sarde-video-wrapper\"><iframe src=\"https://player.vimeo.com/video/%s\"%s></iframe></div></div>\n", htmlutil.EscapeHTML(id), iframeAttrs)
	} else {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-video-embed\"><div class=\"sarde-video-wrapper\"><video src=\"%s\" controls></video></div></div>\n", htmlutil.EscapeHTML(raw))
	}
	return ast.WalkContinue, nil
}

func extractYouTubeID(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Host)
	if host == "youtu.be" || host == "www.youtu.be" {
		id := strings.TrimPrefix(u.Path, "/")
		if id != "" {
			return id, true
		}
	}
	if host == "youtube.com" || host == "www.youtube.com" || host == "m.youtube.com" {
		if v := u.Query().Get("v"); v != "" {
			return v, true
		}
	}
	return "", false
}

func extractVimeoID(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Host)
	if host == "vimeo.com" || host == "www.vimeo.com" {
		id := strings.TrimPrefix(u.Path, "/")
		if id != "" && !strings.Contains(id, "/") {
			return id, true
		}
	}
	return "", false
}

