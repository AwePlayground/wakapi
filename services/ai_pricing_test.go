package services

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestEstimateAICostTracksUnpricedTokens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pricing.json")
	data := `{"updated_at":"2026-10-28T00:00:00Z","aliases":{"Opus":"claude-opus"},"models":{"claude-opus":{"input":0.000005,"cached_input":0.0000005,"output":0.000025,"source":"litellm"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadAIPriceCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	groups := []*AIActivityGroup{
		{Name: "Opus", AIActivityTotals: AIActivityTotals{InputTokens: 100, CachedInputTokens: 200, OutputTokens: 10}},
		{Name: "Unknown", AIActivityTotals: AIActivityTotals{InputTokens: 50, OutputTokens: 5}},
	}
	estimate := EstimateAICost(groups, catalog)
	want := 100*0.000005 + 200*0.0000005 + 10*0.000025
	if math.Abs(estimate.EstimatedUSD-want) > 1e-12 || estimate.PricedTokens != 310 || estimate.UnpricedTokens != 55 {
		t.Fatalf("wrong estimate: %+v, want USD %f", estimate, want)
	}
	if len(estimate.Models) != 2 || !estimate.Models[0].Mapped || estimate.Models[1].Mapped {
		t.Fatalf("wrong model coverage: %+v", estimate.Models)
	}
}
