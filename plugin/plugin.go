// Package plugin implements the template's example component: a Bomly
// MATCHER that annotates every package in the registry with a greeting.
//
// A matcher is the simplest Bomly plugin role to run end to end: it receives
// the resolved dependency graph plus the PURL-keyed package registry, and
// returns package enrichment (licenses, vulnerabilities, lifecycle data — or,
// here, a demonstration metadata annotation).
//
// To turn this template into a real plugin, keep the structure and replace:
//   - the descriptor identity (name, capabilities, config schema),
//   - the Config fields your matcher needs,
//   - the Match implementation.
package plugin

import (
	"context"
	"fmt"

	sdk "github.com/bomly-dev/bomly-sdk"
)

// Name is the plugin's identity. It MUST equal the "id" field in
// bomly-plugin.json — Bomly refuses to load a plugin whose manifest id and
// runtime descriptor name disagree.
const Name = "bomly.template.matcher"

// MetadataKey is the Package.Metadata key this matcher writes. Namespace your
// metadata keys with your plugin id so they never collide with other
// matchers.
const MetadataKey = "bomly.template.greeting"

// Config is the matcher's typed configuration block. Users set it in their
// Bomly config under plugins.matchers.<id>:
//
//	plugins:
//	  matchers:
//	    bomly.template.matcher:
//	      greeting: "hi from my config"
//
// The struct tags drive everything: `json` names the key, `doc` documents it,
// and `default` supplies the value when the user sets nothing. The same
// struct is advertised to Bomly as a JSON Schema via sdk.MustConfigSchemaFor,
// so users get validation and documentation for free.
type Config struct {
	Greeting string `json:"greeting" doc:"Annotation text added to package metadata" default:"hello from the template"`
}

// Matcher is the component. Embedding sdk.BaseMatcher supplies default
// Ready/Applicable implementations (always ready, always applicable), so this
// struct only has to implement Descriptor and Match. Override Ready when your
// matcher depends on something that can be missing (a token, a reachable
// endpoint) and Applicable when it should only run for certain ecosystems.
type Matcher struct {
	sdk.BaseMatcher
	config Config
}

// descriptor is the matcher's static registration data.
func descriptor() sdk.MatcherDescriptor {
	return sdk.MatcherDescriptor{
		Name:        Name,
		DisplayName: "Template Matcher",
		// CapabilityPackageUpdates tells Bomly this matcher can return
		// package-update deltas (only the packages it touched) instead of
		// echoing the full registry. Hosts that do not understand deltas
		// simply never set AcceptPackageUpdates, and the matcher falls back
		// to the full-registry baseline below.
		Capabilities: []string{sdk.CapabilityPackageUpdates},
		// Advertise the Config shape so `bomly plugin info` and the config
		// schema tooling can document and validate the block.
		ConfigSchema: sdk.MustConfigSchemaFor(Config{}),
	}
}

// Descriptor identifies the matcher to Bomly.
func (m *Matcher) Descriptor() sdk.MatcherDescriptor { return descriptor() }

// Match is the matcher's action. It runs once per scan with the full package
// registry and returns enrichment for those packages.
func (m *Matcher) Match(_ context.Context, req sdk.MatchRequest) (sdk.MatchResult, error) {
	stats := sdk.MatcherStats{Name: Name, DisplayName: "Template Matcher"}
	if req.Registry == nil {
		// Nothing to enrich. Returning an empty result is fine.
		return sdk.MatchResult{MatcherStats: stats}, nil
	}

	if req.AcceptPackageUpdates {
		// Delta path: the host understands package updates, so return ONLY
		// the packages we touched. Each update is a sparse Package carrying
		// the PURL (the merge key) plus the new data; the host merges it
		// into its registry.
		updates := make([]*sdk.Package, 0, req.Registry.Len())
		for _, pkg := range req.Registry.All() {
			update := &sdk.Package{Coordinates: sdk.Coordinates{PURL: pkg.PURL}}
			update.Metadata = map[string]any{MetadataKey: m.config.Greeting}
			updates = append(updates, update)
		}
		stats.MatchedPackages = len(updates)
		return sdk.MatchResult{PackageUpdates: updates, MatcherStats: stats}, nil
	}

	// Baseline path (protocol v1): annotate the registry in place and echo
	// the full registry back.
	for _, pkg := range req.Registry.All() {
		if pkg.Metadata == nil {
			pkg.Metadata = make(map[string]any, 1)
		}
		pkg.Metadata[MetadataKey] = m.config.Greeting
		stats.MatchedPackages++
	}
	return sdk.MatchResult{Registry: req.Registry, MatcherStats: stats}, nil
}

// Module packages the matcher for both execution modes: Bomly can embed it
// in-process or serve it as a managed plugin subprocess (see
// cmd/bomly-plugin-template). The constructor receives a HostContext — the
// only channel to host services (logger, HTTP client, runtime info, config).
func Module() sdk.Module {
	return sdk.Module{
		Kind: sdk.PluginKindMatcher,
		Matcher: &sdk.MatcherModule{
			Descriptor: descriptor(),
			New: func(_ context.Context, host sdk.HostContext) (sdk.Matcher, error) {
				matcher := &Matcher{}
				// DecodeConfig fills Config from the user's
				// plugins.matchers.<id> block; `default` tags apply to
				// fields the user left unset only if you pre-populate them,
				// so set defaults explicitly before decoding.
				matcher.config = Config{Greeting: "hello from the template"}
				if err := host.DecodeConfig(&matcher.config); err != nil {
					return nil, fmt.Errorf("decode %s config: %w", Name, err)
				}
				host.Logger().Debug("template matcher configured")
				return matcher, nil
			},
		},
	}
}
