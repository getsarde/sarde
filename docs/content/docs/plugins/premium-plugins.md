---
title: Premium Plugins
description: "License premium external plugins with offline, signed license files"
sidebar:
  order: 3
---

:::caution[License format is provisional]
The signed license file format shipped with Sarde 1.0 and is **not** covered by the 1.x compatibility promise: its fields may still change in a 1.x release while the format settles. Verification behavior described below (offline, warn and disable, never fail the build) is stable. Sellers should expect to reissue license files if the format changes, and any such change will be announced in the [changelog](/resources/changelog/).
:::

Premium plugins are [external plugins](/plugins/external-plugins/) that declare `premium: true` in their manifest. A premium plugin installs like any other plugin but stays inactive until a matching license file is present.

## How licensing works

A license is a small signed file issued by the plugin's seller. Sarde verifies the signature entirely offline at build time: no network call, no account, no activation server. Builds behind firewalls and in air-gapped CI work unchanged.

Licenses are obtained directly from whoever sells the plugin, through the `purchase_url` shown by `sarde plugin install` and `sarde plugin info`. There is no central plugin marketplace.

## License file locations

Sarde looks for `<slug>.license` in two places, in order:

```
<project>/.sarde/licenses/grade-sync.license    # checked first
~/.sarde/licenses/grade-sync.license            # checked second
```

The home-directory location is the default: one purchase covers every project on the machine. The project-level location exists for CI pipelines and per-project overrides.

Projects scaffolded with `sarde new site` gitignore `.sarde/`, so license files are not committed by accident.

## Installing a license

```
sarde license install <license-file> [--project]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | bool | `false` | Install into the current project's `.sarde/licenses/` instead of the user home directory. |

```
$ sarde license install grade-sync.license
Installed license for "grade-sync" (licensee: jane@school.edu) to ~/.sarde/licenses/grade-sync.license
License verified successfully.
```

A license that fails verification is still copied, but the command says so plainly:

```
$ sarde license install grade-sync.license
Installed license for "grade-sync" (licensee: jane@school.edu) to ~/.sarde/licenses/grade-sync.license
Warning: license does not verify: invalid license signature
The plugin will stay inactive until a valid license is installed.
```

Copying the file into either location by hand works identically; the command only adds immediate verification feedback.

## Checking license status

```
sarde license list
```

```
PLUGIN               LICENSEE                       EXPIRES      STATUS
grade-sync           jane@school.edu                never        valid
quiz-widgets         jane@school.edu                2026-06-30   license expired on 2026-06-30
```

`sarde plugin list` summarizes the same information in its LICENSE column:

| Value | Meaning |
|-------|---------|
| `free` | Not a premium plugin; no license involved. |
| `premium (no license)` | No license file found at either location. |
| `premium (invalid)` | A license file exists but fails verification. |
| `premium (licensed)` | A valid license is in place; the plugin is active. |

`sarde plugin info <slug>` shows the exact verification problem for a premium plugin, plus its purchase URL.

## What a license file contains

A license is a JSON file:

```json
{
  "v": 1,
  "slug": "grade-sync",
  "licensee": "jane@school.edu",
  "issued": "2026-07-01",
  "expires": "2027-12-31",
  "max_version": "2.9.9",
  "seats": 5,
  "sig": "sCsrqFzF8Cbw/fCircqcaL4HA4Yi..."
}
```

| Field | Description |
|-------|-------------|
| `v` | License format version. |
| `slug` | The plugin this license is for. Must match the installed plugin's slug. |
| `licensee` | Purchaser name or email. |
| `issued` | Issue date (`YYYY-MM-DD`). |
| `expires` | Expiry date. Empty or absent means the license never expires. |
| `max_version` | Highest plugin version covered. Empty or absent covers every version. |
| `seats` | Seat count, informational only. Sarde does not enforce it. |
| `sig` | Cryptographic signature over the other fields. |

Editing any field invalidates the signature. Verification checks the slug, the signature, the expiry date, and `max_version` against the installed plugin's version.

## Missing or invalid licenses

A premium plugin without a valid license is skipped, never fatal. The build completes and reports a warning naming the exact problem and where to buy a license:

```
1 warning(s):
  plugins/grade-sync/plugin.yaml: premium plugin "grade-sync" disabled: no license
  file found (looked for .sarde/licenses/grade-sync.license and
  ~/.sarde/licenses/grade-sync.license). Install a license with
  'sarde license install' (purchase at https://example.com/buy/grade-sync)
```

The possible problems, each with its own message:

- No license file found at either location
- Invalid signature (tampered or wrongly issued file)
- License expired on a given date
- License issued for a different plugin slug
- Installed plugin version above the license's `max_version`

While unlicensed, the plugin injects nothing, copies no assets, and contributes no templates. A deploy never breaks because a license file went missing.

## Licenses survive plugin removal

`sarde plugin remove <slug>` deletes only `plugins/<slug>/`. License files live in `.sarde/licenses/`, outside the plugin directory, so removing, upgrading, or reinstalling a plugin never touches its license.

## Using licenses in CI

`.sarde/` is gitignored, so CI needs to place the license at build time. Store the license file content as a CI secret, then either write it directly:

```
mkdir -p .sarde/licenses
echo "$GRADE_SYNC_LICENSE" > .sarde/licenses/grade-sync.license
sarde build
```

or run the install command as a build step:

```
sarde license install grade-sync.license --project
sarde build
```

Private repositories can instead commit the license file by removing `.sarde/` from `.gitignore`. Avoid this in public repositories.

## Edge cases

- A license is bound to exactly one plugin slug. A valid license for `grade-sync` is rejected for any other plugin, even from the same seller.
- A license remains valid through its expiry date and becomes invalid the day after.
- A license with no `max_version` covers every version of the plugin, including future ones.
- When both locations contain a license for the same slug, the project-level file wins.
