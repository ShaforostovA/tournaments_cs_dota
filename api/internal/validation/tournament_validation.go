package validation

import "strings"

func IsSupportedGame(game string) bool {
	normalizedGame := strings.ToLower(strings.TrimSpace(game))

	return normalizedGame == "cs2" || normalizedGame == "dota2"
}

func IsValidTournamentName(name string) bool {
	trimmedName := strings.TrimSpace(name)

	return len(trimmedName) >= 3 && len(trimmedName) <= 100
}

func IsValidTeamLimit(teamLimit int) bool {
	return teamLimit >= 2 && teamLimit <= 64 && teamLimit%2 == 0
}
