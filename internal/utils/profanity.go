package utils

import (
	"regexp"
	"strings"
)

var profaneWords = map[string]bool{
	"kerfuffle": true,
	"sharbert":  true,
	"fornax":    true,
}

func CleanProfanity(s string) string {
	re := regexp.MustCompile(`(\S+|\s+)`)
	parts := re.FindAllString(s, -1)

	var result strings.Builder
	for _, part := range parts {
		if len(strings.TrimSpace(part)) == 0 {
			result.WriteString(part)
			continue
		}

		isWord := true
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				isWord = false
				break
			}
		}

		if isWord && profaneWords[strings.ToLower(part)] {
			result.WriteString("****")
		} else {
			result.WriteString(part)
		}
	}
	return result.String()
}