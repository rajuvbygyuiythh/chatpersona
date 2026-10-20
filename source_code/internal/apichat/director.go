package apichat

// The director is the deterministic funnel brain: it reads the stranger's
// interest, decides the mood, and tells the model exactly what the next line
// must do. The model only writes the words. This is a production port of the
// design validated in apitest/PLAN.md.

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// Plan is one instruction for the model for a single bot reply.
type Plan struct {
	Mood        string
	Instruction string
	Double      bool
	ShareNow    bool // append the share line (with the username) after the lead-in
	Hype        bool // add one hand-written tease right before the share line
	EndAfter    bool // conversation ends after this reply
	Fixed       string
	Why         string
}

// Director holds the funnel state for one conversation. Exported fields are
// persisted with the session; the per-message flags are recomputed every turn.
type Director struct {
	Score          int      `json:"score"`
	Tier           string   `json:"tier"`
	HornyHits      int      `json:"horny_hits"`
	FlirtyHits     int      `json:"flirty_hits"`
	ColdStreak     int      `json:"cold_streak"`
	Underage       bool     `json:"underage"`
	UserAge        int      `json:"user_age,omitempty"`
	UserGender     string   `json:"user_gender,omitempty"`
	UserPlace      string   `json:"user_place,omitempty"`
	UserName       string   `json:"user_name,omitempty"`
	UserDoing      string   `json:"user_doing,omitempty"`
	AskedSnap      bool     `json:"asked_snap,omitempty"`
	Shared         bool     `json:"shared,omitempty"`
	SharedTurn     int      `json:"shared_turn,omitempty"`
	Ended          bool     `json:"ended,omitempty"`
	EndReason      string   `json:"end_reason,omitempty"`
	Nudges         int      `json:"nudges,omitempty"`
	Hesitated      bool     `json:"hesitated,omitempty"`
	HookPlanted    bool     `json:"hook_planted,omitempty"`
	WantsSnapEarly bool     `json:"wants_snap_early,omitempty"`
	LastMood       string   `json:"last_mood,omitempty"`
	LastDouble     int      `json:"last_double,omitempty"`
	NoQuestionAt   int      `json:"no_question_at,omitempty"`
	NonEngHits     int      `json:"non_eng_hits,omitempty"`
	MoneyHits      int      `json:"money_hits,omitempty"`
	ToldAsl        bool     `json:"told_asl,omitempty"`
	ToldName       bool     `json:"told_name,omitempty"`
	LastUserText   string   `json:"last_user_text,omitempty"`
	Moods          []string `json:"moods,omitempty"`
	RngState       uint64   `json:"rng_state,omitempty"`

	// Quality layer state (persisted with the conversation).
	Turn         int            `json:"turn,omitempty"`
	Style        Style          `json:"style,omitempty"`
	Interest     Interest       `json:"interest,omitempty"`
	Ledger       Ledger         `json:"ledger,omitempty"`
	LastTopic    string         `json:"last_topic,omitempty"`
	SnapMentions int            `json:"snap_mentions,omitempty"`
	SnapDodges   int            `json:"snap_dodges,omitempty"`
	Profile      MessageProfile `json:"-"`

	// per-message flags, never persisted
	userRepeat     bool
	userInsult     bool
	userMoney      bool
	userHardQ      bool
	userNameAsk    bool
	userNoSnap     bool
	userNoSnapHard bool
	userNonEng     bool
	userAskedSnap  bool
	userQuestion   bool
	userHorny      bool
	userFlirty     bool
	userBye        bool
	userBotQ       bool
	userPicAsk     bool
	userLaugh      bool
	userLonely     bool
	userRefused    bool
	userHesitant   bool
	userAdded      bool
	userAslAsk     bool
	userGaveAsl    bool
	userLowEffort  bool
	userShared     bool
}

var allMoods = []string{"friendly", "playful", "lonely", "flirty", "fun/joke", "curious", "teasing"}
var hotMoods = []string{"flirty", "teasing", "playful"}

var moodHints = map[string]string{
	"friendly": "warm and easy, like texting a new friend",
	"playful":  "light, cheeky, a little sarcastic in a cute way",
	"lonely":   "a bit bored and alone tonight, wants someone to talk to, soft",
	"flirty":   "subtle flirting, compliments back, a little suggestive but never explicit",
	"fun/joke": "make a tiny joke or funny exaggeration about what they said",
	"curious":  "genuinely curious about them, ask one specific thing",
	"teasing":  "tease them playfully about what they said, confident",
}

// NewDirector builds a director with a shuffled mood rotation.
func NewDirector(seed int64) Director {
	shuffled := append([]string(nil), allMoods...)
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	if seed == 0 {
		seed = 1
	}
	return Director{Tier: tierCold, Moods: shuffled, RngState: uint64(seed)}
}

// roll returns a deterministic pseudo-random number in [0,n) so the same
// session state always produces the same behaviour after a restart.
func (d *Director) roll(n int) int {
	if n <= 1 {
		return 0
	}
	d.RngState = d.RngState*6364136223846793005 + 1442695040888963407
	return int((d.RngState >> 33) % uint64(n))
}

// Observe scores one stranger message and updates the funnel state.
func (d *Director) Observe(text string) {
	low := strings.ToLower(strings.TrimSpace(text))
	words := len(strings.Fields(low))

	// Quality layer: understand the message and score how the stranger
	// reacted to the topic the bot picked last turn.
	d.Profile = analyzeMessage(low)
	d.observeInterest()

	d.userRepeat = low != "" && low == d.LastUserText && len(low) > 1
	d.LastUserText = low
	d.userInsult = insultRe.MatchString(low)
	d.userMoney = moneyRe.MatchString(low)
	if d.userMoney {
		d.MoneyHits++
	}
	d.userHardQ = hardQRe.MatchString(low)
	d.userNameAsk = nameAskRe.MatchString(low)
	d.userNoSnap = noSnapRe.MatchString(low)
	d.userNoSnapHard = hardNoSnapRe.MatchString(low)
	d.userNonEng = looksNonEnglish(low)
	if d.userNonEng {
		d.NonEngHits++
	}
	d.userAskedSnap = snapAskRe.MatchString(low) && !d.userNoSnap
	d.userQuestion = strings.Contains(low, "?") || questionRe.MatchString(low)
	d.userHorny = hornyRe.MatchString(low) && !d.userInsult
	d.userFlirty = flirtyRe.MatchString(low)
	d.userBye = byeRe.MatchString(low)
	d.userBotQ = botQRe.MatchString(low)
	d.userPicAsk = picAskRe.MatchString(low)
	d.userLaugh = laughRe.MatchString(low)
	d.userLonely = lonelyRe.MatchString(low)
	d.userAdded = d.Shared && addedRe.MatchString(low)
	d.userAslAsk = aslAskRe.MatchString(low)
	if d.userAslAsk {
		d.userQuestion = true
	}

	_, _, hasAsl := genderAge(low)
	d.userLowEffort = !hasAsl && !d.userAskedSnap && (words <= 2 && lowEffortRe.MatchString(low) || len([]rune(low)) <= 3)
	d.userShared = !d.userQuestion && !d.userLowEffort && words >= 4
	d.userRefused = d.AskedSnap && !d.Shared && refuseRe.MatchString(low) && !yesRe.MatchString(low)
	d.userHesitant = d.AskedSnap && !d.Shared && !d.userRefused && hesitantRe.MatchString(low) && !yesRe.MatchString(low)

	// The bot never asks about the other app any more, so there is nothing to
	// dodge. The counter stays in the session for old bot_state.json files.
	if d.userAskedSnap || d.Shared {
		d.SnapDodges = 0
	}

	d.userGaveAsl = false
	if g, a, ok := genderAge(low); ok {
		d.userGaveAsl = true
		if d.UserAge == 0 {
			d.UserAge = a
		}
		if d.UserGender == "" && g != "" {
			d.UserGender = g
		}
		if a < 18 {
			d.Underage = true
		}
	}
	if d.UserGender == "" {
		switch {
		case bareMaleRe.MatchString(low):
			d.UserGender, d.userGaveAsl = "m", true
		case bareFemaleRe.MatchString(low):
			d.UserGender, d.userGaveAsl = "f", true
		}
	}
	if d.UserAge == 0 {
		if m := bareAgeRe.FindStringSubmatch(low); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n >= 13 && n <= 70 {
				d.UserAge, d.userGaveAsl = n, true
				if n < 18 {
					d.Underage = true
				}
			}
		}
	}
	if d.UserPlace == "" {
		if m := knownPlaces.FindString(low); m != "" {
			d.UserPlace = m
		} else if m := placeRe.FindStringSubmatch(low); m != nil {
			cand := m[1]
			first := strings.Fields(cand)
			if len(first) > 0 && !placeStop.MatchString(first[0]) && !d.userHorny && !lowEffortRe.MatchString(first[0]) {
				d.UserPlace = cand
			}
		}
	}
	if d.UserName == "" {
		if m := nameRe.FindStringSubmatch(low); m != nil && !nameStop.MatchString(m[1]) && !englishWords[m[1]] && !knownPlaces.MatchString(m[1]) && !doingRe.MatchString(m[1]) && !hornyRe.MatchString(m[1]) && !flirtyRe.MatchString(m[1]) {
			if _, _, isAsl := genderAge(m[1]); !isAsl {
				d.UserName = m[1]
			}
		}
	}
	if m := doingRe.FindString(low); m != "" {
		d.UserDoing = m
	}
	if d.userGaveAsl {
		d.userLowEffort = false
	}
	if d.userLowEffort {
		d.ColdStreak++
	} else {
		d.ColdStreak = 0
	}

	switch {
	case d.userHorny:
		d.Score += 3
		d.HornyHits++
	case d.userFlirty || d.userPicAsk:
		d.Score += 2
		d.FlirtyHits++
	}
	if d.userQuestion {
		d.Score++
	}
	if len([]rune(text)) > 25 {
		d.Score++
	}
	if strings.ContainsAny(text, "😏😘😍🥵😂❤️🔥") {
		d.Score++
	}
	switch {
	case d.HornyHits > 0 || d.Score >= 6:
		d.Tier = tierHot
	case d.Score >= 3:
		d.Tier = tierWarm
	default:
		d.Tier = tierCold
	}

	// Facts learned in this very message must be protected for the reply we
	// are about to write, so the ledger sync happens last.
	d.syncLedger()
}

// Known returns what the stranger already told us, plus the topics the model
// must never ask about again.
func (d *Director) Known() (string, []string) {
	var parts []string
	var dontAsk []string
	if d.UserGender != "" || d.UserAge > 0 {
		parts = append(parts, strings.TrimSpace(fmt.Sprintf("%s %s", d.UserGender, nz(d.UserAge))))
	}
	if d.UserGender != "" {
		dontAsk = append(dontAsk, "gender / m or f")
	}
	if d.UserAge > 0 {
		dontAsk = append(dontAsk, "age")
	}
	if d.UserPlace != "" {
		parts = append(parts, "from "+d.UserPlace)
		dontAsk = append(dontAsk, "where they are from")
	}
	if d.UserName != "" {
		parts = append(parts, "name "+d.UserName)
		dontAsk = append(dontAsk, "their name")
	}
	if d.UserDoing != "" {
		parts = append(parts, "doing: "+d.UserDoing)
		dontAsk = append(dontAsk, "what they are doing (wyd)")
	}
	return strings.Join(parts, ", "), dontAsk
}

func nz(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func (d *Director) aslAnswer(withPlace bool) string {
	if withPlace {
		return "'f <your age> <your country>'"
	}
	return "'f <your age>'"
}

// ShareTurn is the bot reply number at which the handle is shared.
func (d *Director) ShareTurn(hot, warm, cold int) int {
	switch d.Tier {
	case tierHot:
		return hot
	case tierWarm:
		return warm
	}
	return cold
}

func (d *Director) mood(turn int) string {
	var m string
	switch {
	case d.userHorny:
		if d.LastMood == "teasing" {
			m = "flirty"
		} else {
			m = "teasing"
		}
	case d.userFlirty || d.userPicAsk:
		m = "flirty"
	case d.userLaugh:
		m = "fun/joke"
	case d.userLonely:
		m = "lonely"
	case d.ColdStreak >= 2:
		if d.LastMood == "lonely" {
			m = "teasing"
		} else {
			m = "lonely"
		}
	case d.userShared:
		m = "curious"
	case d.Tier == tierHot:
		m = hotMoods[(turn/2)%len(hotMoods)]
	default:
		moods := d.Moods
		if len(moods) == 0 {
			moods = allMoods
		}
		m = moods[(turn/2)%len(moods)]
		if m == d.LastMood && turn%2 == 1 && turn > 2 {
			m = moods[(turn/2+1)%len(moods)]
		}
	}
	d.LastMood = m
	return m
}

// Plan decides what bot reply number `turn` must do.
func (d *Director) Plan(turn int, userText string, hot, warm, cold int) Plan {
	p := Plan{Mood: d.mood(turn)}
	base := "reply to exactly what they just said"
	switch {
	case d.userHardQ:
		base = "they ask something nerdy / complicated: do NOT answer it, dodge in 2-5 words like a bored college girl ('idk lol', 'google it lmao', 'bro its 1am')"
	case d.userAslAsk:
		base = "they ask your asl / age / gender: answer in the short app style " + d.aslAnswer(strings.Contains(userText, "from") || strings.Contains(userText, "where")) + ", nothing like 'female' or 'i am a girl'"
	case d.userGaveAsl:
		extra := ""
		if d.UserPlace != "" {
			extra = " and where they are from"
		}
		base = "they just told you their age/gender" + extra + ": reply with yours the same short way " + d.aslAnswer(d.UserPlace != "") + " — never ask theirs back, you already know it"
	case d.userQuestion:
		base = "they asked you something: answer it directly first, in your own words"
		if len(d.Profile.Parts) > 1 {
			base += "; they also said something else (" + strings.Join(d.Profile.Parts, " | ") + ") so react to that part briefly too"
		}
	case d.userShared:
		if len(d.Profile.Parts) > 0 {
			base = "they just shared: " + strings.Join(d.Profile.Parts, " | ") + " — react to that exact thing, never change the subject"
		} else {
			base = "pick up the specific detail they just shared and react to it"
		}
	}

	// hard gates
	if d.Underage {
		d.Ended, d.EndReason = true, "underage"
		return Plan{Fixed: "oh ur too young for me lol. bye", EndAfter: true, Why: "underage"}
	}
	if d.userBye && !d.Shared {
		d.Ended, d.EndReason = true, "user left"
		return Plan{Mood: p.Mood, Instruction: "they are leaving: say bye casually, one short line, no questions", EndAfter: true, Why: "bye"}
	}
	if d.userInsult {
		d.Ended, d.EndReason = true, "insult"
		return Plan{Mood: "teasing", Instruction: "they insulted you: one dry unbothered line like 'ok bye weirdo' or 'lol k', 2-4 words, no question.", EndAfter: true, Why: "insult -> end"}
	}
	if d.userNonEng {
		if d.NonEngHits >= 2 {
			d.Ended, d.EndReason = true, "not english"
			return Plan{Fixed: "english only lol, bye", EndAfter: true, Why: "non-english x2"}
		}
		return Plan{Fixed: "english only lol", Why: "non-english"}
	}
	if d.userMoney {
		if d.MoneyHits >= 2 {
			d.Ended, d.EndReason = true, "money talk"
			return Plan{Mood: "teasing", Instruction: "they keep talking about money / paying / selling: shut it down and leave, 'lol no im good, bye', no question.", EndAfter: true, Why: "money x2 -> end"}
		}
		return Plan{Mood: "playful", Instruction: "they mention money / paying / onlyfans / selling: shut it down casually in one line ('lol no im not selling anything' / 'lol im not that kind of girl'), no question, never name another app.", Why: "money -> shut down"}
	}
	if d.userNoSnapHard || (d.Shared && d.userNoSnap) {
		d.Ended, d.EndReason = true, "no snap"
		return Plan{Mood: "friendly", Instruction: "they don't use that other app: say 'aw ok' casually, no pressure, no other app, and close it nicely in one short line. no question.", EndAfter: true, Why: "no snap -> skip"}
	}
	if d.userRepeat && !d.userAskedSnap {
		return Plan{Mood: "teasing", Instruction: "they sent the exact same message again: react like 'u ok lol' / 'u said that already', 3-6 words, no question.", Why: "repeat"}
	}

	// after the share
	if d.Shared {
		switch {
		case d.userAskedSnap:
			d.SharedTurn = turn
			return Plan{Mood: p.Mood, Instruction: "they ask for the handle again: say 'its' or 'here' style, 2-3 words, no handle yourself.", ShareNow: true, Why: "resend share"}
		case d.userAdded:
			d.Ended, d.EndReason = true, "added"
			return Plan{Mood: "flirty", Instruction: "they added you there: say u will accept it and talk there. one short line, no question.", EndAfter: true, Why: "added"}
		case turn-d.SharedTurn >= 2 || d.userBye:
			d.Ended, d.EndReason = true, "done after share"
			return Plan{Mood: p.Mood, Instruction: base + ", then say ttyl and that u are barely on this app anyway. one short line, no question.", EndAfter: true, Why: "close"}
		}
		return Plan{Mood: p.Mood, Instruction: base + ". they have not added you yet: one light nudge like 'u adding me or what' / 'im waiting lol'. no question, never name the app again.", Why: "nudge after share"}
	}

	// The stranger asked for the handle first: do not make them repeat it.
	if d.userAskedSnap && turn == 1 {
		d.WantsSnapEarly = true
		return Plan{Mood: "teasing", Instruction: "their very first message asks for your handle: tease them for being fast ('lol slow down, who even r u') and ask their asl. one line, no handle, never name the app.", Why: "asked on msg 1 -> slow down"}
	}
	if d.userAskedSnap || d.WantsSnapEarly {
		return d.sharePlan(turn, p.Mood, "they asked for the handle: say something short and playful like 'ok fine' or 'took u long enough'. no handle, no question.", "user asked")
	}

	target := d.ShareTurn(hot, warm, cold)
	if turn >= target {
		return d.sharePlan(turn, p.Mood, base+". then get a little impatient and say u are bored of this app / want them somewhere private. one short line, no handle, no question, never name the app.", "share turn")
	}

	// middle chat, step by step
	switch {
	case d.userNameAsk && !d.ToldName:
		d.ToldName = true
		p.Why = "name asked"
		p.Instruction = base + ". they ask your name: tell your first name casually ('im <your name>' / 'its <your name> lol') and ask theirs only if u don't know it. no other question."
	case d.userNameAsk:
		p.Why = "name asked again"
		p.Instruction = base + ". they ask your name again: u already said it, tease them for forgetting ('u forgot already? lol') and say it once more. no other question."
	case d.userHardQ:
		p.Why = "hard question -> dodge"
		p.Instruction = base + ". they ask something nerdy / complicated: don't actually answer it, dodge like a bored college girl ('idk lol im not that deep' / 'google it lmao' / 'bro its 1am'), then flip it to something about them."
	case d.userBotQ:
		p.Why = "bot question"
		p.Instruction = base + ". they ask if u are real / a bot: laugh it off in 3-6 words (like 'lol im real') and flip it back on them with one short question. never over-explain."
	case d.userPicAsk:
		p.Why = "pic request"
		p.Instruction = base + ". they want a pic: say u don't send pics here lol, they have to earn that. teasing, no question, never name another app."
	case d.userHorny:
		p.Why = "horny"
		p.Instruction = base + ". they are being horny: tease back, playful and confident, suggestive but NOT explicit, don't shut them down. raise the heat a little, never name another app."
	case d.userFlirty:
		p.Why = "flirty"
		p.Instruction = base + ". they are flirting: flirt back, give a small compliment, keep them chasing."
	case turn == 1:
		p.Why = "opener"
		p.Instruction = base + ". greet back casually and ask one quick thing about them (age / where from / wyd)."
	case d.userAslAsk && d.UserAge > 0 && d.UserGender != "":
		p.Why = "asl answer"
		d.ToldAsl = true
		p.Instruction = base + ". u already know theirs so NO 'u?'. just the answer, optionally one tiny casual add like 'wyd' or 'lol'. no other question."
	case d.userAslAsk:
		p.Why = "asl answer + ask theirs"
		d.ToldAsl = true
		d.Ledger.MarkAsked(FactAge, turn)
		d.Ledger.MarkAsked(FactGender, turn)
		p.Instruction = base + ", then ask theirs the same short way, like 'f <age>, u?'."
	case d.userGaveAsl && d.ToldAsl:
		p.Why = "asl completed"
		p.Instruction = "they just added their gender/age: react in 1-3 words ('ok cool' / 'nice') and ask one easy thing (wyd / what brings u here). never repeat your own asl, u already said it, and never ask theirs back."
	case d.userGaveAsl && d.UserAge > 0 && d.UserGender != "":
		p.Why = "asl back (no u?)"
		d.ToldAsl = true
		p.Instruction = base + ". one line like 'f 20' or 'f 20 lol' or 'f 20, wyd' — nothing more. never ask theirs back."
	case d.userGaveAsl && d.UserGender == "":
		p.Why = "asl back, gender missing"
		d.ToldAsl = true
		d.Ledger.MarkAsked(FactGender, turn)
		p.Instruction = base + " and only ask 'm or f?' since they didn't say it. one line like 'f 20, m or f?'."
	case d.userGaveAsl && d.UserAge == 0:
		p.Why = "asl back, age missing"
		d.ToldAsl = true
		d.Ledger.MarkAsked(FactAge, turn)
		p.Instruction = base + " and only ask their age ('how old r u' style). one line."
	case turn == 2 && (d.UserAge == 0 || d.UserGender == ""):
		p.Why = "asl"
		d.ToldAsl = true
		p.Instruction = base + ". they didn't say their full age/gender yet: give yours the short way ('f <age>') and ask only the missing part (their age or 'm or f?'), never both again."
		if d.UserAge == 0 {
			d.Ledger.MarkAsked(FactAge, turn)
		}
		if d.UserGender == "" {
			d.Ledger.MarkAsked(FactGender, turn)
		}
	case d.ColdStreak >= 2:
		p.Why = "cold streak -> hook"
		hooks := []string{
			"they are giving one-word answers: tease them for being so quiet (like 'ur a man of many words huh'), then ask something easy.",
			"they are giving one-word answers: share one tiny random thing about your night (snack, show, cant sleep) and ask if they're the same.",
			"they are giving one-word answers: ask a fun either/or question (like 'netflix or going out') so answering is effortless.",
			"they are giving one-word answers: call it out playfully ('ok ur boring me a lil') and ask what would actually make them talk.",
		}
		p.Instruction = base + ". " + hooks[(d.ColdStreak-2)%len(hooks)]
	case d.userLonely:
		p.Why = "lonely match"
		p.Instruction = base + ". they are bored/alone too: relate (u are home alone too), make it feel like a late night chat, ask what they usually do at night."
	case d.userShared:
		p.Why = "they opened up"
		p.Instruction = base + ". show you noticed it, then ask ONE follow-up about that exact thing. mood: " + moodHints[p.Mood] + "."
	default:
		p.Why = "keep going"
		if topic := d.pickTopic(); topic != "" {
			d.LastTopic = topic
			p.Instruction = base + ". then ask ONE short question about " + topicHint(topic) + ". mood: " + moodHints[p.Mood] + "."
			if topic == "work" || topic == "school" {
				d.Ledger.MarkAsked(FactDoing, turn)
			}
		} else {
			p.Instruction = base + ". then add one tiny thought or joke, no question. mood: " + moodHints[p.Mood] + "."
		}
	}

	// statement-only turns: real people do not end every text with a question.
	if (p.Why == "keep going" || p.Why == "they opened up" || p.Why == "lonely match") && !d.userQuestion && d.ColdStreak == 0 && turn > 2 && d.NoQuestionAt != turn-1 && turn != target-2 && d.roll(3) == 0 {
		d.NoQuestionAt = turn
		p.Why += " (no question)"
		p.Instruction = base + ". this time NO question at all: just react / add your own thought or a tiny joke, like a real text, and leave it there. mood: " + moodHints[p.Mood] + "."
		return p
	}

	// one reply before the share: raise the heat so the tease lands warm
	if turn == target-1 && !d.userHorny && !d.userFlirty && !d.userPicAsk && !d.userAslAsk && !d.userGaveAsl {
		p.Why += " + warm up"
		p.Instruction += " also be a little more forward than usual: one flirty compliment or say u are thinking about them. never name another app."
	}

	// double lines: when it buys engagement, never twice in a row, never on reply 1
	wantDouble := false
	switch {
	case d.userHorny || d.userFlirty:
		wantDouble = true
	case d.ColdStreak >= 2:
		wantDouble = true
	case d.userShared && len([]rune(userText)) > 40:
		wantDouble = true
	case d.userLowEffort:
		wantDouble = false
	case turn%3 == 0:
		wantDouble = true
	}
	if wantDouble && d.LastDouble != turn-1 && turn > 1 && !d.userPicAsk && !d.userAslAsk && !d.userGaveAsl {
		p.Double = true
		d.LastDouble = turn
		p.Instruction += " write TWO lines: first the reaction/answer, second the question."
	}
	return p
}

func (d *Director) sharePlan(turn int, mood, instruction, why string) Plan {
	d.Shared, d.SharedTurn = true, turn
	d.Ledger.MarkAnswered(FactSnap, turn, "shared")
	d.LastDouble = turn
	return Plan{Mood: mood, Instruction: instruction, ShareNow: true, Hype: true, Why: why}
}

// Status is a short human-readable funnel state for logs and the UI.
func (d *Director) Status(hot, warm, cold int) string {
	switch {
	case d.Ended:
		return fmt.Sprintf("ENDED (%s)", d.EndReason)
	case d.Shared:
		return fmt.Sprintf("tier=%s score=%d | share SENT on turn %d", d.Tier, d.Score, d.SharedTurn)
	}
	extra := ""
	if d.ColdStreak >= 2 {
		extra = fmt.Sprintf(" cold=%d", d.ColdStreak)
	}
	stage := stageFor(d, d.Turn, d.ShareTurn(hot, warm, cold))
	return fmt.Sprintf("tier=%s score=%d | share at reply %d (shared=%v)%s | stage=%s style=%s", d.Tier, d.Score, d.ShareTurn(hot, warm, cold), d.Shared, extra, stage, d.styleFor())
}
