---
title: "Icons"
description: "Inline SVG icons from Lucide, Tabler, and Simple Icons."
sidebar:
  order: 14
---

Inline icons render as SVG elements embedded directly in the page. Three icon sets are bundled: Lucide (general UI icons), Tabler (extended UI icons), and Simple Icons (brand logos).

## Basic Usage

```
:icon[rocket]
```

:icon[rocket]

Lucide is the default set. The icon name maps directly to the Lucide icon name.

## Specifying a Set

Prefix the icon name with the set name and a colon.

```
:icon[lucide:rocket]
:icon[tabler:rocket]
```

:icon[lucide:rocket] :icon[tabler:rocket]

## Brand Icons

Use the `brands:` or `simpleicons:` prefix for brand logos from the Simple Icons set.

```
:icon[brands:github]
:icon[brands:twitter]
:icon[simpleicons:figma]
```

:icon[brands:github] :icon[brands:twitter] :icon[simpleicons:figma]

## Local Icons

Place SVG files in the `icons/` directory at the project root. Reference them by filename without the `.svg` extension.

```
:icon[my-logo]
```

Sarde checks local icons before the bundled sets, so a local file named `rocket.svg` takes precedence over the bundled Lucide `rocket` icon.

## Sizing

Set width and height in pixels or any CSS unit.

```
:icon[rocket width="32" height="32"]
:icon[rocket width="1.5em"]
```

:icon[rocket width="32" height="32"] :icon[rocket width="1.5em"]

When only one dimension is set, the icon scales proportionally.

## Accessibility

Provide a `title` for tooltip text and `aria-label` for screen readers.

```
:icon[github title="GitHub" aria-label="View on GitHub"]
```

:icon[brands:github title="GitHub" aria-label="View on GitHub"]

Icons without a `title` or `aria-label` are treated as decorative and hidden from assistive technology.

## Transform

Rotate or flip an icon without writing CSS.

```
:icon[arrow-right rotate="90"]
:icon[arrow-right rotate="-45"]
:icon[arrow-right flip="horizontal"]
:icon[arrow-right flip="vertical"]
```

:icon[arrow-right rotate="90"] :icon[arrow-right rotate="-45"] :icon[arrow-right flip="horizontal"] :icon[arrow-right flip="vertical"]

## Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| name | string | **Required.** Icon name, with optional `set:` prefix (e.g., `lucide:rocket`, `brands:github`). |
| `width` | string | SVG width. Defaults to `1em`. |
| `height` | string | SVG height. Defaults to `1em`. |
| `title` | string | Tooltip text and accessible name for the icon. |
| `aria-label` | string | Accessible label (alternative to `title`). |
| `rotate` | string | Rotation in degrees, e.g., `"90"` or `"-45"`. |
| `flip` | string | Mirror the icon: `horizontal` or `vertical`. |
