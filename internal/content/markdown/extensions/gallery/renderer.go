package gallery

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
)

var galleryCounter atomic.Int64

type galleryRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &galleryRenderer{} }

func (r *galleryRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindGalleryBlock, r.render)
}

func (r *galleryRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	g := node.(*GalleryBlock)

	// Extract images from child AST nodes (goldmark parses ![alt](src) as Image nodes)
	if len(g.Images) == 0 {
		var walk func(n ast.Node)
		walk = func(n ast.Node) {
			if n.Kind() == ast.KindImage {
				img := n.(*ast.Image)
				alt := extractTextFromNode(n, source)
				g.Images = append(g.Images, GalleryImage{
					Src: string(img.Destination),
					Alt: alt,
				})
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}

	if len(g.Images) == 0 {
		_, _ = w.WriteString(`<div class="sarde-gallery-error">Error: Gallery requires at least one image.</div>`)
		return ast.WalkSkipChildren, nil
	}

	id := galleryCounter.Add(1)
	galleryID := fmt.Sprintf("gallery-%d", id)
	lightboxID := fmt.Sprintf("lightbox-%d", id)

	// Gallery container
	_, _ = fmt.Fprintf(w, "<div class=\"sarde-gallery\" id=\"%s\">\n", galleryID)

	if g.Label != "" {
		_, _ = fmt.Fprintf(w, "<h3 class=\"sarde-gallery-title\">%s</h3>\n", htmlutil.EscapeHTML(g.Label))
	}

	_, _ = w.WriteString("<div class=\"sarde-gallery-grid\">\n")

	for i, img := range g.Images {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-gallery-item\" data-index=\"%d\" role=\"button\" tabindex=\"0\" aria-label=\"View image %d of %d\">\n", i, i+1, len(g.Images))
		_, _ = fmt.Fprintf(w, "<img src=\"%s\" alt=\"%s\" loading=\"lazy\" />\n", htmlutil.EscapeHTML(img.Src), htmlutil.EscapeHTML(img.Alt))
		_, _ = w.WriteString("<div class=\"sarde-gallery-item-overlay\">" + icons.GetWithClass("zoom-in", "sarde-gallery-zoom-icon") + "</div>\n")
		_, _ = w.WriteString("</div>\n")
	}

	_, _ = w.WriteString("</div>\n</div>\n")

	// Lightbox
	_, _ = fmt.Fprintf(w, "<div class=\"sarde-gallery-lightbox\" id=\"%s\" role=\"dialog\" aria-modal=\"true\" aria-label=\"Image gallery\">\n", lightboxID)
	_, _ = w.WriteString("<button class=\"sarde-lightbox-close\" aria-label=\"Close lightbox\">&times;</button>\n")
	_, _ = w.WriteString("<button class=\"sarde-lightbox-prev\" aria-label=\"Previous image\">&lsaquo;</button>\n")
	_, _ = w.WriteString("<button class=\"sarde-lightbox-next\" aria-label=\"Next image\">&rsaquo;</button>\n")
	_, _ = w.WriteString("<div class=\"sarde-lightbox-content\">\n")
	_, _ = w.WriteString("<img class=\"sarde-lightbox-image\" src=\"\" alt=\"\" />\n")
	_, _ = w.WriteString("<div class=\"sarde-lightbox-caption\"></div>\n")
	_, _ = w.WriteString("<div class=\"sarde-lightbox-counter\"></div>\n")
	_, _ = w.WriteString("</div>\n</div>\n")

	// Inline script for lightbox
	_, _ = fmt.Fprintf(w, `<script>
(function(){
  var gallery = document.getElementById('%s');
  var lightbox = document.getElementById('%s');
  if (!gallery || !lightbox) return;
  var items = gallery.querySelectorAll('.sarde-gallery-item');
  var lbImg = lightbox.querySelector('.sarde-lightbox-image');
  var lbCaption = lightbox.querySelector('.sarde-lightbox-caption');
  var lbCounter = lightbox.querySelector('.sarde-lightbox-counter');
  var images = %s;
  var current = 0;
  function show(idx) {
    current = idx;
    lbImg.src = images[idx].src;
    lbImg.alt = images[idx].alt;
    lbCaption.textContent = images[idx].alt;
    lbCounter.textContent = (idx+1) + ' / ' + images.length;
    lightbox.classList.add('active');
    document.body.style.overflow = 'hidden';
    lightbox.querySelector('.sarde-lightbox-close').focus();
  }
  function hide() {
    lightbox.classList.remove('active');
    document.body.style.overflow = '';
    if (trigger) { trigger.focus(); trigger = null; }
  }
  var trigger = null;
  var focusable = lightbox.querySelectorAll('button');
  items.forEach(function(item) {
    item.addEventListener('click', function() { trigger = item; show(parseInt(item.dataset.index)); });
    item.addEventListener('keydown', function(e) { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); trigger = item; show(parseInt(item.dataset.index)); } });
  });
  lightbox.querySelector('.sarde-lightbox-close').addEventListener('click', hide);
  lightbox.querySelector('.sarde-lightbox-prev').addEventListener('click', function() { show((current - 1 + images.length) %% images.length); });
  lightbox.querySelector('.sarde-lightbox-next').addEventListener('click', function() { show((current + 1) %% images.length); });
  lightbox.addEventListener('click', function(e) { if (e.target === lightbox) hide(); });
  document.addEventListener('keydown', function(e) {
    if (!lightbox.classList.contains('active')) return;
    if (e.key === 'Escape') hide();
    if (e.key === 'ArrowLeft') show((current - 1 + images.length) %% images.length);
    if (e.key === 'ArrowRight') show((current + 1) %% images.length);
    if (e.key === 'Tab') { e.preventDefault(); var idx = Array.prototype.indexOf.call(focusable, document.activeElement); focusable[(idx + (e.shiftKey ? -1 + focusable.length : 1)) %% focusable.length].focus(); }
  });
})();
</script>`, galleryID, lightboxID, buildImageJSON(g.Images))

	_, _ = w.WriteString("\n")

	return ast.WalkSkipChildren, nil
}

func buildImageJSON(images []GalleryImage) string {
	var parts []string
	for _, img := range images {
		parts = append(parts, fmt.Sprintf(`{"src":"%s","alt":"%s"}`, escapeJSString(img.Src), escapeJSString(img.Alt)))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func escapeJSString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func extractTextFromNode(n ast.Node, source []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == ast.KindText {
			t := c.(*ast.Text)
			sb.Write(t.Segment.Value(source))
		}
	}
	return sb.String()
}

