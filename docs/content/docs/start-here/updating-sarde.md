---
title: Updating Sarde
description: "Check for new releases and update the sarde binary in place with a single command"
sidebar:
  order: 6
---

Sarde ships as a single program with no installer, so updating it does not mean re-downloading an archive, digging out the old file, and swapping it by hand. The program updates itself: run `sarde update`, confirm, and the new version is in place a few seconds later, ready for the next command.

New releases bring new extensions, theme improvements, and bug fixes, and appear on the [GitHub releases page](https://github.com/getsarde/sarde/releases). There is no need to watch that page. Sarde mentions new releases on its own while running, and one command from the terminal handles the rest. This page walks through checking which version is installed, seeing what a new release contains, and installing it, along with what happens behind the scenes to keep updates safe.

## Check the installed version

Before updating, it helps to know the starting point:

```sh
sarde version
```

→ The terminal prints the installed version, Go version, and platform:

```text
sarde 1.1.0
Go: go1.25.0
OS/Arch: windows/amd64
```

## Check for updates

To see whether a newer release exists, and what changed in it, without installing anything yet:

```sh
sarde update --check
```

→ When a newer release exists, the terminal shows the current and latest versions, the release notes, and a link to the release page. Nothing is installed.

```text
  Current: v1.1.0
  Latest:  v1.2.0

## Changelog
* feat(theme): underline unstyled prose links ...
  ...

  Release page: https://github.com/getsarde/sarde/releases/tag/v1.2.0

  Run sarde update to install
```

## Install the latest version

When ready to update, run the command without flags:

```sh
sarde update
```

The command shows the release notes, then asks for confirmation before replacing the program:

```text
  Update to v1.2.0? [y/N] y
✓ Updated to v1.2.0
```

The program file is swapped in place, in the same folder where it already lives. The next `sarde` command runs the new version. There is no restart, re-download, or PATH change to make.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--check` | bool | `false` | Only check for updates without installing. |
| `--yes`, `-y` | bool | `false` | Skip the confirmation prompt. |

In scripts or CI, stdin is not a terminal and the confirmation prompt cannot be answered. Pass `--yes` to update non-interactively:

```sh
sarde update --yes
```

## Signed releases

Replacing the very program being run is a sensitive operation, so every release is cryptographically signed and verified before anything is touched. `sarde update` checks the downloaded archive against the release checksums, then checks the checksums file itself against an ed25519 signature using keys built into the program. A download that fails either check is never installed, even if it downloaded without errors. In practice this means a tampered or corrupted release cannot replace a working installation.

## Package manager installs

Package managers like Homebrew keep their own records of what is installed where. If `sarde update` replaced a managed program behind the manager's back, the manager's records would no longer match reality and its next upgrade could misbehave. So when Sarde detects it was installed through a package manager, it steps aside and prints the matching upgrade command instead:

| Install method | Suggested command |
|----------------|-------------------|
| Homebrew | `brew upgrade sarde` |
| Scoop | `scoop update sarde` |
| Chocolatey | `choco upgrade sarde` |
| winget | `winget upgrade sarde` |
| System package manager | your distribution's package manager |

## Update notices in build and dev

There is no need to run `sarde update --check` on a schedule. During normal work, `sarde build` and `sarde dev` show a one-line notice when a newer release is available, so news of a release arrives on its own. The lookup runs at most once per 24 hours and caches its result in `~/.sarde/update-check.json`. Each release is announced only once, so the notice never nags.

The notice and the background lookup are skipped when any of these apply:

- the `CI` environment variable is set
- the `SARDE_NO_UPDATE_CHECK` environment variable is set (permanent opt-out)
- `--quiet` is passed
- output is not a terminal (for example, piped or redirected)
- the binary is a dev build

## Dev builds

A copy of Sarde compiled from source code, rather than downloaded as a release, reports its version as `dev` and refuses to self-update, since there is no release version to compare against. To switch to release builds and get updates, install Sarde with the [installation steps](/docs/start-here/getting-started/#install-sarde).
