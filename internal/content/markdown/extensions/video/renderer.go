package video

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
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

	wrapperClass := "sarde-video-wrapper" + ratioClass(v.Ratio)
	iframeAttrs := ` frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen loading="lazy"`

	switch v.Platform {
	case "youtube":
		embedURL := buildYouTubeURL(v.VideoID, v)
		title := resolveTitle(v.Title, "YouTube video")
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-video-embed\"><div class=\"%s\"><iframe src=\"%s\" title=\"%s\"%s></iframe></div>",
			wrapperClass, embedURL, title, iframeAttrs)
		writeCaption(w, v.Title)
		_, _ = w.WriteString("</div>\n")

	case "vimeo":
		embedURL := buildVimeoURL(v.VideoID, v)
		title := resolveTitle(v.Title, "Vimeo video")
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-video-embed\"><div class=\"%s\"><iframe src=\"%s\" title=\"%s\"%s></iframe></div>",
			wrapperClass, embedURL, title, iframeAttrs)
		writeCaption(w, v.Title)
		_, _ = w.WriteString("</div>\n")

	default:
		var videoAttrs strings.Builder
		videoAttrs.WriteString(" controls")
		if v.Autoplay {
			videoAttrs.WriteString(" autoplay")
		}
		if v.Muted {
			videoAttrs.WriteString(" muted")
		}
		if v.Loop {
			videoAttrs.WriteString(" loop")
		}
		if v.Title != "" {
			videoAttrs.WriteString(` title="` + htmlutil.EscapeHTML(v.Title) + `"`)
		}
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-video-embed\"><div class=\"%s\"><video src=\"%s\"%s></video></div>",
			wrapperClass, htmlutil.EscapeHTML(v.URL), videoAttrs.String())
		writeCaption(w, v.Title)
		_, _ = w.WriteString("</div>\n")
	}

	return ast.WalkContinue, nil
}

func buildYouTubeURL(id string, v *VideoBlock) string {
	base := "https://www.youtube-nocookie.com/embed/" + htmlutil.EscapeHTML(id)
	var params []string
	if v.Autoplay {
		params = append(params, "autoplay=1")
	}
	if v.Muted {
		params = append(params, "mute=1")
	}
	if v.Loop {
		params = append(params, "loop=1", "playlist="+htmlutil.EscapeHTML(id))
	}
	if len(params) > 0 {
		return base + "?" + strings.Join(params, "&amp;")
	}
	return base
}

func buildVimeoURL(id string, v *VideoBlock) string {
	base := "https://player.vimeo.com/video/" + htmlutil.EscapeHTML(id)
	var params []string
	if v.Autoplay {
		params = append(params, "autoplay=1")
	}
	if v.Muted {
		params = append(params, "muted=1")
	}
	if v.Loop {
		params = append(params, "loop=1")
	}
	if len(params) > 0 {
		return base + "?" + strings.Join(params, "&amp;")
	}
	return base
}

func ratioClass(ratio string) string {
	switch ratio {
	case "4:3":
		return " sarde-video-4x3"
	case "1:1":
		return " sarde-video-1x1"
	case "9:16":
		return " sarde-video-9x16"
	default:
		return ""
	}
}

func resolveTitle(userTitle, fallback string) string {
	if userTitle != "" {
		return htmlutil.EscapeHTML(userTitle)
	}
	return fallback
}

func writeCaption(w util.BufWriter, title string) {
	if title != "" {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-video-caption\">%s</div>", htmlutil.EscapeHTML(title))
	}
}
