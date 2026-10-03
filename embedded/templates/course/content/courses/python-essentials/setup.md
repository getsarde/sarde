---
title: Setting Up Python
description: Install Python and check your setup.
sidebar:
  order: 1
tags: [python]
---

This lesson walks through installing Python and running your first program. Replace this with your own content.

:::tip[Writing your own content]
Every callout, tab, and step in this template is a Sarde extension. See the [extensions guide](https://getsarde.github.io/sarde/docs/extensions/using-extensions/) for the full list.
:::

## Installing Python

:::tabs
== Windows

Download the installer from [python.org/downloads](https://www.python.org/downloads/) and run it. On the first screen, tick **Add python.exe to PATH**.

== macOS

```bash
brew install python
```

== Linux

```bash
sudo apt install python3
```
:::

:::note
This course uses Python 3.10 or later. Run `python3 --version` in a new terminal to check, or `py --version` on Windows.
:::

## Your first program

Create a file named `main.py`:

```python title="main.py"
def main():
    print("Hello, Python!")


if __name__ == "__main__":
    main()
```

Run it from the same folder:

```sh frame=terminal
python3 main.py
```

The terminal prints `Hello, Python!`. On Windows, run `py main.py` instead.

## Topics

- Installing Python
- Configuring your editor
- Running your first program
