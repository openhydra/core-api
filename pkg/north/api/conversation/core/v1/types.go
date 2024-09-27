package v1

// Conversation represents a chat conversation
type Conversation struct {
	ID           string `json:"id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Name         string `json:"name,omitempty"`
	ChatType     string `json:"chat_type,omitempty"`
	TempKBId     string `json:"temp_kb_id,omitempty"`
	TempFileName string `json:"temp_file_name,omitempty"`
	CreateTime   string `json:"create_time,omitempty"`
}

type ConversationMessage struct {
	ConversationID string `json:"conversation_id,omitempty"`
	Id             string `json:"id,omitempty"`
	ChatType       string `json:"chat_type,omitempty"`
	Query          string `json:"query,omitempty"`
	Response       string `json:"response,omitempty"`
	TempKBId       string `json:"temp_kb_id,omitempty"`
	TempFileName   string `json:"temp_file_name,omitempty"`
	CreateTime     string `json:"create_time,omitempty"`
}
