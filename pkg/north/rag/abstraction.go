package rag

import (
	"core-api/cmd/core-api-server/app/config"
	"io"
)

type ITempFileChatProvider interface {
	SaveFiles(files []TempChatChatFile, workspacePath string) error
	GetFiles(conversationId string, user1Id string, workspacePath string) (io.Reader, string, error)
	DeleteFile(conversationId string, user1Id string, workspacePath string) error
}

func NewTempFileChatProvider(config *config.Config) ITempFileChatProvider {
	return &DefaultTempFileChatProvider{
		config: config,
	}
}
