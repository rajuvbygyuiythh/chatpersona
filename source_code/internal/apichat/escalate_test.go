package apichat

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestMentionsSnapCatchesEverySpelling(t *testing.T) {
	for _, line := range []string{
		"add me on snap",
		"my snapchat is private",
		"sc: someone",
		"snp later",
		"u got snap?",
		"add me rn",
		"add my other app",
		"whats ur sc",
		"just add u later",
	} {
		if !mentionsSnap(line) {
			t.Errorf("mentionsSnap(%q) = false, want true", line)
		}
	}
	for _, line := range []string{
		"ur cute",
		"im addicted to u",
		"scratch that",
		"ur a snack lol",
		"discover me later",
		"u talk big for someone who'd beg",
	} {
		if mentionsSnap(line) {
			t.Errorf("mentionsSnap(%q) = true, want false", line)
		}
	}
}

func TestSanitizeDropsEveryAppMention(t *testing.T) {
	lines, warn := sanitizeLines("u should add me on snap lol", false)
	if len(lines) != 0 {
		t.Fatalf("a snap line must never reach the chat: %v", lines)
	}
	if !strings.Contains(warn, "other app") {
		t.Fatalf("warn should explain the drop, got %q", warn)
	}
}

func TestGuardRejectsAnAppMention(t *testing.T) {
	d := &Director{UserAge: 22, UserGender: "m"}
	if got := validateReply([]string{"snap me later"}, d, analyzeMessage("hey"), "hey", nil); !strings.Contains(got, "other app") {
		t.Fatalf("guard reason = %q, want the app mention", got)
	}
}

func TestEveryHypeLineIsClean(t *testing.T) {
	for _, pool := range [][]string{hypeHotLines, hypePlayfulLines, hypeSoftLines} {
		for _, line := range pool {
			if mentionsSnap(line) {
				t.Errorf("hype line names the app: %q", line)
			}
			if strings.Contains(line, "?") {
				t.Errorf("hype line asks a question: %q", line)
			}
			if line != strings.ToLower(line) {
				t.Errorf("hype line must be lowercase: %q", line)
			}
		}
	}
}

func TestHypePoolFollowsTheVibe(t *testing.T) {
	d := &Director{Tier: tierHot, Score: 8}
	if got := hypePoolFor(d, Plan{Mood: "flirty"}); len(got) != len(hypeHotLines) {
		t.Fatalf("hot stranger should get the hot deck, got %d lines", len(got))
	}
	d.Profile.Kind = KindJoke
	if got := hypePoolFor(d, Plan{Mood: "fun/joke"}); len(got) != len(hypePlayfulLines) {
		t.Fatalf("joke should get the playful deck, got %d lines", len(got))
	}
	cold := &Director{Tier: tierCold, Score: 0}
	if got := hypePoolFor(cold, Plan{Mood: "friendly"}); len(got) != len(hypeSoftLines) {
		t.Fatalf("cold stranger should get the soft deck, got %d lines", len(got))
	}
}

func TestHypeLineNeverRepeats(t *testing.T) {
	s := NewSession(nil)
	e := &Engine{}
	seen := map[string]bool{}
	for i := 0; i < 5; i++ {
		line := e.pickHypeLine(&s, Plan{Mood: "flirty"}, nil)
		if line == "" {
			t.Fatalf("pick %d returned an empty tease", i)
		}
		if seen[line] {
			t.Fatalf("tease repeated: %q", line)
		}
		seen[line] = true
	}
}

func TestShareSendsOneTeaseBeforeTheHandle(t *testing.T) {
	var calls int
	engine := newTestEngine(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		fmt.Fprint(w, replyJSON("ok fine lol"))
	}))

	session := NewSession(nil)
	session.Director.Tier = tierHot
	ctx := context.Background()
	_ = engine.Start(ctx, &session)
	result := engine.Reply(ctx, &session, "u got snap?")
	if calls != 1 {
		t.Fatalf("the share turn must be hand-written, no model call; calls = %d", calls)
	}
	if !result.Shared || !result.HasUsername {
		t.Fatalf("share missing: %+v", result)
	}
	if len(result.Lines) != 2 {
		t.Fatalf("want exactly [tease, share], got %v", result.Lines)
	}
	if mentionsSnap(result.Lines[0]) {
		t.Fatalf("the tease mentions the app: %q", result.Lines[0])
	}
	if !strings.Contains(result.Lines[1], UsernamePlaceholder) {
		t.Fatalf("last line must be the share with the placeholder: %v", result.Lines)
	}
	if !strings.Contains(result.Detail, "share") {
		t.Fatalf("detail should describe the share state: %q", result.Detail)
	}
}

func TestModelSnapLineIsRetriedInsteadOfSent(t *testing.T) {
	var calls int
	engine := newTestEngine(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls++; calls == 1 {
			fmt.Fprint(w, replyJSON("add me on snap lol"))
			return
		}
		fmt.Fprint(w, replyJSON("lol nice one"))
	}))

	session := NewSession(nil)
	result := engine.Reply(context.Background(), &session, "hey wyd")
	if len(result.Lines) == 0 {
		t.Fatal("empty reply")
	}
	for _, line := range result.Lines {
		if mentionsSnap(line) {
			t.Fatalf("engine sent an app mention: %v", result.Lines)
		}
	}
	if calls < 2 {
		t.Fatalf("a snap line must trigger a retry, calls = %d", calls)
	}
}
