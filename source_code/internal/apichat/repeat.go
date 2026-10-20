package apichat

// Anti-repetition helpers: near-duplicate lines, repeated openers and repeated
// questions are cheaper to detect here than to hope the model behaves.

import (
	"strings"
)

// normalizeForCompare lowercases and strips punctuation/emoji.
func normalizeForCompare(s string) string {
	s = strings.ToLower(s)
	s = compareClean.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

// shingles builds the unigram + bigram set used for Jaccard similarity.
func shingles(s string) map[string]bool {
	words := strings.Fields(normalizeForCompare(s))
	set := make(map[string]bool, len(words)*2)
	for i, w := range words {
		set["w:"+w] = true
		if i+1 < len(words) {
			set["b:"+w+" "+words[i+1]] = true
		}
	}
	return set
}

// similarity is a 0..1 repeat score. Word overlap carries most of the weight;
// two lines that start with the same two words get a floor, because "ok fine
// lol" and "ok fine lmao" are the same reply even though one word changed.
func similarity(a, b string) float64 {
	if normalizeForCompare(a) == normalizeForCompare(b) {
		return 1
	}
	wordsA, wordsB := wordsOf(a), wordsOf(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	score := 0.75*jaccard(unigrams(a), unigrams(b)) + 0.25*jaccard(shingles(a), shingles(b))
	shortest := len(wordsA)
	if len(wordsB) < shortest {
		shortest = len(wordsB)
	}
	prefix := sharedPrefix(wordsA, wordsB)
	if prefix >= 2 {
		if floor := 0.6 + 0.4*float64(prefix)/float64(shortest); floor > score {
			score = floor
		}
	}
	return score
}

// wordsOf is the normalized word list of a line.
func wordsOf(s string) []string {
	return strings.Fields(normalizeForCompare(s))
}

// sharedPrefix counts the identical leading words of two lines.
func sharedPrefix(a, b []string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

// unigrams is the set of plain words used for the weighted word overlap.
func unigrams(s string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(normalizeForCompare(s)) {
		set["w:"+w] = true
	}
	return set
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// lastBotLines returns the newest bot lines first.
func lastBotLines(history []Message, n int) []string {
	out := make([]string, 0, n)
	for i := len(history) - 1; i >= 0 && len(out) < n; i-- {
		if history[i].Role == "assistant" && strings.TrimSpace(history[i].Content) != "" {
			out = append(out, history[i].Content)
		}
	}
	return out
}

// mostSimilar returns the closest earlier bot line and its score.
func mostSimilar(line string, history []Message, n int) (float64, string) {
	best, match := 0.0, ""
	for _, old := range lastBotLines(history, n) {
		if score := similarity(line, old); score > best {
			best, match = score, old
		}
	}
	return best, match
}

// openerOf returns the first two normalized words.
func openerOf(line string) string {
	words := strings.Fields(normalizeForCompare(line))
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	default:
		return words[0] + " " + words[1]
	}
}

// repeatedOpener reports an identical opening to one of the last two lines.
func repeatedOpener(line string, history []Message) bool {
	opener := openerOf(line)
	if opener == "" {
		return false
	}
	for i, old := range lastBotLines(history, 2) {
		if i >= 2 {
			break
		}
		if openerOf(old) == opener {
			return true
		}
	}
	return false
}

// questionKeys names the personal questions present in a line.
func questionKeys(line string) []string {
	keys := make([]string, 0, 3)
	for _, fact := range askedFacts(line) {
		switch fact {
		case FactSnap:
			// mentioning snap is fine; asking is handled by the ledger.
			continue
		default:
			keys = append(keys, string(fact))
		}
	}
	low := strings.ToLower(line)
	if howAreYouRe.MatchString(low) {
		keys = append(keys, "howareyou")
	}
	if doingAskRe.MatchString(low) {
		keys = append(keys, "doing")
	}
	return keys
}

// repeatedQuestion reports the same question in the previous two bot lines.
func repeatedQuestion(line string, history []Message) bool {
	keys := questionKeys(line)
	if len(keys) == 0 {
		return false
	}
	previous := map[string]bool{}
	for _, old := range lastBotLines(history, 2) {
		for _, key := range questionKeys(old) {
			previous[key] = true
		}
	}
	for _, key := range keys {
		if previous[key] {
			return true
		}
	}
	return false
}

// lastBotLine returns the newest bot line, or "".
func lastBotLine(history []Message) string {
	lines := lastBotLines(history, 1)
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

// countQuestions counts question marks across the generated lines.
func countQuestions(lines []string) int {
	total := 0
	for _, line := range lines {
		total += strings.Count(line, "?")
	}
	return total
}
