---
title: Hello Python
description: Write a command-line greeter in Python.
sidebar:
  order: 1
---

Write a Python program that greets the user by name. Replace this with your assignment instructions.

## Requirements

- Accept a name as a command-line argument
- Print a greeting to stdout
- Handle the case where no name is provided

Run your program with `python main.py` and press ::kbd[Ctrl+C] to stop it.

:::details[Hint]
Command-line arguments are available in `sys.argv`. The first element is the script name.
:::

:::details[Solution]
```python title="main.py" showLineNumbers
import sys


def main():
    name = sys.argv[1] if len(sys.argv) > 1 else "world"
    print(f"Hello, {name}!")


if __name__ == "__main__":
    main()
```
:::
