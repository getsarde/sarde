---
title: Video
description: "Embed YouTube, Vimeo, or self-hosted video files with a responsive wrapper"
sidebar:
  order: 27
  group: Block
---

Video embeds YouTube, Vimeo, or self-hosted video files with a responsive wrapper. YouTube and Vimeo URLs are auto-detected and rendered as privacy-friendly iframes.

## Basic syntax

````
:::video(src="https://www.youtube.com/watch?v=dQw4w9WgXcQ")
:::
````

→ A responsive YouTube embed appears in a 16:9 wrapper. The iframe uses the `youtube-nocookie.com` domain for privacy.

## Vimeo

````
:::video(src="https://vimeo.com/76979871")
:::
````

→ A responsive Vimeo embed appears using the `player.vimeo.com` iframe.

## Self-hosted video

For non-YouTube/Vimeo URLs, a native `<video>` element renders with playback controls:

````
:::video(src="/media/lecture-recording.mp4")
:::
````

→ A video player appears with standard browser controls (play, pause, seek, volume).

## Title and caption

Add a `title` attribute to set the iframe's accessible title and display a caption below the video:

````
:::video(src="https://www.youtube.com/watch?v=dQw4w9WgXcQ" title="Introduction to molecular biology")
:::
````

→ The iframe has `title="Introduction to molecular biology"` for screen readers, and the caption text appears below the video.

Without a `title`, YouTube embeds use "YouTube video" and Vimeo embeds use "Vimeo video" as the default iframe title.

## Aspect ratio

Set the `ratio` attribute to change the video container proportions:

````
:::video(src="https://www.youtube.com/watch?v=dQw4w9WgXcQ" ratio="4:3")
:::
````

| Ratio | Description |
|-------|-------------|
| (default) | 16:9 widescreen. |
| `4:3` | Standard/legacy aspect ratio. |
| `1:1` | Square. |
| `9:16` | Vertical/portrait. |

Unrecognized ratio values are ignored (16:9 default applies).

## Playback options

Control autoplay, mute, and loop with bare flags:

````
:::video(src="https://www.youtube.com/watch?v=dQw4w9WgXcQ" autoplay muted loop)
:::
````

| Option | Description |
|--------|-------------|
| `autoplay` | Start playback automatically. |
| `muted` | Start with audio muted. Most browsers require `muted` for autoplay to work. |
| `loop` | Restart the video when it reaches the end. |

For YouTube, these translate to URL parameters (`autoplay=1`, `mute=1`, `loop=1`). For Vimeo, they become `autoplay=1`, `muted=1`, `loop=1`. For self-hosted video, they become HTML attributes on the `<video>` element.

## Supported URL formats

YouTube URLs are recognized in these formats:
- `youtube.com/watch?v=ID`
- `youtu.be/ID`
- `youtube.com/embed/ID`
- `youtube.com/shorts/ID`

Vimeo URLs use the `vimeo.com/ID` format.

Any other URL is treated as a direct video file and rendered with a `<video>` element.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| `src` | `src="URL"` | — | Video URL. Required. |
| `title` | `title="text"` | Platform default | Accessible title and visible caption. |
| `ratio` | `ratio="4:3"` | 16:9 | Aspect ratio: `4:3`, `1:1`, `9:16`. |
| `autoplay` | `autoplay` | `false` | Start playback automatically. |
| `muted` | `muted` | `false` | Start with audio muted. |
| `loop` | `loop` | `false` | Loop playback. |

## Edge cases

- A video block without `src` is silently ignored.
- YouTube embeds use `youtube-nocookie.com` for enhanced privacy (no tracking cookies until the viewer plays).
- The iframe includes `loading="lazy"` for deferred loading below the fold.
- For YouTube loops, the `playlist` parameter is automatically set to the video ID (required by the YouTube embed API for looping to work).
