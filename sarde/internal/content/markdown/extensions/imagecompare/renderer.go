package imagecompare

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

var compareCounter atomic.Int64

type imageCompareRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &imageCompareRenderer{} }

func (r *imageCompareRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindImageCompareBlock, r.render)
}

func (r *imageCompareRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	ic := node.(*ImageCompareBlock)

	// Extract images from child AST nodes if not already set
	if !ic.HasImages() {
		var walk func(n ast.Node)
		walk = func(n ast.Node) {
			if n.Kind() == ast.KindImage {
				img := n.(*ast.Image)
				alt := extractTextFromNode(n, source)
				if ic.BeforeSrc == "" {
					ic.BeforeSrc = string(img.Destination)
					ic.BeforeAlt = alt
				} else if ic.AfterSrc == "" {
					ic.AfterSrc = string(img.Destination)
					ic.AfterAlt = alt
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}

	if !ic.HasImages() {
		_, _ = w.WriteString(`<div class="sarde-image-compare-error">Error: Image compare requires two images (before and after).</div>`)
		return ast.WalkSkipChildren, nil
	}

	id := compareCounter.Add(1)
	containerID := fmt.Sprintf("img-compare-%d", id)

	_, _ = w.WriteString("<div class=\"sarde-image-compare-wrapper\">\n")
	if ic.Label != "" {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-image-compare-label\">%s</div>\n", htmlutil.EscapeHTML(ic.Label))
	}

	_, _ = fmt.Fprintf(w, "<div class=\"sarde-image-compare-container\" id=\"%s\" data-position=\"50\">\n", containerID)

	// Before image
	_, _ = fmt.Fprintf(w, "<div class=\"sarde-image-compare-before\"><img src=\"%s\" alt=\"%s\" draggable=\"false\" /></div>\n",
		htmlutil.EscapeHTML(ic.BeforeSrc), htmlutil.EscapeHTML(ic.BeforeAlt))

	// After image
	_, _ = fmt.Fprintf(w, "<div class=\"sarde-image-compare-after\"><img src=\"%s\" alt=\"%s\" draggable=\"false\" /></div>\n",
		htmlutil.EscapeHTML(ic.AfterSrc), htmlutil.EscapeHTML(ic.AfterAlt))

	// Handle
	_, _ = w.WriteString("<div class=\"sarde-image-compare-handle\">\n")
	_, _ = w.WriteString("<div class=\"sarde-image-compare-handle-line\"></div>\n")
	_, _ = w.WriteString("<div class=\"sarde-image-compare-handle-button\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"24\" height=\"24\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\"><polyline points=\"15 18 9 12 15 6\"/><polyline points=\"9 18 15 12 9 6\" transform=\"translate(6,0)\"/></svg></div>\n")
	_, _ = w.WriteString("</div>\n")

	// Labels
	_, _ = fmt.Fprintf(w, "<div class=\"sarde-image-compare-label-before\">%s</div>\n", htmlutil.EscapeHTML(ic.BeforeAlt))
	_, _ = fmt.Fprintf(w, "<div class=\"sarde-image-compare-label-after\">%s</div>\n", htmlutil.EscapeHTML(ic.AfterAlt))

	_, _ = w.WriteString("</div>\n</div>\n")

	// Inline script for slider
	_, _ = fmt.Fprintf(w, `<script>
(function(){
  var container = document.getElementById('%s');
  if (!container) return;
  var handle = container.querySelector('.sarde-image-compare-handle');
  var after = container.querySelector('.sarde-image-compare-after');
  var dragging = false;
  function update(x) {
    var rect = container.getBoundingClientRect();
    var pos = Math.max(0, Math.min(100, ((x - rect.left) / rect.width) * 100));
    container.dataset.position = pos;
    handle.style.left = pos + '%%';
    after.style.clipPath = 'inset(0 0 0 ' + pos + '%%)';
  }
  container.addEventListener('mousedown', function(e) { dragging = true; update(e.clientX); e.preventDefault(); });
  document.addEventListener('mousemove', function(e) { if (dragging) update(e.clientX); });
  document.addEventListener('mouseup', function() { dragging = false; });
  container.addEventListener('touchstart', function(e) { dragging = true; update(e.touches[0].clientX); e.preventDefault(); }, {passive:false});
  document.addEventListener('touchmove', function(e) { if (dragging) update(e.touches[0].clientX); });
  document.addEventListener('touchend', function() { dragging = false; });
  update(container.getBoundingClientRect().left + container.getBoundingClientRect().width * 0.5);
})();
</script>`, containerID)

	_, _ = w.WriteString("\n")

	return ast.WalkSkipChildren, nil
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

