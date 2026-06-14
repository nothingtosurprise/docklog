package access

import (
	"log"
	"regexp"
	"strings"

	"docklog/db"
)

const maxContainerPatternLen = 256

func GetAuthorizedPatterns(userID int) []string {
	var isRestricted bool
	var pattern string
	err := db.DB.QueryRow("SELECT is_restricted_access, allowed_containers FROM users WHERE id = ?", userID).Scan(&isRestricted, &pattern)
	if err != nil {
		return []string{"^$"}
	}

	if !isRestricted {
		return []string{".*"}
	}

	if pattern == "" {
		return []string{""}
	}

	rawPatterns := strings.Split(pattern, ",")
	var finalPatterns []string
	for _, p := range rawPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		if strings.HasPrefix(p, "^") || strings.HasSuffix(p, "$") {
			finalPatterns = appendValidatedPattern(finalPatterns, p)
			continue
		}

		regP := strings.ReplaceAll(p, "*", ".*")
		regP = strings.ReplaceAll(regP, "..*", ".*")

		if !strings.ContainsAny(regP, "()[]{}|") {
			if !strings.HasPrefix(regP, "^") {
				regP = "^" + regP
			}
			if !strings.HasSuffix(regP, "$") {
				regP = regP + "$"
			}
		}
		finalPatterns = appendValidatedPattern(finalPatterns, regP)
	}
	return finalPatterns
}

func appendValidatedPattern(patterns []string, regP string) []string {
	if len(regP) > maxContainerPatternLen {
		log.Printf("Skipping container pattern: exceeds %d characters", maxContainerPatternLen)
		return patterns
	}
	if _, err := regexp.Compile(regP); err != nil {
		log.Printf("Skipping invalid container pattern: %v", err)
		return patterns
	}
	return append(patterns, regP)
}
