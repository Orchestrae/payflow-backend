package utils

import "testing"

func TestNameMatchScore(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		resolved string
		minScore float64
	}{
		{"exact match", "John Doe", "John Doe", 1.0},
		{"case insensitive", "JOHN DOE", "john doe", 1.0},
		{"reversed order", "Doe John", "John Doe", 1.0},
		{"extra middle name", "John Doe", "John Michael Doe", 0.6},
		{"abbreviation match", "Tolu Thomas", "Toluwase Thomas", 0.6},
		{"completely different", "John Doe", "Jane Smith", 0.0},
		{"partial match", "Adebayo Ogunlesi", "Ogunlesi Adebayo Oluwatobi", 0.6},
		{"hyphenated name", "Ngozi Obi-Uchendu", "Ngozi Obi Uchendu", 0.6},
		{"single name match", "Chukwuemeka", "Chukwuemeka Obi", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := NameMatchScore(tt.provided, tt.resolved)
			if score < tt.minScore {
				t.Errorf("NameMatchScore(%q, %q) = %f, want >= %f", tt.provided, tt.resolved, score, tt.minScore)
			}
		})
	}
}

func TestNamesMatch(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		resolved string
		want     bool
	}{
		{"exact", "Ade Bayo", "Ade Bayo", true},
		{"reversed", "Bayo Ade", "Ade Bayo", true},
		{"completely different", "John Doe", "Jane Smith", false},
		{"close enough", "Tolu Thomas", "Toluwase Thomas", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NamesMatch(tt.provided, tt.resolved)
			if got != tt.want {
				t.Errorf("NamesMatch(%q, %q) = %v, want %v (score: %f)",
					tt.provided, tt.resolved, got, tt.want, NameMatchScore(tt.provided, tt.resolved))
			}
		})
	}
}
