package bookmark

import "strings"

func MatchesAnyTag(tags []string, filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	for _, want := range strings.Split(filter, ",") {
		want = strings.TrimSpace(want)
		for _, have := range tags {
			if strings.EqualFold(want, have) {
				return true
			}
		}
	}
	return false
}
