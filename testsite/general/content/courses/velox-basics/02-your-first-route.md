---
title: "Your First Route"
description: "Create your first HTTP route with Velox."
weight: 2
difficulty: "beginner"
duration_minutes: 15
---

In this lesson, you'll create a simple HTTP server with a single route.

:::steps

### Create a new Go module

```bash
mkdir velox-demo && cd velox-demo
go mod init velox-demo
go get github.com/example/velox@latest
```

### Write the server code

```go collapse title="main.go — full source"
package main

import (
    "fmt"
    "github.com/example/velox"
)

func main() {
    r := velox.New()

    r.GET("/", func(c *velox.Context) {
        c.String(200, "Hello, Velox!")
    })

    r.GET("/greet/:name", func(c *velox.Context) {
        name := c.Param("name")
        c.String(200, fmt.Sprintf("Hello, %s!", name))
    })

    r.Listen(":8080")
}
```

### Run and test

```bash
go run main.go
```

:::

Open your browser to `http://localhost:8080/greet/World` and you should see "Hello, World!".
