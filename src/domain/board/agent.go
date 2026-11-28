package board

const (
	Codex  = "codex"
	Claude = "claude"
)

func Agents() []string { return []string{Codex, Claude} }

// EffectiveDefaultAgent は既定の担当が未設定または未対応なら Codex を返す。
func (b State) EffectiveDefaultAgent() string {
	if b.DefaultAgent == Claude {
		return Claude
	}
	return Codex
}

// 新規登録時だけボードの既定値を使い、フォームで選んだ担当を優先する。
func (b State) AgentForNewItem(selected string) string {
	if selected != "" {
		return selected
	}
	return b.EffectiveDefaultAgent()
}

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
