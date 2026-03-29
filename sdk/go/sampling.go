package xray

import (
	"crypto/md5"
	"encoding/binary"
)

// SamplingConfig defines outcome-based sampling rates.
//
// Values must be between 0.0 and 1.0. Use "*" as a default catch-all rate.
// Example: map[string]float64{"rejected": 1.0, "accepted": 0.01, "*": 0.05}
type SamplingConfig struct {
	OutcomeRates map[string]float64
}

// ShouldSample deterministically decides if a decision should be stored.
func (s *SamplingConfig) ShouldSample(outcome, itemID string) bool {
	if s == nil {
		return true
	}
	rate, ok := s.OutcomeRates[outcome]
	if !ok {
		rate, ok = s.OutcomeRates["*"]
		if !ok {
			rate = 0.01
		}
	}
	if rate >= 1.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}
	hash := md5.Sum([]byte(itemID))
	v := binary.BigEndian.Uint32(hash[:4])
	pct := float64(v%10000) / 10000.0
	return pct < rate
}
