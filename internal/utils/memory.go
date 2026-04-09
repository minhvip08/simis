package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseMemorySize parses a human-readable memory size string into bytes.
// Accepts plain integers (bytes) or strings with unit suffixes:
// b, k/kb, m/mb, g/gb, t/tb (case-insensitive).
// Examples: "0", "1024", "100mb", "1gb", "512KB".
func ParseMemorySize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "0" {
		return 0, nil
	}

	units := []struct {
		suffix     string
		multiplier int64
	}{
		{"tb", 1024 * 1024 * 1024 * 1024},
		{"gb", 1024 * 1024 * 1024},
		{"mb", 1024 * 1024},
		{"kb", 1024},
		{"t", 1024 * 1024 * 1024 * 1024},
		{"g", 1024 * 1024 * 1024},
		{"m", 1024 * 1024},
		{"k", 1024},
		{"b", 1},
	}

	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			numStr := strings.TrimSuffix(s, u.suffix)
			num, err := strconv.ParseInt(strings.TrimSpace(numStr), 10, 64)
			if err != nil || num < 0 {
				return 0, fmt.Errorf("invalid memory size: %q", s)
			}
			return num * u.multiplier, nil
		}
	}

	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil || num < 0 {
		return 0, fmt.Errorf("invalid memory size: %q", s)
	}
	return num, nil
}
