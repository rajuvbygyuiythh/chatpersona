package apichat

// Goal ladder: every reply has a stage and a single goal. The stage keeps the
// conversation moving toward the snap instead of drifting in small talk.

// Stage is where the funnel currently is.
type Stage string

const (
	StageRapport  Stage = "rapport"
	StageInterest Stage = "interest"
	StageHook     Stage = "hook"
	StageShare    Stage = "share"
	StageClose    Stage = "close"
)

// stageFor derives the stage from the director state and the configured turn.
func stageFor(d *Director, turn, target int) Stage {
	switch {
	case d.Shared:
		return StageClose
	case turn >= target:
		return StageShare
	case turn <= 2:
		return StageRapport
	case turn >= target-2:
		return StageHook
	}
	return StageInterest
}

// stageGoal is the one-line objective handed to the model for this reply.
func stageGoal(stage Stage, p MessageProfile) string {
	switch stage {
	case StageRapport:
		return "make them comfortable, react to their message, learn one small thing about them"
	case StageInterest:
		if topic := firstChatTopic(p.Topics); topic != "" {
			return "show you get them, then dig into " + topicHint(topic)
		}
		return "show you get them, keep them talking about themselves"
	case StageHook:
		return "keep the vibe and get a little more flirty, never name another app"
	case StageShare:
		return "raise the heat, the app sends your handle itself right after"
	case StageClose:
		return "close warmly: ttyl, or say you will accept their add"
	}
	return "react to what they said"
}

func firstChatTopic(topics []string) string {
	for _, topic := range topics {
		if chattableTopic(topic) {
			return topic
		}
	}
	return ""
}
