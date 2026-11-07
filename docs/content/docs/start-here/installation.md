---
title: Installation
description: "Install the sarde command on macOS, Linux, or Windows and check that the terminal can find it"
sidebar:
  order: 2
---

Sarde is one program, `sarde`, that runs in a terminal. Install it once, and every site on the machine uses it. Nothing else needs installing: no Node.js, no Go, and no other dependencies.

## Install Sarde

Choose the tab for your operating system:

:::tabs

== macOS

With [Homebrew](https://brew.sh/):

```sh
brew install getsarde/sarde/sarde
```

Without Homebrew, use the install script in the Linux tab. It works on macOS too.

== Linux

Run the install script:

```sh
curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh
```

The script installs `sarde` into `/usr/local/bin`, or into `~/.local/bin` when `/usr/local/bin` is not writable. If it prints `Add <directory> to your PATH`, run the `export` command it shows. Add the same line to the shell profile (for example, `~/.bashrc` or `~/.zshrc`) to keep it for new terminals.

To install by hand instead, download the archive for your platform from [GitHub Releases](https://github.com/getsarde/sarde/releases), extract it, and move the `sarde` program into a folder on the `PATH`.

== Windows

1. Download `sarde_windows_amd64.zip` from [GitHub Releases](https://github.com/getsarde/sarde/releases). The browser saves it in the **Downloads** folder.
2. Extract the archive and move `sarde.exe` into a permanent folder, for example `C:\Users\<name>\sarde`.
3. Add that folder to the `PATH` environment variable so any terminal can find the program: open **Settings > System > About > Advanced system settings**, click **Environment Variables**, select `Path` under *User variables*, then click **Edit > New** and paste the folder path. Click **OK** in each window to save.
4. Close every open terminal window, then open a new PowerShell window. Terminals that were already open do not pick up the `PATH` change.

== From source

Requires [Go](https://go.dev/dl/) 1.25 or later.

```sh
go install github.com/getsarde/sarde/cmd/sarde@latest
```

A copy built from source reports its version as `dev`, and [`sarde update`](/start-here/updating-sarde/#dev-builds) cannot update it. Install a release from another tab to receive updates.

:::

## Check the installation

Run the following command:

```sh
sarde version
```

→ The terminal prints the Sarde version, the Go version, and the platform:

```text
sarde 1.8.0
Go: go1.25.0
OS/Arch: linux/amd64
```

The version number is the release that was installed, so it can be newer than the one shown here.

## If the terminal cannot find `sarde`

An error such as `sarde: command not found` (macOS and Linux) or `The term 'sarde' is not recognized` (Windows) means the folder that holds the program is not on the `PATH`.

1. Open a new terminal window and run `sarde version` again. A terminal reads the `PATH` only when it starts.
2. If the error remains, check that the folder you added in the install steps contains the program (`sarde`, or `sarde.exe` on Windows), then repeat the `PATH` step for your platform.

## Next steps

- [Getting Started](/start-here/getting-started/) creates a first site and previews it in the browser.
- [Updating Sarde](/start-here/updating-sarde/) installs new releases with `sarde update`.
- The deployment guides, such as [GitHub Pages](/deployment/github-pages/), show how a build server installs Sarde.
