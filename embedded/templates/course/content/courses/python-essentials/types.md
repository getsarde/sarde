---
title: Types and Variables
description: Learn Python's basic types, variables, and conversions.
sidebar:
  order: 2
tags: [python]
---

This lesson introduces Python's type system. Replace this with your own content.

:::warning
Type hints are not checked when the program runs: `age: int = "thirty"` runs without an error. Use a type checker such as mypy to catch mismatches before they reach users.
:::

## Declaring variables

A variable gets its type from the value assigned to it. A type hint after the name documents the type you expect:

```python title="types.py" showLineNumbers {"Type hints":5-7} "type"
count = 0      # int
price = 9.99   # float
name = "Ada"   # str

age: int = 30
ratio: float = 0.5
ready: bool = False

print(type(count), type(price), type(name))
```

## Topics

- Basic types
- Variables and assignment
- Type conversions
