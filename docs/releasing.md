*Document generated with LLM assistance*

# Release setup

This document describes the accounts, tokens, and one-time steps behind pgfmt's release workflow, and what happens when a version is released. After the setup is complete, pushing a version tag publishes every package.

## Channels

| Channel | Package | Published by |
|---|---|---|
| GitHub releases | `pgfmt-<os>-<arch>` archives and `checksums.txt` | the workflow, with the built-in token |
| Homebrew | `sirmspencer/tap/pgfmt`, a formula that installs the prebuilt binary | the workflow, with `TAP_TOKEN` |
| npm | `@sirmspencer/pgfmt` and one `@sirmspencer/pgfmt-<os>-<cpu>` package per platform | the workflow, with `NPM_TOKEN` |
| Visual Studio Marketplace | `sirmspencer.pgfmt`, one package per platform | the workflow, with `VSCE_PAT` |
| Open VSX | `sirmspencer.pgfmt`, one package per platform | the workflow, with `OVSX_PAT` |
| JetBrains Marketplace | the pgfmt plugin, one zip with every platform's binary | by hand for the first version, then the workflow, with `JETBRAINS_MARKETPLACE_TOKEN` |

The platforms are macOS on Apple silicon and Intel, Linux on x64 and ARM, and Windows on x64. pgfmt links the PostgreSQL parser through cgo, so each platform is built and tested on its own runner; macOS builds both of its architectures on one runner.

## Accounts

| Account | Purpose |
|---|---|
| GitHub user `sirmspencer` | owns `sirmspencer/pgfmt` and `sirmspencer/homebrew-tap` |
| npm user or organization `sirmspencer` | owns the `@sirmspencer` scope |
| Visual Studio Marketplace publisher `sirmspencer` | publishes the VS Code extension; tokens come from the Azure DevOps account behind the publisher |
| Open VSX account | publishes to the `sirmspencer` namespace, which the workflow creates on first use |
| JetBrains account with a vendor profile | publishes the IntelliJ plugin |

## Tokens

Each token is stored as a repository secret with the name below, under the repository's Settings, Secrets and variables, Actions.

| Secret | Where to create it | Settings |
|---|---|---|
| `TAP_TOKEN` | github.com/settings/personal-access-tokens/new, signed in as `sirmspencer` | resource owner `sirmspencer`; only the `homebrew-tap` repository; Contents: Read and write |
| `NPM_TOKEN` | npmjs.com, Access Tokens, Generate New Token | granular token with read and write access to the `@sirmspencer` scope |
| `VSCE_PAT` | dev.azure.com, User settings, Personal access tokens | organization: All accessible organizations; scope: Marketplace, Manage |
| `OVSX_PAT` | open-vsx.org, Settings, Access Tokens | |
| `JETBRAINS_MARKETPLACE_TOKEN` | plugins.jetbrains.com, profile, My Tokens | |

Most of these tokens expire. When one does, the job that uses it fails, and a new token replaces the secret.

### Setting the secrets with a script

`scripts/set-secrets.sh` stores every `NAME=value` line of a file as a repository secret, from `.envrc` at the repository root by default. `.envrc` is in `.gitignore`. The script uses only the values in the file, skips empty values, and pipes each value to `gh` on stdin, so values do not appear in the process list or the output. Lines work with or without `export`, and direnv is not needed.

```sh
# .envrc
TAP_TOKEN=...
NPM_TOKEN=...
VSCE_PAT=...
OVSX_PAT=...
JETBRAINS_MARKETPLACE_TOKEN=...
```

```sh
scripts/set-secrets.sh                      # reads .envrc
scripts/set-secrets.sh ~/pgfmt-secrets.env  # reads another file
```

Existing secrets are replaced, so updating one token means changing its line and running the script again. The script never prints a value; it prints the first 8 characters of each value's SHA-256 hash instead, to confirm which token was stored. To compare, copy the intended token and run `pbpaste | shasum -a 256 | cut -c1-8` on macOS, which keeps the token out of the shell history.

The script stores whatever the file contains. If a secret is changed on GitHub directly, the file needs the same change, or the next run of the script replaces the secret with the old value. A `JETBRAINS_PUBLISH` line is stored as a repository variable instead of a secret. `gh` must be signed in as an account with admin access to the repository.

## One-time steps

1. **JetBrains Marketplace.** The first version of the plugin must be uploaded by hand at plugins.jetbrains.com/plugin/add, using the `pgfmt-intellij-<version>.zip` attached to a GitHub release. The form asks for the license (MIT) and the source code link. JetBrains reviews every new plugin before it becomes public.
2. **Automatic JetBrains publishing.** After the first version is approved, the repository variable `JETBRAINS_PUBLISH` is set to `true`, under Settings, Secrets and variables, Actions, Variables. Until then, the workflow builds the plugin and attaches it to the release without publishing it.
3. **Homebrew users** run `brew trust sirmspencer/tap` once, as recent Homebrew versions require for third-party taps.

## Releasing a version

```sh
git tag v0.1.3
git push origin v0.1.3
```

The `release` workflow then runs these jobs:

1. **build:** runs the tests and builds the archives on each platform's runner.
2. **release:** creates the GitHub release with the archives and `checksums.txt`.
3. **homebrew:** writes the formula from `checksums.txt`, installs and tests it on macOS from a checkout of the tap, and pushes it to `sirmspencer/homebrew-tap` only if the test passes.
4. **npm:** publishes the platform packages, then `@sirmspencer/pgfmt`, with provenance. Versions that already exist are skipped, so the job can be rerun.
5. **vscode:** packages one extension per platform with its binary, attaches them to the release, and publishes them to the Visual Studio Marketplace and Open VSX.
6. **jetbrains:** builds the plugin with every binary, attaches it to the release, and publishes it when `JETBRAINS_PUBLISH` is `true`.

Running the workflow by hand from the Actions tab runs only the build job, which tests every platform without publishing anything.

## Scripts

| Script | Purpose |
|---|---|
| `scripts/build.sh <version> <os-arch>...` | builds release archives into `dist/` |
| `scripts/package-npm.sh <version> <dist>` | builds the npm packages from release archives |
| `scripts/package-vscode.sh <version> <dist>` | builds one VS Code package per platform |
| `scripts/package-intellij.sh <version> <dist>` | builds the IntelliJ plugin with every binary |
| `scripts/update-formula.sh <version> <formula>` | writes the Homebrew formula from a release's checksums |
| `scripts/set-secrets.sh [file] [owner/repo]` | stores the release secrets from `.envrc` or another file |

## Known issues

These came up during the first releases and are handled by the current setup.

- **npm rejects the unscoped name `pgfmt`** as too similar to an existing package, so the main package is `@sirmspencer/pgfmt`. The command it installs is still `pgfmt`.
- **npm versions cannot be reused**, even after a package is unpublished. A failed publish is fixed with a new version, not by publishing the same version again.
- **New npm packages can take a few minutes** to appear in the registry after a successful publish.
- **Open VSX scans new uploads** before listing them, so a version can be missing from open-vsx.org for a while after the workflow reports it as published.
- **GitHub retires runner images.** The `macos-14` image stopped working and was replaced with `macos-latest`.
- **Windows checkouts convert line endings** unless told otherwise. `.gitattributes` keeps every file at LF, which the golden tests depend on.
- **The JetBrains plugin verifier reports internal and deprecated APIs** on upload. The plugin uses only public APIs that exist in IntelliJ 2025.1, the oldest version it supports.
