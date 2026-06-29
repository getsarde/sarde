package video

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func render(t *testing.T, src string) string {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		t.Fatalf("convert %q: %v", src, err)
	}
	return buf.String()
}

func TestYouTubeWatchURL(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=dQw4w9WgXcQ\"}\n:::")
	if !strings.Contains(out, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Errorf("expected nocookie embed URL: %q", out)
	}
}

func TestYouTubeShortURL(t *testing.T) {
	out := render(t, ":::video{src=\"https://youtu.be/dQw4w9WgXcQ\"}\n:::")
	if !strings.Contains(out, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Errorf("expected nocookie embed URL: %q", out)
	}
}

func TestYouTubeEmbedURL(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/embed/dQw4w9WgXcQ\"}\n:::")
	if !strings.Contains(out, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Errorf("expected nocookie embed URL: %q", out)
	}
}

func TestYouTubeShortsURL(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/shorts/dQw4w9WgXcQ\"}\n:::")
	if !strings.Contains(out, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Errorf("expected nocookie embed URL: %q", out)
	}
}

func TestNocookieDomain(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc123\"}\n:::")
	if strings.Contains(out, "youtube.com/embed") && !strings.Contains(out, "youtube-nocookie.com") {
		t.Errorf("should use nocookie domain: %q", out)
	}
}

func TestVimeoBasic(t *testing.T) {
	out := render(t, ":::video{src=\"https://vimeo.com/148751763\"}\n:::")
	if !strings.Contains(out, "player.vimeo.com/video/148751763") {
		t.Errorf("expected vimeo embed: %q", out)
	}
}

func TestNativeVideoFallback(t *testing.T) {
	out := render(t, ":::video{src=\"video.mp4\"}\n:::")
	if !strings.Contains(out, "<video") || !strings.Contains(out, "controls") {
		t.Errorf("expected native video with controls: %q", out)
	}
	if !strings.Contains(out, `src="video.mp4"`) {
		t.Errorf("expected src attribute: %q", out)
	}
}

func TestYouTubeAutoplayMutedLoop(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\" autoplay muted loop}\n:::")
	if !strings.Contains(out, "autoplay=1") {
		t.Errorf("expected autoplay param: %q", out)
	}
	if !strings.Contains(out, "mute=1") {
		t.Errorf("expected mute=1 (not muted=1) for YouTube: %q", out)
	}
	if !strings.Contains(out, "loop=1") {
		t.Errorf("expected loop param: %q", out)
	}
	if !strings.Contains(out, "playlist=abc") {
		t.Errorf("expected playlist param for YouTube loop: %q", out)
	}
}

func TestVimeoAutoplayMutedLoop(t *testing.T) {
	out := render(t, ":::video{src=\"https://vimeo.com/123\" autoplay muted loop}\n:::")
	if !strings.Contains(out, "autoplay=1") {
		t.Errorf("expected autoplay: %q", out)
	}
	if !strings.Contains(out, "muted=1") {
		t.Errorf("expected muted=1 (not mute=1) for Vimeo: %q", out)
	}
	if !strings.Contains(out, "loop=1") {
		t.Errorf("expected loop: %q", out)
	}
}

func TestNativeVideoAttributes(t *testing.T) {
	out := render(t, ":::video{src=\"demo.mp4\" autoplay muted loop}\n:::")
	if !strings.Contains(out, " autoplay") {
		t.Errorf("expected autoplay attr: %q", out)
	}
	if !strings.Contains(out, " muted") {
		t.Errorf("expected muted attr: %q", out)
	}
	if !strings.Contains(out, " loop") {
		t.Errorf("expected loop attr: %q", out)
	}
	if !strings.Contains(out, " controls") {
		t.Errorf("controls should always be present: %q", out)
	}
}

func TestTitleOverrideYouTube(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\" title=\"My Video\"}\n:::")
	if !strings.Contains(out, `title="My Video"`) {
		t.Errorf("expected custom title: %q", out)
	}
	if strings.Contains(out, "YouTube video") {
		t.Errorf("should not contain fallback title: %q", out)
	}
}

func TestTitleFallback(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\"}\n:::")
	if !strings.Contains(out, `title="YouTube video"`) {
		t.Errorf("expected fallback title: %q", out)
	}
}

func TestCaptionRendered(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\" title=\"My Caption\"}\n:::")
	if !strings.Contains(out, `<div class="sarde-video-caption">My Caption</div>`) {
		t.Errorf("expected caption div: %q", out)
	}
}

func TestCaptionAbsentWithoutTitle(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\"}\n:::")
	if strings.Contains(out, "sarde-video-caption") {
		t.Errorf("should not have caption without title: %q", out)
	}
}

func TestRatio4x3(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\" ratio=\"4:3\"}\n:::")
	if !strings.Contains(out, "sarde-video-4x3") {
		t.Errorf("expected 4x3 class: %q", out)
	}
}

func TestRatio1x1(t *testing.T) {
	out := render(t, ":::video{src=\"https://vimeo.com/123\" ratio=\"1:1\"}\n:::")
	if !strings.Contains(out, "sarde-video-1x1") {
		t.Errorf("expected 1x1 class: %q", out)
	}
}

func TestRatio9x16(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/shorts/abc\" ratio=\"9:16\"}\n:::")
	if !strings.Contains(out, "sarde-video-9x16") {
		t.Errorf("expected 9x16 class: %q", out)
	}
}

func TestDefaultRatioNoExtraClass(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\"}\n:::")
	if strings.Contains(out, "sarde-video-4x3") || strings.Contains(out, "sarde-video-1x1") || strings.Contains(out, "sarde-video-9x16") {
		t.Errorf("default ratio should have no extra class: %q", out)
	}
	if !strings.Contains(out, `class="sarde-video-wrapper"`) {
		t.Errorf("expected plain wrapper class: %q", out)
	}
}

func TestMissingSrcReturnsNothing(t *testing.T) {
	out := render(t, ":::video{title=\"No source\"}\n:::")
	if strings.Contains(out, "sarde-video") {
		t.Errorf("missing src should not render video: %q", out)
	}
}

func TestHTMLEscapeTitle(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=abc\" title=\"A <script> test\"}\n:::")
	if strings.Contains(out, "<script>") {
		t.Errorf("title should be HTML escaped: %q", out)
	}
}

func TestBackwardCompat(t *testing.T) {
	out := render(t, ":::video{src=\"https://www.youtube.com/watch?v=dQw4w9WgXcQ\"}\n:::")
	if !strings.Contains(out, "sarde-video-embed") {
		t.Errorf("expected video embed: %q", out)
	}
	if !strings.Contains(out, "<iframe") {
		t.Errorf("expected iframe: %q", out)
	}
}
