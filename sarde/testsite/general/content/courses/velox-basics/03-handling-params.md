---
title: "Handling URL Parameters"
description: "How to extract and validate URL path parameters."
weight: 3
difficulty: "beginner"
duration_minutes: 20
---

Velox supports three types of URL parameters: path parameters, query parameters, and wildcards.

:::caution
Path parameter names are case-sensitive. `:userId` and `:userid` are different parameters.
:::

## Path Parameters

Define parameters with a colon prefix:

```go
r.GET("/users/:id", func(c *velox.Context) {
    id := c.Param("id")
    c.JSON(200, map[string]string{"user_id": id})
})
```

## Query Parameters

Access query string values through the context:

```go
r.GET("/search", func(c *velox.Context) {
    q := c.Query("q")
    page := c.QueryInt("page", 1)
    c.JSON(200, map[string]any{"query": q, "page": page})
})
```

:::details[Parameter Validation]

Velox does not validate parameter types automatically. You should validate and convert parameters in your handler:

```go
id, err := strconv.Atoi(c.Param("id"))
if err != nil {
    c.JSON(400, map[string]string{"error": "invalid id"})
    return
}
```

:::
