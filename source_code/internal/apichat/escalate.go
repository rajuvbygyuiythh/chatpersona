package apichat

// Snap hygiene + the pre-share interest spike.
//
// The account stays alive longer when the bot itself never names the other
// app: "snap" may only appear in the final share line, which Go writes from
// sharesnap.txt. Everything the model writes is checked with mentionsSnap(),
// and right before that share line the bot sends one hand-written tease so the
// stranger is already hooked when the handle lands.

import (
	"regexp"
	"strings"
)

// snapMentionRe catches every way the model could leak the share early. The
// share line itself is appended after this check, so it is never filtered.
var snapMentionRe = regexp.MustCompile(`(?i)(\bsnap(chat)?\b|\bsnp\b|\bs\.?\s?c\.?\b|\badd me\b|\badd my\b|\badd u\b|\badd you\b)`)

// mentionsSnap reports whether a line names the other app or asks to be added.
func mentionsSnap(line string) bool {
	return snapMentionRe.MatchString(strings.ToLower(line))
}

// hypeReserveLine is the last resort when every tease was already used in the
// same conversation. It is still snap-free and still a statement.
const hypeReserveLine = "u have no idea what u do to me"

// hypeHotLines is the default tease deck: the strangers who already flirt or
// talk dirty get the boldest version. No app names, no "add", no questions.
var hypeHotLines = []string{
	"dont text me if u cant handle me tonight",
	"lick me baby, im serious",
	"wanna eat something, im the whole meal",
	"behave or im gonna ruin u",
	"u would fold so fast with me",
	"come here before i change my mind",
	"i bite btw",
	"dont act shy now, u started this",
	"this gets so much worse after midnight",
	"bet u cant keep up with me",
	"im not even trying yet",
	"one night with me and ur done",
	"say u want it and ill believe u",
	"u talk big for someone who'd beg",
	"im the best thing happening to u tonight",
	"dont make me wait, i get mean",
	"ur gonna think about me all night",
	"come get it before i change my mind",
	"im trouble when im bored",
	"u have no idea how bad i want u",
}

// hypePlayfulLines fits jokes and light teasing: same confidence, less heat.
var hypePlayfulLines = []string{
	"ur kinda trouble, i like it",
	"dont get attached, im a lot",
	"u seem like u'd survive one night with me, maybe",
	"im trouble and u know it",
	"careful, im addictive",
	"u have no idea what ur getting into",
	"i hope u can keep up",
	"one coffee and ur obsessed, trust me",
}

// hypeSoftLines is for cold, one-word strangers: a spark, not a flood.
var hypeSoftLines = []string{
	"ur actually kinda cute when u try",
	"dont make me like u, thats dangerous",
	"u might be my favorite person tonight",
	"keep talking, im starting to like u",
}

// hypePoolFor picks the deck that matches the stranger's energy.
func hypePoolFor(d *Director, p Plan) []string {
	switch {
	case d.Profile.Kind == KindJoke || p.Mood == "fun/joke":
		return hypePlayfulLines
	case d.Tier == tierCold && d.Score < 3:
		return hypeSoftLines
	}
	return hypeHotLines
}

// pickHypeLine returns one tease that has never been sent in this conversation
// and does not repeat the model lines we are about to send.
func (e *Engine) pickHypeLine(s *Session, p Plan, about []string) string {
	d := &s.Director
	candidates := hypeCandidates(hypePoolFor(d, p), s, about)
	if len(candidates) == 0 {
		candidates = hypeCandidates(hypeHotLines, s, about)
	}
	if len(candidates) == 0 {
		return ""
	}
	pick := candidates[d.roll(len(candidates))]
	s.UsedHypeLines = append(s.UsedHypeLines, pick)
	return pick
}

func hypeCandidates(pool []string, s *Session, about []string) []string {
	used := make(map[string]bool, len(s.UsedHypeLines))
	for _, line := range s.UsedHypeLines {
		used[line] = true
	}
	out := make([]string, 0, len(pool))
	for _, line := range pool {
		if line == "" || used[line] || mentionsSnap(line) {
			continue
		}
		if score, _ := mostSimilar(line, s.History, 8); score >= repeatThreshold {
			continue
		}
		clash := false
		for _, other := range about {
			if similarity(line, other) >= repeatThreshold {
				clash = true
				break
			}
		}
		if clash {
			continue
		}
		out = append(out, line)
	}
	return out
}
