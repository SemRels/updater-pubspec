# updater-pubspec

[![Latest Release](https://img.shields.io/github/v/release/SemRels/updater-pubspec?label=version&color=blue)](https://github.com/SemRels/updater-pubspec/releases/latest)

Updates the `version:` field in `pubspec.yaml` for Dart and Flutter projects.

This plugin is distributed as the standalone Go binary `semrel-plugin-updater-pubspec`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Installation

### Binary

```bash
go install github.com/SemRels/updater-pubspec/cmd/plugin@latest
```

### Docker

Pre-built, multi-platform images (linux/amd64, linux/arm64) are published to the GitHub Container Registry on every release:

```bash
docker pull ghcr.io/semrels/updater-pubspec:latest
```

Images are signed with [cosign](https://github.com/sigstore/cosign) and include a full SBOM attestation. Verify the signature:

```bash
cosign verify ghcr.io/semrels/updater-pubspec:latest \
  --certificate-identity-regexp 'https://github.com/SemRels/updater-pubspec/.github/workflows/release.yml.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Configuration (`.semrel.yaml`)

```yaml
plugins:
  - name: updater-pubspec
    path: ~/.semrel/plugins/semrel-plugin-updater-pubspec
    env:
      SEMREL_PLUGIN_PUBSPEC_PATH: "packages/app/pubspec.yaml"
      SEMREL_PLUGIN_PUBSPEC_BUILD_STRATEGY: "increment"
```

## `SEMREL_PLUGIN_*` variables

| Name | Required | Description | Default |
| --- | --- | --- | --- |
| `SEMREL_PLUGIN_PUBSPEC_PATH` | Optional | Path to the `pubspec.yaml` file to update. | `./pubspec.yaml` |
| `SEMREL_PLUGIN_PUBSPEC_BUILD_STRATEGY` | Optional | Build suffix strategy: `none`, `increment`, or `release-count`. | `none` |
| `SEMREL_PLUGIN_PUBSPEC_BUILD_NUMBER` | Optional | Explicit build number override. When set it always wins and writes `SEMREL_VERSION+<number>`. | unset |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_VERSION` | Resolved release version for the current run. |
| `SEMREL_NEXT_VERSION` | Next version computed by semrel for the release. |
| `SEMREL_DRY_RUN` | Whether semrel is running in dry-run mode. |
| `SEMREL_RELEASE_COUNT` | Optional release-count input if semrel or the caller provides it. |
| `SEMREL_BUILD_NUMBER` | Optional build-number input if CI or semrel provides it. |

## Build number strategies

The plugin always strips a leading `v` from `SEMREL_VERSION` / `SEMREL_NEXT_VERSION` before writing the semver part.

### `none` (default)

Removes any existing Flutter build suffix and writes only the bare semver.

Before:

```yaml
version: 1.4.0+42
```

After `SEMREL_VERSION=v1.5.0`:

```yaml
version: 1.5.0
```

### `increment`

Reads the current top-level `version:` line, increments the existing `+build` number, and preserves the rest of the file unchanged. If no build suffix exists, the plugin starts at `+1`. If the existing suffix is malformed, the plugin logs a warning to stderr and starts fresh at `+1`.

Before:

```yaml
version: 1.4.0+42
```

After `SEMREL_VERSION=v1.5.0`:

```yaml
version: 1.5.0+43
```

### `release-count`

Use this when you want the Flutter build suffix to mirror a monotonically increasing release or CI counter.

Priority order:

1. `SEMREL_PLUGIN_PUBSPEC_BUILD_NUMBER`
2. `SEMREL_RELEASE_COUNT`
3. `SEMREL_BUILD_NUMBER`
4. Fallback to `existing build + 1` with a warning if none of the above are set

Semrel's current ecosystem does not appear to expose a dedicated release-count environment variable yet, so callers commonly pass an external counter (for example `GITHUB_RUN_NUMBER`) through `SEMREL_PLUGIN_PUBSPEC_BUILD_NUMBER`.

Before:

```yaml
version: 1.4.0+42
```

After with `SEMREL_VERSION=v1.5.0` and `SEMREL_PLUGIN_PUBSPEC_BUILD_NUMBER=108`:

```yaml
version: 1.5.0+108
```

## Dry runs

With `SEMREL_DRY_RUN=true`, the plugin validates the file, computes the exact new `version:` line, prints it, and leaves `pubspec.yaml` untouched.

## Example behavior

The plugin performs a surgical line-based replacement of the top-level `version:` key so comments, indentation, dependency blocks, and the rest of the YAML file remain untouched.

## Local validation

```bash
go test ./...
go vet ./...
go build ./...
```

## License

Apache-2.0
