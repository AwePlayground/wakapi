package services

import (
	"encoding/json"
	"os"
)

type AIModelPrice struct {
	Input       *float64 `json:"input"`
	CachedInput *float64 `json:"cached_input"`
	Output      *float64 `json:"output"`
	Source      string   `json:"source"`
}

type AIPriceCatalog struct {
	UpdatedAt string                  `json:"updated_at"`
	Aliases   map[string]string       `json:"aliases"`
	Models    map[string]AIModelPrice `json:"models"`
}

type AIModelCost struct {
	Name           string
	CatalogModel   string
	Source         string
	EstimatedUSD   float64
	PricedTokens   int64
	UnpricedTokens int64
	Mapped         bool
	Assumed        bool
}

type AICostSummary struct {
	EstimatedUSD   float64
	PricedTokens   int64
	UnpricedTokens int64
	Models         []*AIModelCost
}

func LoadAIPriceCatalog(path string) (*AIPriceCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var catalog AIPriceCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	return &catalog, nil
}

// EstimateAICost uses the cached catalog only. Unknown models or token classes
// remain explicitly unpriced instead of silently contributing zero dollars.
func EstimateAICost(groups []*AIActivityGroup, catalog *AIPriceCatalog) AICostSummary {
	result := AICostSummary{}
	for _, group := range groups {
		model := &AIModelCost{Name: group.Name}
		name := group.Name
		if catalog != nil {
			if alias, found := catalog.Aliases[name]; found {
				name = alias
				model.Assumed = true
			}
			if price, found := catalog.Models[name]; found {
				model.Mapped = true
				model.CatalogModel = name
				model.Source = price.Source
				addCostClass(model, group.InputTokens, price.Input)
				addCostClass(model, group.CachedInputTokens, price.CachedInput)
				addCostClass(model, group.OutputTokens, price.Output)
			} else {
				model.UnpricedTokens = group.InputTokens + group.CachedInputTokens + group.OutputTokens
			}
		} else {
			model.UnpricedTokens = group.InputTokens + group.CachedInputTokens + group.OutputTokens
		}
		result.EstimatedUSD += model.EstimatedUSD
		result.PricedTokens += model.PricedTokens
		result.UnpricedTokens += model.UnpricedTokens
		result.Models = append(result.Models, model)
	}
	return result
}

func addCostClass(model *AIModelCost, tokens int64, rate *float64) {
	if tokens <= 0 {
		return
	}
	if rate == nil {
		model.UnpricedTokens += tokens
		return
	}
	model.EstimatedUSD += float64(tokens) * *rate
	model.PricedTokens += tokens
}
