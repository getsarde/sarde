---
title: Hello Go
description: Write a command-line greeter in Go.
sidebar:
  order: 1
---

Write a Go program that greets the user by name. Replace this with your assignment instructions.

## Requirements

- Accept a name as a command-line argument
- Print a greeting to stdout
- Handle the case where no name is provided

Run your program with `go run .` and press ::kbd[Ctrl+C] to stop it.

:::details[Hint]
Command-line arguments are available in `os.Args`. The first element is the program name.
:::

:::details[Solution]
```go
package main

import (
	"fmt"
	"os"
)

func main() {
	name := "world"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	fmt.Printf("Hello, %s!\n", name)
}
```
:::
