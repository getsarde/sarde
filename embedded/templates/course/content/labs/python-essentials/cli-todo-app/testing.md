---
title: Testing
description: Verify the todo commands with automated tests.
sidebar:
  order: 3
---

Write tests for the todo commands. Replace this with your step instructions.

## Instructions

1. Install pytest with `python -m pip install pytest`
2. Create `test_todo.py` and test the add and list commands
3. Run pytest and verify all tests pass

## Write the test

pytest runs every function named `test_*` in files named `test_*.py`. A plain `assert` statement reports the failure:

```python title="test_todo.py" {"Assertion":4-5}
def test_add_todo():
    todos = []
    todos.append("write tests")

    assert len(todos) == 1
```

Run the tests:

:::terminal
$ python -m pytest
1 passed in 0.01s
:::
