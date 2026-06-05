---
title: "Syntax Highlighting Showcase"
weight: 4
---

A showcase of Shiki syntax highlighting across multiple languages.

## Go

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
	})
	http.ListenAndServe(":8080", nil)
}
```

## JavaScript

```javascript
async function fetchUsers(query) {
  const response = await fetch(`/api/users?q=${encodeURIComponent(query)}`);
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }
  const { data, pagination } = await response.json();
  return data.map(user => ({
    id: user.id,
    name: `${user.firstName} ${user.lastName}`,
    role: user.role ?? 'viewer',
  }));
}
```

## TypeScript

```typescript
interface Config<T extends Record<string, unknown>> {
  readonly name: string;
  version: number;
  features: Map<string, boolean>;
  transform: (input: T) => Promise<T>;
}

class AppBuilder<T extends Record<string, unknown>> {
  private plugins: Array<(ctx: T) => void> = [];

  use(plugin: (ctx: T) => void): this {
    this.plugins.push(plugin);
    return this;
  }

  async build(config: Config<T>): Promise<void> {
    for (const plugin of this.plugins) {
      plugin({} as T);
    }
    console.log(`Built ${config.name} v${config.version}`);
  }
}
```

## PHP

```php
<?php

namespace App\Http\Controllers;

use App\Models\User;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class UserController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        $users = User::query()
            ->when($request->has('role'), fn($q) => $q->where('role', $request->role))
            ->orderBy('created_at', 'desc')
            ->paginate(25);

        return response()->json([
            'data' => $users->items(),
            'meta' => [
                'total' => $users->total(),
                'page' => $users->currentPage(),
            ],
        ]);
    }

    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'name' => 'required|string|max:255',
            'email' => 'required|email|unique:users',
            'role' => 'in:admin,editor,viewer',
        ]);

        $user = User::create($validated);

        return response()->json($user, 201);
    }
}
```

## HTML

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Dashboard</title>
  <link rel="stylesheet" href="/assets/css/main.css">
</head>
<body>
  <header class="site-header">
    <nav aria-label="Main navigation">
      <a href="/" class="logo">Velox</a>
      <ul class="nav-links">
        <li><a href="/docs">Docs</a></li>
        <li><a href="/blog">Blog</a></li>
      </ul>
    </nav>
  </header>
  <main id="app">
    <section class="hero" data-animate="fade-in">
      <h1>Build faster with <strong>Velox</strong></h1>
      <p>A lightweight Go HTTP router.</p>
      <a href="/docs/getting-started" class="btn btn-primary">Get Started</a>
    </section>
  </main>
  <script type="module" src="/assets/js/app.js"></script>
</body>
</html>
```

## CSS

```css
:root {
  --color-primary: #6366f1;
  --color-surface: #1e1e2e;
  --radius: 0.5rem;
  --transition: 200ms cubic-bezier(0.4, 0, 0.2, 1);
}

.card {
  background: var(--color-surface);
  border-radius: var(--radius);
  padding: 1.5rem;
  box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
  transition: transform var(--transition), box-shadow var(--transition);

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.2);
  }

  & .card-title {
    font-size: 1.25rem;
    font-weight: 600;
    margin-bottom: 0.5rem;
  }
}

@media (prefers-color-scheme: dark) {
  :root {
    --color-surface: #0f0f1a;
  }
}
```

## Python

```python
from dataclasses import dataclass, field
from typing import Optional
import asyncio
import httpx

@dataclass
class CacheEntry:
    key: str
    value: str
    ttl: int = 300
    hits: int = field(default=0, repr=False)

class ApiClient:
    def __init__(self, base_url: str, token: Optional[str] = None):
        self.base_url = base_url
        self._headers = {"Authorization": f"Bearer {token}"} if token else {}
        self._cache: dict[str, CacheEntry] = {}

    async def get(self, path: str) -> dict:
        if path in self._cache:
            self._cache[path].hits += 1
            return {"data": self._cache[path].value, "cached": True}

        async with httpx.AsyncClient() as client:
            resp = await client.get(
                f"{self.base_url}{path}",
                headers=self._headers,
            )
            resp.raise_for_status()
            data = resp.json()

        self._cache[path] = CacheEntry(key=path, value=data)
        return {"data": data, "cached": False}

async def main():
    client = ApiClient("https://api.example.com", token="sk-abc123")
    users = await client.get("/v1/users")
    print(f"Fetched {len(users['data'])} users (cached={users['cached']})")

if __name__ == "__main__":
    asyncio.run(main())
```

## SQL

```sql
WITH monthly_revenue AS (
  SELECT
    date_trunc('month', o.created_at) AS month,
    p.category,
    SUM(oi.quantity * oi.unit_price) AS revenue,
    COUNT(DISTINCT o.customer_id) AS unique_customers
  FROM orders o
  JOIN order_items oi ON oi.order_id = o.id
  JOIN products p ON p.id = oi.product_id
  WHERE o.status = 'completed'
    AND o.created_at >= NOW() - INTERVAL '12 months'
  GROUP BY 1, 2
)
SELECT
  month,
  category,
  revenue,
  unique_customers,
  ROUND(revenue / NULLIF(unique_customers, 0), 2) AS avg_revenue_per_customer,
  LAG(revenue) OVER (PARTITION BY category ORDER BY month) AS prev_month_revenue
FROM monthly_revenue
ORDER BY month DESC, revenue DESC;
```

## Rust

```rust
use std::collections::HashMap;
use tokio::sync::RwLock;

#[derive(Debug, Clone)]
struct Config {
    max_retries: u32,
    timeout_ms: u64,
    endpoints: Vec<String>,
}

struct ServiceRegistry {
    services: RwLock<HashMap<String, Config>>,
}

impl ServiceRegistry {
    fn new() -> Self {
        Self {
            services: RwLock::new(HashMap::new()),
        }
    }

    async fn register(&self, name: &str, config: Config) -> Result<(), String> {
        let mut services = self.services.write().await;
        if services.contains_key(name) {
            return Err(format!("Service '{}' already registered", name));
        }
        services.insert(name.to_string(), config);
        Ok(())
    }

    async fn get(&self, name: &str) -> Option<Config> {
        let services = self.services.read().await;
        services.get(name).cloned()
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let registry = ServiceRegistry::new();
    registry.register("auth", Config {
        max_retries: 3,
        timeout_ms: 5000,
        endpoints: vec!["https://auth.example.com".into()],
    }).await?;

    if let Some(cfg) = registry.get("auth").await {
        println!("Auth service: {:?}", cfg);
    }
    Ok(())
}
```

## YAML

```yaml
version: "3.8"

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        GO_VERSION: "1.22"
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://user:pass@db:5432/app
      REDIS_URL: redis://cache:6379
      LOG_LEVEL: info
    depends_on:
      db:
        condition: service_healthy
    deploy:
      replicas: 3
      resources:
        limits:
          memory: 512M

  db:
    image: postgres:16-alpine
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user"]
      interval: 5s
      retries: 5

volumes:
  pgdata:
```

## JSON

```json
{
  "name": "@velox/core",
  "version": "2.1.0",
  "description": "Lightweight HTTP router for Go",
  "repository": {
    "type": "git",
    "url": "https://github.com/example/velox"
  },
  "scripts": {
    "build": "go build -ldflags='-s -w' -o dist/velox ./cmd/velox",
    "test": "go test -race -cover ./...",
    "lint": "golangci-lint run"
  },
  "keywords": ["go", "http", "router", "middleware"],
  "license": "MIT"
}
```

## Bash

```bash
#!/usr/bin/env bash
set -euo pipefail

readonly APP_NAME="velox"
readonly BUILD_DIR="dist"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
err() { printf '\033[1;31mERR\033[0m %s\n' "$*" >&2; exit 1; }

build() {
  local version
  version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

  log "Building ${APP_NAME} ${version}..."
  mkdir -p "${BUILD_DIR}"

  for os in linux darwin windows; do
    for arch in amd64 arm64; do
      local output="${BUILD_DIR}/${APP_NAME}-${os}-${arch}"
      [[ "$os" == "windows" ]] && output+=".exe"

      GOOS="$os" GOARCH="$arch" go build \
        -ldflags "-X main.Version=${version}" \
        -o "$output" ./cmd/velox

      log "  Built ${output}"
    done
  done
}

build "$@"
```
