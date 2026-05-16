/**
 * Map a content-relative file path to a site URL path.
 *
 * Examples:
 *   "docs/getting-started.md"  → "/docs/getting-started/"
 *   "blog/01-hello.md"         → "/blog/hello/"
 *   "_index.md"                → "/"
 *   "docs/_index.md"           → "/docs/"
 *   "docs/intro/index.md"      → "/docs/intro/"
 */
export function contentPathToUrl(contentPath) {
  if (!contentPath) return '/'

  let url = contentPath
    .replace(/\\/g, '/')
    .replace(/\.md$/, '')
    .replace(/\/_index$/, '')
    .replace(/\/index$/, '')

  // Strip numeric prefix from each path segment: "01-hello" → "hello"
  url = url
    .split('/')
    .map(seg => seg.replace(/^\d+-/, ''))
    .join('/')

  if (!url || url === '_index') return '/'
  if (!url.startsWith('/')) url = '/' + url
  if (!url.endsWith('/')) url += '/'
  return url
}
