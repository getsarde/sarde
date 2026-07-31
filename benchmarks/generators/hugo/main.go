package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Target counts — tuned so Hugo outputs ~2K pages (matching Sarde's 2,035).
// Hugo generates fewer synthetic pages than Sarde (fewer taxonomy pages,
// no search index page, etc.), so we need more content files to compensate.
const (
	blogPosts        = 700
	articlePosts     = 380
	newsPosts        = 250
	docsSections     = 10
	docsPerSection   = 39
	tutorialSections = 5
	tutorialsPerSec  = 19
	courseSections   = 4
	coursesPerSec    = 24
	frPages          = 30
	standalonePages  = 7
)

var tagPool = []string{
	"announcement", "beginner", "intermediate", "advanced", "performance",
	"security", "deployment", "testing", "configuration", "plugins",
	"api", "middleware", "authentication", "concurrency", "observability",
	"best-practices", "architecture", "open-source", "release", "tutorial",
}

var categoryPool = []string{
	"guides", "releases", "engineering", "community", "updates",
}

var authorPool = []string{
	"Alice Chen", "Bob Martinez", "Clara Johansson", "David Kim",
	"Elena Petrova", "Frank Osei", "Grace Tanaka", "Hassan Ali",
}

var badgeTexts = []string{"New", "Updated", "Beta", "Deprecated"}
var calloutTypes = []string{"note", "warning", "tip", "info", "danger"}
var codeLangs = []string{"go", "yaml", "bash", "javascript", "typescript", "json", "sql", "dockerfile", "toml"}

// Hugo Book's hint shortcode maps: info, warning, danger
var calloutToHint = map[string]string{
	"note":    "info",
	"warning": "warning",
	"tip":     "info",
	"info":    "info",
	"danger":  "danger",
}

var docsSectionDefs = []struct {
	Dir, Title, Icon, Domain string
}{
	{"getting-started", "Getting Started", "rocket", "installation and setup"},
	{"core-concepts", "Core Concepts", "book-open", "routing and middleware"},
	{"configuration", "Configuration", "settings", "config files and options"},
	{"api-reference", "API Reference", "code-2", "types and methods"},
	{"deployment", "Deployment", "cloud", "Docker and CI/CD"},
	{"plugins", "Plugins", "puzzle", "plugin API and extensions"},
	{"security", "Security", "shield", "auth and rate limiting"},
	{"advanced", "Advanced", "zap", "performance and internals"},
	{"internals", "Internals", "cpu", "engine architecture"},
	{"troubleshooting", "Troubleshooting", "help-circle", "debugging and FAQ"},
}

var tutorialSectionDefs = []struct {
	Dir, Title string
}{
	{"go-basics", "Go Basics"},
	{"rest-apis", "REST APIs"},
	{"testing-guide", "Testing Guide"},
	{"deploy-guide", "Deployment Guide"},
	{"observability", "Observability"},
}

var courseSectionDefs = []struct {
	Dir, Title string
}{
	{"go-fundamentals", "Go Fundamentals"},
	{"web-development", "Web Development"},
	{"advanced-go", "Advanced Go"},
	{"microservices", "Microservices"},
}

var docsTopics = []string{
	"routing", "handlers", "middleware", "context", "request lifecycle",
	"response writers", "error handling", "logging", "configuration",
	"dependency injection", "testing", "benchmarking", "profiling",
	"caching", "rate limiting", "authentication", "authorization",
	"CORS", "compression", "static files", "templates", "sessions",
	"cookies", "WebSockets", "server-sent events", "graceful shutdown",
	"health checks", "metrics", "tracing", "service discovery",
	"load balancing", "circuit breakers", "retries", "timeouts",
	"connection pooling", "database integration", "ORM patterns",
	"migration management", "query building", "transaction handling",
}

var blogTopics = []string{
	"performance improvements", "new release features", "community updates",
	"security patches", "plugin ecosystem", "migration guide",
	"benchmark results", "architecture decisions", "roadmap update",
	"contributor spotlight", "case study", "best practices",
	"tips and tricks", "deep dive", "tutorial walkthrough",
	"breaking changes", "deprecation notice", "feature preview",
	"integration guide", "troubleshooting common issues",
}

var sentences = []string{
	"This approach simplifies the overall architecture while maintaining backward compatibility.",
	"Performance benchmarks show a significant improvement over the previous implementation.",
	"The configuration options provide fine-grained control over the behavior of each component.",
	"Error handling follows the standard Go pattern of returning errors as the last value.",
	"Middleware functions can be composed to create complex request processing pipelines.",
	"The plugin system supports hot-reloading for rapid development workflows.",
	"Security best practices recommend validating all input at the boundary of the system.",
	"Caching strategies should be chosen based on the specific access patterns of your data.",
	"The logging framework supports structured output in both JSON and human-readable formats.",
	"Database connections are managed through a pool to optimize resource utilization.",
	"Graceful shutdown ensures all in-flight requests complete before the server terminates.",
	"The template engine supports inheritance, partials, and custom function registration.",
	"Rate limiting can be configured per-route or globally using the middleware chain.",
	"Health check endpoints should return both the service status and dependency health.",
	"Metrics collection enables monitoring and alerting based on application-level indicators.",
	"The router uses a radix tree for efficient path matching with minimal allocations.",
	"Context propagation carries request-scoped values across service boundaries.",
	"Compression middleware reduces response sizes by applying gzip or brotli encoding.",
	"Static file serving supports content negotiation and cache-control headers.",
	"WebSocket connections are upgraded from standard HTTP using the handshake protocol.",
	"Tracing provides end-to-end visibility into request processing across services.",
	"Circuit breakers prevent cascading failures by stopping calls to unhealthy dependencies.",
	"Connection pooling reduces latency by reusing established database connections.",
	"The migration system tracks schema versions and supports rollback operations.",
	"Query builders provide a type-safe interface for constructing database queries.",
	"Transaction handling ensures data consistency across multiple database operations.",
	"Service discovery enables dynamic routing to healthy instances in a cluster.",
	"Load balancing distributes incoming requests across multiple backend servers.",
	"Retry logic with exponential backoff handles transient failures gracefully.",
	"Timeout configuration prevents requests from blocking indefinitely on slow operations.",
}

var codeSnippets = map[string][]string{
	"go": {
		`type Server struct {
	router     *Router
	middleware []Middleware
	config     *Config
	logger     Logger
}

func NewServer(opts ...Option) *Server {
	s := &Server{
		router: NewRouter(),
		config: DefaultConfig(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}`,
		`func (s *Server) Handle(method, path string, handler HandlerFunc) {
	s.router.Add(method, path, func(c *Context) error {
		for _, mw := range s.middleware {
			if err := mw(c); err != nil {
				return err
			}
		}
		return handler(c)
	})
}`,
		`func RateLimiter(max int, window time.Duration) Middleware {
	store := NewTokenBucket(max, window)
	return func(c *Context) error {
		key := c.RealIP()
		if !store.Allow(key) {
			return c.JSON(http.StatusTooManyRequests, Map{
				"error": "rate limit exceeded",
				"retry_after": window.Seconds(),
			})
		}
		return c.Next()
	}
}`,
		`func Logger(level string) Middleware {
	return func(c *Context) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		fields := map[string]any{
			"method":   c.Method(),
			"path":     c.Path(),
			"status":   c.Response().Status(),
			"duration": duration.String(),
			"bytes":    c.Response().Size(),
		}
		if err != nil {
			fields["error"] = err.Error()
		}
		log.WithFields(fields).Info("request")
		return err
	}
}`,
		`type Cache struct {
	mu    sync.RWMutex
	items map[string]*entry
	ttl   time.Duration
}

func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || time.Now().After(e.expiry) {
		return nil, false
	}
	return e.value, true
}`,
	},
	"yaml": {
		`server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  max_header_bytes: 1048576

database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  name: "myapp"
  pool_size: 25
  idle_timeout: 5m`,
		`logging:
  level: "info"
  format: "json"
  output: "stdout"
  fields:
    service: "api"
    version: "1.0.0"

middleware:
  rate_limit:
    enabled: true
    max_requests: 100
    window: "1m"
  cors:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]`,
	},
	"bash": {
		`#!/bin/bash
set -euo pipefail

echo "Building application..."
go build -ldflags="-s -w" -o bin/server ./cmd/server

echo "Running tests..."
go test -race -coverprofile=coverage.out ./...

echo "Building Docker image..."
docker build -t myapp:latest .

echo "Done."`,
		`# Deploy to production
export ENV=production
export VERSION=$(git describe --tags --always)

echo "Deploying $VERSION to production..."
kubectl set image deployment/api api=myapp:$VERSION
kubectl rollout status deployment/api --timeout=120s
echo "Deployment complete."`,
	},
	"javascript": {
		`async function fetchData(endpoint, options = {}) {
  const { timeout = 5000, retries = 3 } = options;

  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      const controller = new AbortController();
      const id = setTimeout(() => controller.abort(), timeout);
      const response = await fetch(endpoint, {
        signal: controller.signal,
        ...options,
      });
      clearTimeout(id);
      if (!response.ok) throw new Error(response.statusText);
      return await response.json();
    } catch (err) {
      if (attempt === retries) throw err;
      await new Promise((r) => setTimeout(r, 1000 * attempt));
    }
  }
}`,
		`class EventEmitter {
  constructor() {
    this.listeners = new Map();
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event).push(callback);
    return () => this.off(event, callback);
  }

  emit(event, ...args) {
    const callbacks = this.listeners.get(event) || [];
    callbacks.forEach((cb) => cb(...args));
  }
}`,
	},
	"typescript": {
		`interface RequestConfig {
  method: "GET" | "POST" | "PUT" | "DELETE";
  path: string;
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
}

interface Response<T> {
  data: T;
  status: number;
  headers: Record<string, string>;
}

async function request<T>(config: RequestConfig): Promise<Response<T>> {
  const response = await fetch(config.path, {
    method: config.method,
    headers: config.headers,
    body: config.body ? JSON.stringify(config.body) : undefined,
  });
  const data = await response.json();
  return { data, status: response.status, headers: {} };
}`,
	},
	"json": {
		`{
  "name": "myapp",
  "version": "2.1.0",
  "endpoints": [
    { "method": "GET", "path": "/api/users", "auth": true },
    { "method": "POST", "path": "/api/users", "auth": true },
    { "method": "GET", "path": "/api/health", "auth": false },
    { "method": "DELETE", "path": "/api/sessions", "auth": true }
  ],
  "features": {
    "rate_limiting": true,
    "cors": true,
    "compression": true
  }
}`,
	},
	"sql": {
		`CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    email      VARCHAR(255) NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    role       VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);`,
		`SELECT
    u.name,
    u.email,
    COUNT(o.id) AS order_count,
    COALESCE(SUM(o.total), 0) AS total_spent
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.created_at >= NOW() - INTERVAL '30 days'
GROUP BY u.id, u.name, u.email
HAVING COUNT(o.id) > 0
ORDER BY total_spent DESC
LIMIT 50;`,
	},
	"dockerfile": {
		`FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]`,
	},
	"toml": {
		`[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"

[database]
driver = "postgres"
host = "localhost"
port = 5432
name = "myapp"
max_open_conns = 25
max_idle_conns = 5

[cache]
driver = "redis"
address = "localhost:6379"
ttl = "5m"`,
	},
}

var tableHeaders = [][]string{
	{"Option", "Type", "Default", "Description"},
	{"Method", "Endpoint", "Auth", "Description"},
	{"Field", "Type", "Required", "Notes"},
	{"Status", "Code", "Meaning", "Action"},
	{"Parameter", "Type", "Default", "Description"},
	{"Feature", "Status", "Version", "Notes"},
}

var tableValues = [][]string{
	{"string", "int", "bool", "duration", "float64", "[]string", "map[string]any"},
	{"true", "false", "nil", `"default"`, "100", "30s", "1024"},
	{"required", "optional", "computed", "deprecated", "internal"},
	{"stable", "beta", "alpha", "experimental", "removed"},
}

// ── RNG helpers ────────────────────────────────────────────────────────

func pick(rng *rand.Rand, items []string) string {
	return items[rng.Intn(len(items))]
}

func pickN(rng *rand.Rand, items []string, n int) []string {
	if n >= len(items) {
		return items
	}
	perm := rng.Perm(len(items))
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = items[perm[i]]
	}
	return result
}

func slug(title string) string {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' {
			return '-'
		}
		return -1
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// ── Markdown content builders ──────────────────────────────────────────

func paragraph(rng *rand.Rand, count int) string {
	if count <= 0 {
		count = 3 + rng.Intn(3)
	}
	picked := pickN(rng, sentences, count)
	return strings.Join(picked, " ") + "\n"
}

func codeBlock(rng *rand.Rand, lang string) string {
	snippets, ok := codeSnippets[lang]
	if !ok {
		snippets = codeSnippets["go"]
	}
	return fmt.Sprintf("```%s\n%s\n```\n", lang, pick(rng, snippets))
}

func table(rng *rand.Rand) string {
	headers := tableHeaders[rng.Intn(len(tableHeaders))]
	var b strings.Builder
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	b.WriteString("|" + strings.Repeat(" --- |", len(headers)) + "\n")
	rows := 4 + rng.Intn(5)
	for r := 0; r < rows; r++ {
		b.WriteString("|")
		for range headers {
			b.WriteString(" `" + pick(rng, tableValues[rng.Intn(len(tableValues))]) + "` |")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Hugo Book uses {{< hint [info|warning|danger] >}} shortcode for callouts.
func callout(rng *rand.Rand) string {
	kind := pick(rng, calloutTypes)
	body := pick(rng, sentences)
	hint := calloutToHint[kind]
	return fmt.Sprintf("{{< hint %s >}}\n%s\n{{< /hint >}}\n", hint, body)
}

func bulletList(rng *rand.Rand) string {
	count := 4 + rng.Intn(5)
	var b strings.Builder
	for i := 0; i < count; i++ {
		b.WriteString("- " + pick(rng, sentences) + "\n")
	}
	return b.String()
}

func standardBody(rng *rand.Rand, sections int, domain string) string {
	var b strings.Builder
	if sections <= 0 {
		sections = 4 + rng.Intn(3)
	}
	for i := 0; i < sections; i++ {
		topic := pick(rng, docsTopics)
		b.WriteString(fmt.Sprintf("## %s\n\n", strings.Title(topic)))
		b.WriteString(paragraph(rng, 0))
		b.WriteString("\n")

		if rng.Intn(3) == 0 {
			b.WriteString("### Implementation Details\n\n")
			b.WriteString(paragraph(rng, 2))
			b.WriteString("\n")
		}

		switch rng.Intn(5) {
		case 0:
			b.WriteString(codeBlock(rng, pick(rng, codeLangs)))
			b.WriteString("\n")
		case 1:
			b.WriteString(table(rng))
			b.WriteString("\n")
		case 2:
			b.WriteString(callout(rng))
			b.WriteString("\n")
		case 3:
			b.WriteString(bulletList(rng))
			b.WriteString("\n")
		default:
			b.WriteString(codeBlock(rng, "go"))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func tocHeavyBody(rng *rand.Rand) string {
	var b strings.Builder
	headings := 8 + rng.Intn(5)
	for i := 0; i < headings; i++ {
		topic := pick(rng, docsTopics)
		b.WriteString(fmt.Sprintf("## %s\n\n", strings.Title(topic)))
		b.WriteString(paragraph(rng, 3))
		b.WriteString("\n")
		if rng.Intn(2) == 0 {
			b.WriteString(fmt.Sprintf("### %s Details\n\n", strings.Title(pick(rng, docsTopics))))
			b.WriteString(paragraph(rng, 2))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// ── Frontmatter builders (Hugo-adapted) ───────────────────────────────

func blogFrontmatter(title, date, desc string, tags, cats []string, author string, featured, draft bool) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", title))
	b.WriteString(fmt.Sprintf("date: %s\n", date))
	b.WriteString(fmt.Sprintf("description: %q\n", desc))
	if len(tags) > 0 {
		b.WriteString(fmt.Sprintf("tags: [%s]\n", quoteJoin(tags)))
	}
	if len(cats) > 0 {
		b.WriteString(fmt.Sprintf("categories: [%s]\n", quoteJoin(cats)))
	}
	if draft {
		b.WriteString("draft: true\n")
	}
	b.WriteString(fmt.Sprintf("author: %q\n", author))
	if featured {
		b.WriteString("featured: true\n")
	}
	b.WriteString("---\n\n")
	return b.String()
}

func docsFrontmatter(title, desc string, weight int, tags []string, badge string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", title))
	b.WriteString(fmt.Sprintf("description: %q\n", desc))
	b.WriteString(fmt.Sprintf("weight: %d\n", weight))
	if len(tags) > 0 {
		b.WriteString(fmt.Sprintf("tags: [%s]\n", quoteJoin(tags)))
	}
	if badge != "" {
		b.WriteString(fmt.Sprintf("badge: %q\n", badge))
	}
	b.WriteString("---\n\n")
	return b.String()
}

func courseFrontmatter(title, desc string, weight int, difficulty string, duration int) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", title))
	b.WriteString(fmt.Sprintf("description: %q\n", desc))
	b.WriteString(fmt.Sprintf("weight: %d\n", weight))
	b.WriteString(fmt.Sprintf("difficulty: %q\n", difficulty))
	b.WriteString(fmt.Sprintf("duration_minutes: %d\n", duration))
	b.WriteString("---\n\n")
	return b.String()
}

func sectionFrontmatter(title, desc string, weight int, collapse bool) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", title))
	b.WriteString(fmt.Sprintf("description: %q\n", desc))
	if weight > 0 {
		b.WriteString(fmt.Sprintf("weight: %d\n", weight))
	}
	if collapse {
		b.WriteString("bookCollapseSection: true\n")
	}
	b.WriteString("---\n\n")
	return b.String()
}

func quoteJoin(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(quoted, ", ")
}

// ── Page assemblers ────────────────────────────────────────────────────

func buildBlogPost(rng *rand.Rand, i int, date time.Time) (filename, content string) {
	topic := pick(rng, blogTopics)
	title := fmt.Sprintf("%s — Part %d", strings.Title(topic), i%10+1)
	desc := fmt.Sprintf("A deep look at %s and how it affects your workflow.", topic)
	dateStr := date.Format("2006-01-02")

	var tags []string
	if i%4 == 0 {
		tags = pickN(rng, tagPool, 1+rng.Intn(3))
	}
	var cats []string
	if i%6 == 0 {
		cats = pickN(rng, categoryPool, 1)
	}
	author := pick(rng, authorPool)
	featured := i%20 == 0
	draft := i%20 == 19

	fm := blogFrontmatter(title, dateStr, desc, tags, cats, author, featured, draft)
	body := standardBody(rng, 4+rng.Intn(3), topic)

	filename = fmt.Sprintf("%s-%s.md", dateStr, slug(topic)+fmt.Sprintf("-%d", i))
	content = fm + body
	return
}

func buildDocsPage(rng *rand.Rand, section string, weight int) (filename, content string) {
	topic := pick(rng, docsTopics)
	title := fmt.Sprintf("%s in %s", strings.Title(topic), strings.Title(section))
	desc := fmt.Sprintf("Learn about %s for the %s section.", topic, section)

	var tags []string
	if rng.Intn(8) == 0 {
		tags = pickN(rng, tagPool, 1+rng.Intn(2))
	}
	var badge string
	if weight%10 == 0 {
		badge = badgeTexts[rng.Intn(len(badgeTexts))]
	}

	fm := docsFrontmatter(title, desc, weight, tags, badge)
	var body string
	if weight%10 == 1 {
		body = tocHeavyBody(rng)
	} else {
		body = standardBody(rng, 5+rng.Intn(3), section)
	}

	filename = fmt.Sprintf("%02d-%s.md", weight, slug(topic))
	content = fm + body
	return
}

func buildTutorialPage(rng *rand.Rand, section string, weight int) (filename, content string) {
	topic := pick(rng, docsTopics)
	title := fmt.Sprintf("Tutorial: %s", strings.Title(topic))
	desc := fmt.Sprintf("Step-by-step tutorial on %s.", topic)

	fm := docsFrontmatter(title, desc, weight, nil, "")

	var b strings.Builder
	b.WriteString("{{< hint info >}}\n**Prerequisites:** Basic knowledge of Go and command-line tools.\n{{< /hint >}}\n\n")
	b.WriteString(standardBody(rng, 4+rng.Intn(3), section))
	body := b.String()

	filename = fmt.Sprintf("%02d-%s.md", weight, slug(topic))
	content = fm + body
	return
}

func buildCoursePage(rng *rand.Rand, course string, weight int) (filename, content string) {
	topic := pick(rng, docsTopics)
	title := fmt.Sprintf("Lesson: %s", strings.Title(topic))
	desc := fmt.Sprintf("Course lesson covering %s.", topic)

	difficulties := []string{"beginner", "intermediate", "advanced"}
	diff := difficulties[rng.Intn(len(difficulties))]
	duration := 10 + rng.Intn(50)

	fm := courseFrontmatter(title, desc, weight, diff, duration)
	body := standardBody(rng, 4+rng.Intn(4), course)

	filename = fmt.Sprintf("%02d-%s.md", weight, slug(topic))
	content = fm + body
	return
}

func buildArticle(rng *rand.Rand, i int, date time.Time) (filename, content string) {
	topic := pick(rng, blogTopics)
	title := fmt.Sprintf("Article: %s (#%d)", strings.Title(topic), i+1)
	desc := fmt.Sprintf("An in-depth article about %s.", topic)
	dateStr := date.Format("2006-01-02")

	var tags []string
	if i%5 == 0 {
		tags = pickN(rng, tagPool, 1+rng.Intn(2))
	}
	author := pick(rng, authorPool)
	draft := i%20 == 19

	fm := blogFrontmatter(title, dateStr, desc, tags, nil, author, false, draft)
	body := standardBody(rng, 5+rng.Intn(3), topic)

	filename = fmt.Sprintf("%s-%s-%d.md", dateStr, slug(topic), i)
	content = fm + body
	return
}

func buildNewsPost(rng *rand.Rand, i int, date time.Time) (filename, content string) {
	topic := pick(rng, blogTopics)
	title := fmt.Sprintf("News: %s", strings.Title(topic))
	desc := fmt.Sprintf("Latest update on %s.", topic)
	dateStr := date.Format("2006-01-02")

	fm := blogFrontmatter(title, dateStr, desc, nil, nil, pick(rng, authorPool), false, false)
	body := standardBody(rng, 2+rng.Intn(2), topic)

	filename = fmt.Sprintf("%s-%s-%d.md", dateStr, slug(topic), i)
	content = fm + body
	return
}

// ── File system helpers ────────────────────────────────────────────────

func mustWrite(root, relPath, content string) {
	fullPath := filepath.Join(root, relPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: mkdir %s: %v\n", dir, err)
		os.Exit(1)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: write %s: %v\n", fullPath, err)
		os.Exit(1)
	}
}

// ── Generation orchestrators ───────────────────────────────────────────

func generateBlog(rng *rand.Rand, root string) int {
	mustWrite(root, filepath.Join("content", "blog", "_index.md"),
		sectionFrontmatter("Blog", "Latest posts and articles from the team.", 0, false))
	count := 1

	baseDate := time.Date(2022, 1, 3, 0, 0, 0, 0, time.UTC)
	for i := 0; i < blogPosts; i++ {
		date := baseDate.AddDate(0, 0, i*2+rng.Intn(3))
		filename, content := buildBlogPost(rng, i, date)
		mustWrite(root, filepath.Join("content", "blog", filename), content)
		count++
	}
	return count
}

func generateArticles(rng *rand.Rand, root string) int {
	mustWrite(root, filepath.Join("content", "articles", "_index.md"),
		sectionFrontmatter("Articles", "In-depth articles on software engineering.", 0, false))
	count := 1

	baseDate := time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < articlePosts; i++ {
		date := baseDate.AddDate(0, 0, i*3+rng.Intn(4))
		filename, content := buildArticle(rng, i, date)
		mustWrite(root, filepath.Join("content", "articles", filename), content)
		count++
	}
	return count
}

func generateNews(rng *rand.Rand, root string) int {
	mustWrite(root, filepath.Join("content", "news", "_index.md"),
		sectionFrontmatter("News", "Short updates and announcements.", 0, false))
	count := 1

	baseDate := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)
	for i := 0; i < newsPosts; i++ {
		date := baseDate.AddDate(0, 0, i*5+rng.Intn(5))
		filename, content := buildNewsPost(rng, i, date)
		mustWrite(root, filepath.Join("content", "news", filename), content)
		count++
	}
	return count
}

func generateDocs(rng *rand.Rand, root string) int {
	mustWrite(root, filepath.Join("content", "docs", "_index.md"),
		sectionFrontmatter("Documentation", "Complete reference documentation.", 0, false))
	count := 1

	for si, sec := range docsSectionDefs {
		secDir := filepath.Join("content", "docs", sec.Dir)
		mustWrite(root, filepath.Join(secDir, "_index.md"),
			sectionFrontmatter(sec.Title, fmt.Sprintf("Documentation for %s.", sec.Domain), (si+1)*10, true))
		count++

		for w := 1; w <= docsPerSection; w++ {
			filename, content := buildDocsPage(rng, sec.Dir, w)
			mustWrite(root, filepath.Join(secDir, filename), content)
			count++
		}
	}
	return count
}

func generateTutorials(rng *rand.Rand, root string) int {
	mustWrite(root, filepath.Join("content", "tutorials", "_index.md"),
		sectionFrontmatter("Tutorials", "Hands-on tutorials to learn by doing.", 0, false))
	count := 1

	for si, sec := range tutorialSectionDefs {
		secDir := filepath.Join("content", "tutorials", sec.Dir)
		mustWrite(root, filepath.Join(secDir, "_index.md"),
			sectionFrontmatter(sec.Title, fmt.Sprintf("Tutorials about %s.", sec.Title), (si+1)*10, true))
		count++

		for w := 1; w <= tutorialsPerSec; w++ {
			filename, content := buildTutorialPage(rng, sec.Dir, w)
			mustWrite(root, filepath.Join(secDir, filename), content)
			count++
		}
	}
	return count
}

func generateCourses(rng *rand.Rand, root string) int {
	mustWrite(root, filepath.Join("content", "courses", "_index.md"),
		sectionFrontmatter("Courses", "Structured courses for all skill levels.", 0, false))
	count := 1

	for si, sec := range courseSectionDefs {
		secDir := filepath.Join("content", "courses", sec.Dir)
		mustWrite(root, filepath.Join(secDir, "_index.md"),
			sectionFrontmatter(sec.Title, fmt.Sprintf("A complete course on %s.", sec.Title), (si+1)*10, true))
		count++

		for w := 1; w <= coursesPerSec; w++ {
			filename, content := buildCoursePage(rng, sec.Dir, w)
			mustWrite(root, filepath.Join(secDir, filename), content)
			count++
		}
	}
	return count
}

func generateStandalones(root string) int {
	pages := []struct {
		file, title, desc, body string
	}{
		{"_index.md", "Hugo Benchmark", "A 2000-page benchmark site for Hugo.", "Welcome to the Hugo benchmark site.\n"},
		{"about.md", "About", "About this benchmark site.", "## About\n\nThis site exists to benchmark the Hugo static site generator build pipeline.\n"},
		{"contact.md", "Contact", "Get in touch.", "## Contact\n\nReach out via the project repository.\n"},
		{"privacy.md", "Privacy Policy", "Privacy policy.", "## Privacy\n\nThis is a benchmark site. No data is collected.\n"},
		{"terms.md", "Terms of Service", "Terms of service.", "## Terms\n\nThis site is provided as-is for benchmarking purposes.\n"},
		{"changelog.md", "Changelog", "Version history.", "## Changelog\n\n### v1.0.0\n\n- Initial benchmark site generation.\n"},
		{"404.md", "Page Not Found", "The page you are looking for does not exist.", "## 404\n\nThe page you requested could not be found.\n"},
	}

	for _, p := range pages {
		fm := sectionFrontmatter(p.title, p.desc, 0, false)
		mustWrite(root, filepath.Join("content", p.file), fm+p.body)
	}
	return len(pages)
}

func generateFR(rng *rand.Rand, root string) int {
	count := 0

	mustWrite(root, filepath.Join("content", "fr", "_index.md"),
		sectionFrontmatter("Accueil", "Site de benchmark Hugo en français.", 0, false))
	count++

	mustWrite(root, filepath.Join("content", "fr", "blog", "_index.md"),
		sectionFrontmatter("Blog", "Articles et publications de l'équipe.", 0, false))
	count++

	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 6; i++ {
		date := baseDate.AddDate(0, i*2, 0)
		dateStr := date.Format("2006-01-02")
		topic := pick(rng, blogTopics)
		title := fmt.Sprintf("Article : %s", strings.Title(topic))
		fm := blogFrontmatter(title, dateStr, fmt.Sprintf("Un article sur %s.", topic), nil, nil, pick(rng, authorPool), false, false)
		body := standardBody(rng, 3, topic)
		filename := fmt.Sprintf("%s-%s-%d.md", dateStr, slug(topic), i)
		mustWrite(root, filepath.Join("content", "fr", "blog", filename), fm+body)
		count++
	}

	mustWrite(root, filepath.Join("content", "fr", "docs", "_index.md"),
		sectionFrontmatter("Documentation", "Documentation complète en français.", 0, false))
	count++

	frDocsSections := []struct {
		dir, title string
		pages      int
	}{
		{"getting-started", "Premiers pas", 7},
		{"core-concepts", "Concepts fondamentaux", 7},
		{"configuration", "Configuration", 4},
	}

	for si, sec := range frDocsSections {
		secDir := filepath.Join("content", "fr", "docs", sec.dir)
		mustWrite(root, filepath.Join(secDir, "_index.md"),
			sectionFrontmatter(sec.title, fmt.Sprintf("Section %s en français.", sec.title), (si+1)*10, true))
		count++

		for w := 1; w <= sec.pages; w++ {
			topic := pick(rng, docsTopics)
			title := fmt.Sprintf("%s — %s", strings.Title(topic), sec.title)
			desc := fmt.Sprintf("Apprenez %s dans la section %s.", topic, sec.title)
			fm := docsFrontmatter(title, desc, w, nil, "")
			body := standardBody(rng, 4, sec.dir)
			filename := fmt.Sprintf("%02d-%s.md", w, slug(topic))
			mustWrite(root, filepath.Join(secDir, filename), fm+body)
			count++
		}
	}

	return count
}

// ── main ───────────────────────────────────────────────────────────────

func main() {
	seed := flag.Int64("seed", 42, "random seed for deterministic generation")
	root := flag.String("root", "", "output root directory (default: auto-detect)")
	dryRun := flag.Bool("dry-run", false, "print what would be generated without writing files")
	flag.Parse()

	if *root == "" {
		// Auto-detection only works for a binary built into this source dir
		// (benchmarks/generators/hugo); under `go run` the executable lives
		// in the build cache, so always pass -root explicitly there.
		exe, err := os.Executable()
		if err == nil {
			candidate := filepath.Join(filepath.Dir(exe), "..", "..", "fixtures", "hugo")
			if abs, err := filepath.Abs(candidate); err == nil {
				*root = abs
			}
		}
		if *root == "" {
			*root = "."
		}
	}

	if *dryRun {
		total := standalonePages +
			(blogPosts + 1) +
			(articlePosts + 1) +
			(newsPosts + 1) +
			(1 + docsSections*(1+docsPerSection)) +
			(1 + tutorialSections*(1+tutorialsPerSec)) +
			(1 + courseSections*(1+coursesPerSec)) +
			frPages
		fmt.Printf("Would generate %d pages in %s\n", total, *root)
		return
	}

	rng := rand.New(rand.NewSource(*seed))

	// Clean previous content before regenerating.
	contentDir := filepath.Join(*root, "content")
	if _, err := os.Stat(contentDir); err == nil {
		fmt.Printf("Cleaning %s ...\n", contentDir)
		if err := os.RemoveAll(contentDir); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: cleaning content dir: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Generating Hugo benchmark site in %s ...\n", *root)

	total := 0
	counts := make(map[string]int)

	n := generateStandalones(*root)
	counts["standalone"] = n
	total += n

	n = generateBlog(rng, *root)
	counts["blog"] = n
	total += n

	n = generateArticles(rng, *root)
	counts["articles"] = n
	total += n

	n = generateNews(rng, *root)
	counts["news"] = n
	total += n

	n = generateDocs(rng, *root)
	counts["docs"] = n
	total += n

	n = generateTutorials(rng, *root)
	counts["tutorials"] = n
	total += n

	n = generateCourses(rng, *root)
	counts["courses"] = n
	total += n

	n = generateFR(rng, *root)
	counts["fr"] = n
	total += n

	fmt.Println()
	fmt.Println("Generated content files:")
	order := []string{"blog", "articles", "news", "docs", "tutorials", "courses", "standalone", "fr"}
	for _, key := range order {
		fmt.Printf("  %-14s %d\n", key+"/", counts[key])
	}
	fmt.Printf("\n  Total:         %d pages\n", total)
}
