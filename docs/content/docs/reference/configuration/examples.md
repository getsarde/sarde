---
title: Examples
description: "Complete sarde.yaml files for common kinds of site, and every key with its default value"
sidebar:
  order: 1
---

Copy the example closest to your site and change the values. Every key you leave out uses its default, so a working `sarde.yaml` only needs the keys that differ from the defaults listed [at the end of this page](#every-key-with-its-default).

`sarde new` writes a starter `sarde.yaml` with a site title, theme, and homepage hero. See [`new`](/reference/cli-commands#new) in CLI Commands.

## Minimal site

A site title, description, and production URL are enough to build and publish:

```yaml title="sarde.yaml"
site:
  title: Chemistry Lab Notes
  description: Lab procedures and safety notes for CHEM 110
  url: https://chem110.example.org
```

→ Sarde builds the site with the default theme, search, sitemap, feeds, and SEO tags. Collections come from the directory names under `content/`.

## Documentation site

A documentation site published to GitHub Pages under the repository name, with a docs-oriented theme preset and a collapsible sidebar:

```yaml title="sarde.yaml"
site:
  title: Research Computing Guide
  description: How to use the university cluster, storage, and software modules
  url: https://research-it.github.io
  edit_url: https://github.com/research-it/guide/edit/main/content
  logo: /images/logo.svg
build:
  base_path: /guide/
theme:
  preset: docs
markdown:
  asides:
    style: galaxy
  codeblocks:
    light_theme: github-light
    dark_theme: github-dark-default
collections:
  docs:
    sidebar:
      collapsible: true
      collapsed_by_default: true
social:
  - label: GitHub
    url: https://github.com/research-it/guide
    icon: github
deploy:
  provider: github
```

→ Pages under `content/docs/` get the docs layout with a sidebar whose groups start collapsed, and every page shows an "Edit this page" link. `sarde deploy` pushes the build to the `gh-pages` branch. See [GitHub Pages](/deployment/github-pages/).

## Blog

A blog with tags and categories, ten posts per list page, and a contact link in the header:

```yaml title="sarde.yaml"
site:
  title: Field Notes
  description: Weekly notes from a high school science classroom
  url: https://fieldnotes.example.com
social:
  - label: Contact
    url: https://fieldnotes.example.com/contact/
    icon: mail
collections:
  blog:
    paginate: 10
taxonomies:
  tags: tag
  categories: category
```

→ Posts under `content/blog/` are sorted newest first, with RSS and Atom feeds. Each tag and category gets a listing page.

## Course site

A course with lessons, numbered labs, and a homepage that links to the first lesson:

```yaml title="sarde.yaml"
site:
  title: Introduction to Biology
  description: Lessons, labs, and slides for BIO 101
  url: https://bio101.example.org
theme:
  preset: academic
collections:
  courses:
    sidebar:
      collapsible: true
  labs:
    labs:
      label: Exercise
homepage:
  hero:
    title: Introduction to Biology
    subtitle: Twelve weeks of lessons, labs, and review slides
    cta:
      label: Start the course
      url: /courses/week-1/
```

→ `content/courses/` gets the docs layout ordered by weight, and each page under `content/labs/` shows an "Exercise" badge with its number. KaTeX math and Mermaid diagrams are on by default.

## Every key with its default

This file sets every top-level key to its default value. Setting a key to the value shown here changes nothing, so copy only the keys you want to change. Keys whose default is empty (`""`, `[]`, or `{}`) are documented on the page named in each comment.

```yaml title="sarde.yaml"
# Site and Branding: /reference/configuration/site-and-branding/
site:
  title: "My Site"
  description: ""
  url: ""
  language: "en"
  logo:
    light: ""
    dark: ""
    alt: ""
    replaces_title: false
  favicon: ""
  edit_url: ""
  title_delimiter: "|"
  heading_links: true

social: []

header:
  search: true
  theme_toggle: true
  social: true
  links: []

footer:
  text: ""
  links: []
  credits: true

head:
  tags: []
  custom_css: []
  custom_js: []

homepage:
  template: "hero"
  hero:
    title: ""
    subtitle: ""
    background: "gradient"

# Theme and Appearance: /reference/configuration/theme-and-appearance/
theme:
  name: "default"
  preset: ""
  dark: true
  overrides: {}
  date_format: "short"

toc:
  enabled: true
  min_level: 2
  max_level: 4

icons:
  default_prefix: "lucide"
  sets: []
  sets_dir: ""
  local_dir: "icons"
  attribution: ""
  render: "inline"

prefetch:
  enabled: true
  strategy: "hover"
  delay: 300

# Content: /reference/configuration/content/
content:
  dir: "content"
  summary_length: 70

collections: {}

taxonomies:
  tags: "tag"

permalinks: {}

markdown:
  katex: true
  mermaid: true
  cdn: false
  unsafe: false
  typographer: true
  github_alerts: true
  triple_colon_callouts: true
  hard_wraps: false
  toc:
    min_heading_level: 2
    max_heading_level: 4
  asides:
    style: "classic"
  codeblocks:
    style: "class"
    light_theme: "github-light"
    dark_theme: "github-dark"
    theme: ""
    dark_mode_selector: "[data-theme=\"dark\"]"

i18n:
  default_language: "en"
  languages: {}

# Build and Output: /reference/configuration/build-and-output/
build:
  output: "dist"
  base_path: ""
  clean: true
  sitemap: true
  minify: false
  last_updated: git
  feed: true
  drafts: false
  future: false
  parallel: true

images:
  widths: [400, 800, 1200]
  formats: ["webp"]
  quality: 80
  placeholder: "lqip"
  max_width: 2400
  lazy_loading: true
  dimensions: true

search:
  enabled: true
  provider: "orama"

analytics:
  provider: ""
  site_id: ""

llms_txt:
  enabled: true
  include_blog: true

security:
  blocked_href_schemes:
    - "javascript:"
    - "data:"
    - "vbscript:"

server:
  port: 4727
  live_reload: true

# Plugins and Checks: /reference/configuration/plugins-and-checks/
plugins:
  enabled:
    - search
    - seo
    - sitemap
    - robots
    - rss
    - atom
    - content_lint
    - link_validator
    - redirects
    - llms_txt
    - katex
    - mermaid
    - social_cards
  config: {}

link_validation:
  enabled: true
  level: "warn"
  on_broken: "error"
  on_broken_anchor: "error"
  report: "pretty"
  on_relative_links: "warn"
  on_local_links: "warn"
  on_unverified_internal: "warn"
  check_anchors: true
  check_images: true
  same_site_policy: "ignore"
  site_root_escape_prefix: "site:"
  exclude: []
  fail_build: false
  external:
    check: false
    concurrency: 8
    timeout: "10s"
    cache: ".sarde/linkcache.json"
    cache_ttl: "72h"
    on_broken: "warn"
    ignore: []
    method: "head-then-get"

content_lint:
  enabled: true
  rules:
    heading_max_length: 60
    heading_increment: true
    image_alt_required: true
    no_empty_links: true
    frontmatter_required: []
    tabs_marker_syntax: true

# Deploy and Redirects: /reference/configuration/deploy-and-redirects/
deploy:
  provider: ""
  branch: ""
  cname: ""
  site_id: ""
  account_id: ""
  project_name: ""
  project_id: ""
  team_id: ""
  command: ""
  redirect_format: ""

redirects: {}
```

A test in the Sarde repository compares this block with the defaults compiled into Sarde, so the values match the release these docs describe.
