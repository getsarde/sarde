---
description: The story behind the Velox project and its contributors.
params:
  author: Velox Team
title: About Velox
toc: true
head:
  - tag: meta
    attrs:
      name: author
      content: "Velox Team"
  - tag: meta
    attrs:
      property: "article:section"
      content: "About"
edit_url: "https://github.com/example/velox/edit/main/content/about.md"
---

Velox started as an internal routing library at a mid-size SaaS company. After years of using the standard library's `http.ServeMux` and finding it too limited, the team built a purpose-built router optimized for API servers.

In early 2024, the project was open-sourced under the MIT license. Since then, it has grown into a community-driven effort with contributors from around the world.

## Project Milestones

:::timeline

== January 2024

**Alpha release.** Core routing engine with radix tree matching, path parameters, and wildcard routes.

== March 2024

**Beta release.** Middleware system, plugin API, and initial documentation. First external contributors joined.

== June 2024

**v1.0 stable.** Full test coverage, performance benchmarks, and production deployment guides. Adopted by three major companies.

== December 2024

**v1.5 release.** Plugin ecosystem with auth, rate limiting, and CORS plugins. Documentation site launched.

:::
