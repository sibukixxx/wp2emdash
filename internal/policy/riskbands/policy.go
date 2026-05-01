package riskbands

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sibukixxx/wp2emdash/internal/domain/score"
)

//go:embed default.json
var defaultPolicyJSON []byte

type Policy struct {
	Bands []Band `json:"bands"`
}

type Band struct {
	MaxScore int         `json:"max_score"`
	Level    score.Level `json:"level"`
	Estimate string      `json:"estimate"`
}

func Load(path string) (Policy, error) {
	if path == "" {
		return decode(defaultPolicyJSON)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read risk bands policy: %w", err)
	}
	return decode(raw)
}

func (p Policy) Classify(rawScore int) (score.Level, string, error) {
	for _, band := range p.Bands {
		if band.MaxScore < 0 || rawScore <= band.MaxScore {
			return band.Level, band.Estimate, nil
		}
	}
	return "", "", fmt.Errorf("no risk band matched score %d", rawScore)
}

func decode(raw []byte) (Policy, error) {
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return Policy{}, fmt.Errorf("decode risk bands policy: %w", err)
	}
	if len(p.Bands) == 0 {
		return Policy{}, fmt.Errorf("risk bands policy has no bands")
	}
	return p, nil
}
