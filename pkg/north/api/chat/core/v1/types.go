package v1

type ChatPost struct {
	Query          string    `json:"query,omitempty"`
	ConversationId string    `json:"conversation_id,omitempty"`
	HistoryLength  int       `json:"history_len,omitempty"`
	Messages       []Message `json:"messages,omitempty"`
	Stream         bool      `json:"stream,omitempty"`
	Model          string    `json:"model,omitempty"`
	Temperature    float32   `json:"temperature,omitempty"`
}

type KbChatPost struct {
	Query          string    `json:"query,omitempty"`
	ConversationId string    `json:"conversation_id,omitempty"`
	HistoryLength  int       `json:"history_len,omitempty"`
	Messages       []Message `json:"messages,omitempty"`
	Stream         bool      `json:"stream,omitempty"`
	Model          string    `json:"model,omitempty"`
	Temperature    float32   `json:"temperature,omitempty"`
	KBName         string    `json:"kb_name,omitempty"`
	KBId           string    `json:"kb_id,omitempty"`
	TopK           int       `json:"top_k,omitempty"`
	ScoreThreshold float32   `json:"score_threshold,omitempty"`
}

type FileChatPost struct {
	Query          string    `json:"query,omitempty"`
	ConversationId string    `json:"conversation_id,omitempty"`
	TempKBId       string    `json:"knowledge_id,omitempty"`
	Messages       []Message `json:"messages,omitempty"`
	Stream         bool      `json:"stream,omitempty"`
	HistoryLength  int       `json:"history_len,omitempty"`
	Model          string    `json:"model,omitempty"`
	Temperature    float32   `json:"temperature,omitempty"`
}

type Message struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type ChatQuickStart struct {
	Name  string   `json:"name,omitempty" yaml:"name,omitempty"`
	Title string   `json:"title,omitempty" yaml:"title,omitempty"`
	Query string   `json:"query,omitempty" yaml:"query,omitempty"`
	Files []string `json:"files,omitempty" yaml:"files,omitempty"`
}

type ChatQuickStarts struct {
	ChatQuickStarts map[string][]ChatQuickStart `json:"chat_quick_starts,omitempty" yaml:"chatQuickStarts,omitempty"`
}
