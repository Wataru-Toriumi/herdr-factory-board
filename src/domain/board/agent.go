package board

const (
	Codex  = "codex"
	Claude = "claude"
)

// 担当を保存していない旧版のカードと予定は Codex で実行する。
func AgentLabel(agent string) string {
	switch agent {
	case "", Codex:
		return "Codex"
	case Claude:
		return "Claude Code"
	default:
		return agent
	}
}

// NextAgent は担当切替で次に選ぶエージェントを返す。
func NextAgent(current string) string {
	if current == Claude {
		return Codex
	}
	return Claude
}
