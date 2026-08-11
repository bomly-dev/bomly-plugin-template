package plugin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	sdk "github.com/bomly-dev/bomly-sdk"
	"github.com/bomly-dev/bomly-sdk/conformance"
	"go.uber.org/zap"
)

// testHost is a minimal HostContext for unit tests.
type testHost struct {
	config json.RawMessage
}

func (h testHost) Logger() *zap.Logger                 { return zap.NewNop() }
func (h testHost) HTTPClient() *sdk.HTTPClientProvider { return nil }
func (h testHost) Runtime() sdk.RuntimeInfo {
	return sdk.RuntimeInfo{Execution: sdk.ExecutionEmbedded}
}

func (h testHost) DecodeConfig(v any) error {
	payload := h.config
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	return json.Unmarshal(payload, v)
}

func newMatcher(t *testing.T, config json.RawMessage) sdk.Matcher {
	t.Helper()
	module := Module()
	matcher, err := module.Matcher.New(context.Background(), testHost{config: config})
	if err != nil {
		t.Fatalf("construct matcher: %v", err)
	}
	return matcher
}

func loadFixtureRegistry(t *testing.T) *sdk.PackageRegistry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", "registry.json"))
	if err != nil {
		t.Fatalf("read fixture registry: %v", err)
	}
	registry := sdk.NewPackageRegistry()
	if err := json.Unmarshal(data, registry); err != nil {
		t.Fatalf("decode fixture registry: %v", err)
	}
	if registry.Len() == 0 {
		t.Fatal("fixture registry is empty")
	}
	return registry
}

func TestMatchAnnotatesFullRegistry(t *testing.T) {
	matcher := newMatcher(t, nil)
	registry := loadFixtureRegistry(t)

	result, err := matcher.Match(context.Background(), sdk.MatchRequest{Registry: registry})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if result.Registry == nil {
		t.Fatal("baseline path must return the full registry")
	}
	if len(result.PackageUpdates) != 0 {
		t.Fatal("baseline path must not return package updates")
	}
	for _, pkg := range result.Registry.All() {
		if pkg.Metadata[MetadataKey] != "hello from the template" {
			t.Fatalf("package %s missing default greeting annotation, got %v", pkg.PURL, pkg.Metadata[MetadataKey])
		}
	}
	if result.MatcherStats.MatchedPackages != registry.Len() {
		t.Fatalf("expected %d matched packages, got %d", registry.Len(), result.MatcherStats.MatchedPackages)
	}
}

func TestMatchReturnsPackageUpdatesWithConfiguredGreeting(t *testing.T) {
	matcher := newMatcher(t, json.RawMessage(`{"greeting":"custom greeting"}`))
	registry := loadFixtureRegistry(t)

	result, err := matcher.Match(context.Background(), sdk.MatchRequest{
		Registry:             registry,
		AcceptPackageUpdates: true,
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if result.Registry != nil {
		t.Fatal("delta path must not return the full registry")
	}
	if len(result.PackageUpdates) != registry.Len() {
		t.Fatalf("expected %d package updates, got %d", registry.Len(), len(result.PackageUpdates))
	}

	merged := sdk.ApplyPackageUpdates(registry, result.PackageUpdates)
	for _, pkg := range merged.All() {
		if pkg.Metadata[MetadataKey] != "custom greeting" {
			t.Fatalf("package %s missing configured greeting after merge, got %v", pkg.PURL, pkg.Metadata[MetadataKey])
		}
	}
}

func TestMatchEmptyRequest(t *testing.T) {
	matcher := newMatcher(t, nil)
	result, err := matcher.Match(context.Background(), sdk.MatchRequest{})
	if err != nil {
		t.Fatalf("Match with no registry: %v", err)
	}
	if result.Registry != nil || len(result.PackageUpdates) != 0 {
		t.Fatal("empty request must produce an empty result")
	}
}

// TestConformance runs the SDK conformance suite against the module,
// including the bomly-plugin.json identity cross-check.
func TestConformance(t *testing.T) {
	conformance.Test(t, conformance.Config{
		Module:       Module(),
		ManifestPath: filepath.Join("..", "bomly-plugin.json"),
		SampleConfig: json.RawMessage(`{"greeting":"conformance greeting"}`),
	})
}
