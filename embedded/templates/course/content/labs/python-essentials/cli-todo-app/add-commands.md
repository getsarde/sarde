---
title: Add Commands
description: Implement the add, list, and done subcommands.
sidebar:
  order: 2
---

Add the add, list, and done commands. Replace this with your step instructions.

## Instructions

1. Read the subcommand from `sys.argv`
2. Implement `add`, `list`, and `done`
3. Store todos in a JSON file

## Dispatch the subcommand

`main` reads the subcommand from `sys.argv[1]` and passes the remaining arguments on. The `add_todo`, `list_todos`, and `mark_done` functions live in a separate `todo.py` file that you write in this step.

```python title="main.py" focus={10-18}
import sys

from todo import add_todo, list_todos, mark_done


def main():
    if len(sys.argv) < 2:
        sys.exit("usage: todo <add|list|done> [args]")

    match sys.argv[1]:
        case "add":
            add_todo(sys.argv[2:])
        case "list":
            list_todos()
        case "done":
            mark_done(sys.argv[2:])
        case other:
            sys.exit(f"unknown command: {other}")


if __name__ == "__main__":
    main()
```
