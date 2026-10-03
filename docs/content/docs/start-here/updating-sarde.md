---
title: Updating Sarde
description: "Check for new releases and update the sarde binary in place with a single command"
sidebar:
  order: 6
---

Run `sarde update` to replace the installed program with the latest release. Sarde downloads the release, verifies it, and swaps the program file in place. The next `sarde` command runs the new version. Releases are published on the [GitHub releases page](https://github.com/getsarde/sarde/releases).

## Check the installed version

Print the installed version:

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

Check whether a newer release exists and read its notes, without installing anything:

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

When the installed version is already the latest, the command prints `✓ Already up to date (v1.2.0)` instead.

## Install the latest version

Run the command without flags:

```sh
sarde update
```

The command shows the release notes, then asks for confirmation before replacing the program:

```text
  Update to v1.2.0? [y/N] y
✓ Updated to v1.2.0
```

Sarde swaps the program file in place, in the folder where it is installed, so the `PATH` stays the same.

If the command fails with `Permission denied while replacing the binary`, the current user cannot write to the program file. Rerun the command with elevated privileges, for example `sudo sarde update`, or reinstall Sarde into a folder your user can write to.

## Flags

`sarde update` accepts two flags:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--check` | bool | `false` | Only check for updates without installing. |
| `--yes`, `-y` | bool | `false` | Skip the confirmation prompt. |

In scripts or continuous integration (CI), stdin is not a terminal and the confirmation prompt cannot be answered. Pass `--yes` to update non-interactively:

```sh
sarde update --yes
```

Without `--yes`, the command exits with `stdin is not a terminal; pass --yes to update non-interactively`.

## Signed releases

Every release is cryptographically signed, and `sarde update` verifies the download before it replaces anything. It checks the downloaded archive against the release checksums, then checks the checksums file itself against an ed25519 signature using keys built into the program. Sarde never installs a download that fails either check, so a tampered or corrupted release cannot replace a working installation.

## Package manager installs

When Sarde was installed through a package manager, `sarde update` does not replace the program, and `--check` does not look up releases. Both print the matching upgrade command instead, because a self-update would leave the package manager's record of the installed version out of date:

| Install method | Suggested command |
|----------------|-------------------|
| Homebrew | `brew upgrade sarde` |
| Scoop | `scoop update sarde` |
| Chocolatey | `choco upgrade sarde` |
| winget | `winget upgrade sarde` |
| System package manager | your distribution's package manager |

## Update notices in `sarde build` and `sarde dev`

`sarde build` and `sarde dev` show a one-line notice when a newer release is available. The lookup runs at most once per 24 hours and caches its result in `~/.sarde/update-check.json`. Sarde announces each release only once.

Sarde skips the notice and the background lookup when any of these apply:

- the `CI` environment variable is set
- the `SARDE_NO_UPDATE_CHECK` environment variable is set (permanent opt-out)
- `--quiet` is passed
- output is not a terminal (for example, piped or redirected)
- the binary is a dev build

## Dev builds

A copy of Sarde compiled from source code, rather than downloaded as a release, reports its version as `dev`. It has no release version to compare against, so `sarde update` exits with `cannot update a dev build; install a release version first`. To switch to release builds and get updates, follow the [installation steps](/start-here/getting-started/#install-sarde).
