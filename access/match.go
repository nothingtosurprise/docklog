package access

import "regexp"

func MatchesAnyPattern(patterns []string, identifiers ...string) bool {
	for _, id := range identifiers {
		for _, p := range patterns {
			if matched, _ := regexp.MatchString(p, id); matched {
				return true
			}
		}
	}
	return false
}

func PodAllowed(patterns []string, namespace, podName string) bool {
	return MatchesAnyPattern(patterns, podName, namespace+"/"+podName)
}

func ResourceAllowed(patterns []string, namespace, name string) bool {
	return MatchesAnyPattern(patterns, name, namespace+"/"+name)
}

func NamespaceVisible(patterns []string, namespace string) bool {
	return MatchesAnyPattern(patterns, namespace, namespace+"/placeholder")
}
