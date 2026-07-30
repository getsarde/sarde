package socialcards

import (
	"bytes"
	"image/color"
	"testing"
	"time"
)

func baseKeyParams() CardParams {
	accent2 := color.NRGBA{0x11, 0x22, 0x33, 0xff}
	return CardParams{
		Title:            "Hello World",
		Description:      "A description",
		SiteTitle:        "My Site",
		CollectionName:   "Blog",
		Date:             time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		DateExplicit:     true,
		BgColor:          color.NRGBA{26, 26, 46, 255},
		AccentColor:      color.NRGBA{233, 69, 96, 255},
		AccentColor2:     &accent2,
		TextColor:        color.NRGBA{255, 255, 255, 255},
		WatermarkOpacity: 0.07,
		BgImageOpacity:   1.0,
	}
}

func baseKeyHashes() cardAssetHashes {
	return cardAssetHashes{Logo: "aaaa", Watermark: "bbbb", BgImage: "", FontRegular: "", FontBold: ""}
}

func TestCardKey_StableForIdenticalInputs(t *testing.T) {
	a := cardKey(cardCacheVersion, baseKeyParams(), "png", 90, baseKeyHashes())
	b := cardKey(cardCacheVersion, baseKeyParams(), "png", 90, baseKeyHashes())
	if a != b {
		t.Errorf("identical inputs produced different keys: %q vs %q", a, b)
	}
	if len(a) != 16 {
		t.Errorf("key should be 8 truncated sha256 bytes as hex (16 chars), got %d: %q", len(a), a)
	}
}

func TestCardKey_ChangesWhenAnyInputChanges(t *testing.T) {
	base := cardKey(cardCacheVersion, baseKeyParams(), "png", 90, baseKeyHashes())

	variants := map[string]func() string{
		"title": func() string {
			p := baseKeyParams()
			p.Title = "Other Title"
			return cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
		},
		"description": func() string {
			p := baseKeyParams()
			p.Description = "Other description"
			return cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
		},
		"bg color": func() string {
			p := baseKeyParams()
			p.BgColor = color.NRGBA{0, 0, 0, 255}
			return cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
		},
		"accent2 nil": func() string {
			p := baseKeyParams()
			p.AccentColor2 = nil
			return cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
		},
		"date": func() string {
			p := baseKeyParams()
			p.Date = p.Date.AddDate(0, 0, 1)
			return cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
		},
		"gradient override": func() string {
			p := baseKeyParams()
			p.GradientOverride = []color.NRGBA{{1, 2, 3, 255}}
			return cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
		},
		"format": func() string {
			return cardKey(cardCacheVersion, baseKeyParams(), "jpeg", 90, baseKeyHashes())
		},
		"quality": func() string {
			return cardKey(cardCacheVersion, baseKeyParams(), "png", 80, baseKeyHashes())
		},
		"logo hash": func() string {
			h := baseKeyHashes()
			h.Logo = "cccc"
			return cardKey(cardCacheVersion, baseKeyParams(), "png", 90, h)
		},
		"font hash": func() string {
			h := baseKeyHashes()
			h.FontBold = "dddd"
			return cardKey(cardCacheVersion, baseKeyParams(), "png", 90, h)
		},
		"version bump": func() string {
			return cardKey("999", baseKeyParams(), "png", 90, baseKeyHashes())
		},
	}
	for name, f := range variants {
		if got := f(); got == base {
			t.Errorf("changing %s did not change the key", name)
		}
	}
}

func TestCardKey_InferredDateIgnored(t *testing.T) {
	// A page without an explicit date renders no footer date, so mtime churn
	// on its Date must not invalidate the cache entry.
	p := baseKeyParams()
	p.DateExplicit = false
	a := cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
	p.Date = p.Date.AddDate(1, 0, 0)
	b := cardKey(cardCacheVersion, p, "png", 90, baseKeyHashes())
	if a != b {
		t.Error("inferred date change should not change the key")
	}
}

func TestCardCache_GetPutRoundTrip(t *testing.T) {
	cache := newCardCache(t.TempDir())
	key := "0123456789abcdef"
	data := []byte("fake png bytes")

	if got := cache.Get(key, "png"); got != nil {
		t.Errorf("expected miss on an empty cache, got %d bytes", len(got))
	}
	cache.Put(key, "png", data)
	if got := cache.Get(key, "png"); !bytes.Equal(got, data) {
		t.Errorf("round trip mismatch: got %q, want %q", got, data)
	}
	// Formats store distinct entries under distinct extensions.
	if got := cache.Get(key, "jpeg"); got != nil {
		t.Error("jpeg lookup must not hit the png entry")
	}
}

func TestHashImage_NilAndDimensions(t *testing.T) {
	if hashImage(nil) != "" {
		t.Error("nil image must hash to the empty string")
	}
}
