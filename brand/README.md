# Sarde brand assets

Master copies of the Sarde logo, favicon, wordmarks, and raster exports in the Lebanese Red
identity. This folder is the source of truth. Files served by the documentation site and the
assets embedded in the binary are copies, because Sarde only serves static assets from
`docs/public/` and the social-card plugin embeds its artwork at compile time.

Copying is manual. After editing a master here, copy it to every destination listed below.

The primary identity is the centred woven-narrative mark. Its red ribbon spans `x 41-215` and
`y 37.5-218.5`, centring the artwork at `(128, 128)` on the 256 px canvas.

## SVG masters

- `icon.svg`: primary woven application and brand icon, intended for 48 px and above
- `icon-small.svg`: matching simplified ribbon for 16 and 32 px use
- `icon-detail.svg`: compatibility copy of the detailed woven mark
- `icon-bare.svg`: complete woven red mark without a background tile
- `monochrome.svg`: complete one-colour woven mark using `currentColor`
- `wordmark-light.svg` and `wordmark-dark.svg`: horizontal woven lockups

## Social card masters

- `social-card-mark.png`: the corner mark drawn on generated social cards. A copy of
  `png/icon-128.png`.
- `social-card-watermark.png`: the large low-opacity watermark on social cards. A 1024 px
  rasterization of the bare woven mark; on the navy card background its navy weave bars recede,
  leaving the woven red ribbon.
- `social-card-watermark-4x.svg`: the rasterizer input for the watermark. It is `icon-bare.svg`
  with every coordinate and stroke width multiplied by 4, because the offline rasterizer
  (oksvg) does not scale stroke widths or rounded rectangle corners when scaling. Edit
  `icon-bare.svg` first, then re-bake this file, then re-rasterize.

## Files and their destinations

| Master | Copy to | Used by |
|--------|---------|---------|
| `icon.svg` | `docs/public/images/sarde-icon.svg` | Docs site header logo (`site.logo`) and homepage hero image (`homepage.hero.image`) in `docs/sarde.yaml` |
| `favicon.ico` | `docs/public/favicon.ico` | Docs site favicon, picked up by Sarde's favicon auto-detection |
| `social-card-mark.png` | `internal/plugin/socialcards/assets/logo/sarde-mark.png` | Social card corner mark when a site sets `social_cards.logo: "sarde"` |
| `social-card-watermark.png` | `internal/plugin/socialcards/assets/logo/sarde-ribbon.png` | Social card watermark for `logo: "sarde"` |
| `wordmark-light.svg`, `wordmark-dark.svg` | no copy yet | Available for README headers and future site use |

After changing either social-card master, rebuild the binary so the embedded copies update:
the PNGs live under `//go:embed all:assets` in `internal/plugin/socialcards`.

## Raster exports

- `png/icon-16.png` and `png/icon-32.png` use `icon-small.svg`
- `png/icon-{48,128,256,512,1024}.png` use `icon.svg`
- `favicon.ico` uses the simplified mark at 16/32 px and the woven mark at larger sizes
- `png/wordmark-light.png` and `png/wordmark-dark.png` are rendered at 192 DPI

## Monochrome usage

Use `monochrome.svg` when the medium or surrounding design restricts the identity to exactly one
colour. Appropriate applications include:

- laser engraving, etching, embossing, and debossing
- rubber stamps and one-ink printing
- single-colour screen printing, vinyl cutting, and embroidery
- sponsor or partner logo walls where every logo shares one colour
- watermarks and restrained website footer or navigation treatments
- inline SVG icons or CSS masks that follow the surrounding interface colour

Do not use the monochrome mark for the primary application icon, website hero branding, social
profile images, store listings, or favicons when the full Lebanese Red identity is available.
Prefer the full-colour woven icon whenever the medium permits it.

The mark uses `currentColor`. It inherits CSS colour when embedded inline:

```html
<div style="color: #111827">
  <!-- Inline the contents of monochrome.svg here. -->
</div>
```

When loaded through `<img src="monochrome.svg">`, the SVG normally renders black and does not
inherit the parent element's CSS `color`. Avoid the detailed monochrome mark at 16 px because its
terminal cutouts become too small; use `icon-small.svg` for tiny full-colour placements.

## Palette

| Colour | Hex | Role |
|--------|-----|------|
| Lebanese Red | `#E43D3D` | Ribbon, brand accent |
| Deep Navy | `#111827` | Icon tile, social card background |
| Warm Ivory | `#F4F1EA` | Narrative bars, light text on navy |
