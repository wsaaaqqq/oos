package main

import "strings"

type Filter struct {
	Include []string
	Exclude []string
}

func ParseKeys(input string) Filter {
	var f Filter
	for _, part := range strings.Fields(input) {
		if strings.HasPrefix(part, "!") && len(part) > 1 {
			f.Exclude = append(f.Exclude, strings.ToLower(part[1:]))
		} else {
			f.Include = append(f.Include, strings.ToLower(part))
		}
	}
	return f
}

func MatchSession(s Session, f Filter, msgTexts []string) bool {
	for _, kw := range f.Include {
		if !sessionContains(s, kw) && !msgsContain(msgTexts, kw) {
			return false
		}
	}
	for _, kw := range f.Exclude {
		if sessionContains(s, kw) || msgsContain(msgTexts, kw) {
			return false
		}
	}
	return true
}

func sessionContains(s Session, kw string) bool {
	return strings.Contains(strings.ToLower(s.Title), kw) ||
		strings.Contains(strings.ToLower(s.Slug), kw) ||
		strings.Contains(strings.ToLower(s.Directory), kw) ||
		strings.Contains(strings.ToLower(s.ModelID), kw) ||
		strings.Contains(strings.ToLower(s.Agent), kw) ||
		strings.Contains(strings.ToLower(s.FirstUserMsg), kw)
}

func msgsContain(msgs []string, kw string) bool {
	if msgs == nil {
		return false
	}
	for _, m := range msgs {
		if strings.Contains(strings.ToLower(m), kw) {
			return true
		}
	}
	return false
}

func FilterSessions(sessions []Session, f Filter, msgMap map[string][]string) []Session {
	if len(f.Include) == 0 && len(f.Exclude) == 0 {
		return sessions
	}

	var result []Session
	for _, s := range sessions {
		msgs := msgMap[s.ID]
		if MatchSession(s, f, msgs) {
			result = append(result, s)
		}
	}
	return result
}
