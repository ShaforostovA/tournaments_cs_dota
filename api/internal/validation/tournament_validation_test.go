package validation

import "testing"

func TestIsSupportedGame(t *testing.T) {
	tests := []struct {
		name     string
		game     string
		expected bool
	}{
		{name: "CS2 is supported", game: "cs2", expected: true},
		{name: "Dota 2 is supported", game: "dota2", expected: true},
		{name: "Game name is normalized", game: " CS2 ", expected: true},
		{name: "Unsupported game", game: "valorant", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsSupportedGame(test.game)

			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestIsValidTournamentName(t *testing.T) {
	tests := []struct {
		name           string
		tournamentName string
		expected       bool
	}{
		{name: "Valid tournament name", tournamentName: "Summer Cup", expected: true},
		{name: "Too short tournament name", tournamentName: "CS", expected: false},
		{name: "Only spaces", tournamentName: "   ", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsValidTournamentName(test.tournamentName)

			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestIsValidTeamLimit(t *testing.T) {
	tests := []struct {
		name      string
		teamLimit int
		expected  bool
	}{
		{name: "Valid team limit", teamLimit: 16, expected: true},
		{name: "Too few teams", teamLimit: 1, expected: false},
		{name: "Too many teams", teamLimit: 128, expected: false},
		{name: "Odd team limit", teamLimit: 7, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsValidTeamLimit(test.teamLimit)

			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}
