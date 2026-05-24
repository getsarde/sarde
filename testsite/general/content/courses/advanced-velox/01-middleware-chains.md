---
title: "Middleware Chains"
description: "Building composable middleware chains with Velox."
weight: 1
difficulty: "advanced"
duration_minutes: 30
---

Middleware chains in Velox are composable function pipelines. Each middleware wraps the next handler in the chain, creating an onion-like structure.

## Chain Composition

```go
func Chain(middlewares ...Middleware) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        for i := len(middlewares) - 1; i >= 0; i-- {
            next = middlewares[i](next)
        }
        return next
    }
}
```

## Route-Specific Middleware

Apply middleware to specific routes or groups:

```go
r := velox.New()

// Global middleware
r.Use(Logger(), Recovery())

// Group-specific middleware
api := r.Group("/api")
api.Use(Auth())
api.GET("/users", listUsers)
api.POST("/users", createUser)

// Single-route middleware
r.GET("/admin", AdminOnly(), adminHandler)
```
