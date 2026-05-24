---
title: "Custom Request Context"
description: "Extending the request context with custom typed values."
weight: 2
difficulty: "advanced"
duration_minutes: 25
---

Velox contexts can carry request-scoped values that are accessible throughout the middleware chain and handler.

:::important
Context values should only be used for request-scoped data like user identity, request IDs, and tracing spans. Never use them as a substitute for function parameters.
:::

## Setting Values

```go
func AuthMiddleware() velox.Middleware {
    return func(next velox.HandlerFunc) velox.HandlerFunc {
        return func(c *velox.Context) {
            user := authenticateRequest(c)
            c.Set("user", user)
            next(c)
        }
    }
}
```

## Reading Values

```go
func handler(c *velox.Context) {
    user := c.Get("user").(*User)
    c.JSON(200, user)
}
```

## Type-Safe Access

For production code, define typed accessor functions:

```go
type contextKey string

const userKey contextKey = "user"

func SetUser(c *velox.Context, u *User) { c.Set(string(userKey), u) }
func GetUser(c *velox.Context) *User     { return c.Get(string(userKey)).(*User) }
```
