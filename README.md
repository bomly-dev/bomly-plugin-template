# bomly-plugin-template

Template repository for building [Bomly](https://github.com/bomly-dev/bomly-cli) plugins in Go.

It ships a complete, working **matcher** plugin — the simplest role to run end to end — that annotates every package in the scan registry with a configurable greeting. Keep the structure, replace the logic, and you have a production-shaped plugin: typed config with a generated JSON schema, the package-updates delta protocol, unit tests, the SDK conformance suite, CI, and a release workflow that packages platform archives Bomly can install.

## Using the template

1. Click **Use this template** on GitHub (or `gh repo create my-plugin --template bomly-dev/bomly-plugin-template`).
2. Work through the rename checklist below.
3. Replace the matcher logic in `plugin/plugin.go` with your own.

### Rename checklist

- [ ] **Module path** — change `module github.com/bomly-dev/bomly-plugin-template` in `go.mod` to your repo path, and update the import in `cmd/bomly-plugin-template/main.go`.
- [ ] **Plugin id** — change `Name` in `plugin/plugin.go` (e.g. `com.example.my-matcher`) **and** the `id` field in `bomly-plugin.json`. They must be identical: Bomly refuses to load a plugin whose manifest id and runtime descriptor name disagree.
- [ ] **Binary name** — rename `cmd/bomly-plugin-template/` to `cmd/<your-binary>/`, update the `PLUGIN_NAME` env in `.github/workflows/release.yml`, and update every `entrypoint` value in `bomly-plugin.json`.
- [ ] **Manifest metadata** — update `name`, `version`, `description`, `homepage`, and `license` in `bomly-plugin.json`.
- [ ] **Role** — this template is a matcher. For a detector, auditor, or analyzer, change the module kind and descriptor accordingly; the how-to guides below walk through each role.

## Layout

```txt
plugin/                        The component: descriptor, typed Config, Match, Module()
cmd/bomly-plugin-template/     Binary entrypoint: sdk.ServeModule(plugin.Module())
bomly-plugin.json              Package manifest Bomly reads at install time
testdata/                      Fixture registry used by the unit tests
.github/workflows/             CI (test/vet/fmt/tidy) and dispatch-driven release
```

## Local development against Bomly

```sh
# Build the plugin binary
go build -o bin/bomly-plugin-template ./cmd/bomly-plugin-template

# Install it into Bomly as a dev plugin, enable it, and scan
bomly plugin install ./bin/bomly-plugin-template --dev
bomly plugin enable bomly.template.matcher
bomly scan --enrich
```

`bomly plugin list` / `bomly plugin info bomly.template.matcher` show the registration, including the config schema generated from the `Config` struct.

## Configuration

The matcher's config block lives under `plugins.matchers.<id>` in your Bomly config file:

```yaml
plugins:
  matchers:
    bomly.template.matcher:
      greeting: "hi from my config"
```

The `Config` struct in `plugin/plugin.go` is the single source of truth: its `json`/`doc`/`default` tags produce the JSON Schema advertised in the descriptor (`sdk.MustConfigSchemaFor`), and `HostContext.DecodeConfig` fills it at runtime in both embedded and managed execution.

## Testing

```sh
go test ./...
```

The tests cover the matcher's baseline (full registry) and delta (`PackageUpdates`) paths, plus `TestConformance`, which runs the SDK's [`conformance`](https://pkg.go.dev/github.com/bomly-dev/bomly-sdk/conformance) suite: module and descriptor validity, JSON round-trip stability, construction via `HostContext`, the Ready/Applicable contract (including prompt return on a cancelled context), the package-updates merge, and the manifest identity cross-check against `bomly-plugin.json`.

To probe a built binary over the real managed transport, add:

```go
conformance.ProbeBinary(t, "bin/bomly-plugin-template",
	conformance.WithModule(plugin.Module()))
```

## Releasing

1. Bump `version` in `bomly-plugin.json` and commit.
2. Run the **Release** workflow (workflow_dispatch) with the same version.

The workflow builds `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64`, packages each as `<name>_<version>_<os>_<arch>.tar.gz` (`.zip` for windows) containing the binary, `bomly-plugin.json`, `README.md`, and `LICENSE`, generates `SHA256SUMS`, and publishes a GitHub release. Users then install with:

```sh
bomly plugin install github.com/<owner>/<repo>@v<version>
```

## Versioning and compatibility

- Pin a **released** version of `github.com/bomly-dev/bomly-sdk` in `go.mod` and bump it deliberately.
- The plugin wire protocol (`bomly.plugin.v1`) is additive within v1: hosts and plugins negotiate optional features via descriptor `Capabilities` (e.g. `sdk.CapabilityPackageUpdates`), so older hosts and newer plugins keep working on the baseline behavior.
- Version your plugin with semver; keep `bomly-plugin.json` `version` and the release tag in lockstep.

## Security expectations

- **No secrets in logs, ever.** Use the `HostContext.Logger()`; never log tokens, credentials, or PII.
- **Explicit network.** Make outbound calls through `HostContext.HTTPClient()` (backed by `sdk.NewHTTPClientProviderFromEnv` in managed execution) so the host's proxy, TLS, and timeout policy applies. Document every endpoint your plugin talks to.
- Degrade gracefully: report unavailability through `Ready` instead of failing scans.

## Further reading

- [How to implement a matcher](https://github.com/bomly-dev/bomly-cli/blob/main/docs/plugins/how-to-implement-matcher.md)
- [How to implement a detector](https://github.com/bomly-dev/bomly-cli/blob/main/docs/plugins/how-to-implement-detector.md)
- [How to implement an auditor](https://github.com/bomly-dev/bomly-cli/blob/main/docs/plugins/how-to-implement-auditor.md)
- [Bomly plugin development guide](https://github.com/bomly-dev/bomly-cli/blob/main/docs/PLUGINS.md)

## License

Apache-2.0. See [LICENSE](LICENSE).
