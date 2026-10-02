---
title: Project Setup
description: Create the Go module and entry point.
sidebar:
  order: 1
---

Initialize the Go module and create the entry point. Replace this with your step instructions.

:::steps
1. Run `go mod init todo`
2. Create `main.go`
3. Add a `main` function that prints "Todo App"
:::

Your project should look like this:

:::file-tree
- todo/
  - **main.go**
  - go.mod
:::

Check that it runs:

:::terminal
$ go run .
Todo App
:::
