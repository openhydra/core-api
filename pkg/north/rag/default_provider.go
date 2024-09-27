package rag

import (
	"bytes"
	"core-api/cmd/core-api-server/app/config"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type TempChatChatFile struct {
	ConversationId string
	User1Id        string
	File           *multipart.File
	FileHeader     *multipart.FileHeader
}

type DefaultTempFileChatProvider struct {
	config *config.Config
}

func (p *DefaultTempFileChatProvider) SaveFiles(files []TempChatChatFile, workspacePath string) error {
	for _, file := range files {
		// save file
		destinationPath := filepath.Join(workspacePath, "file-chat", file.ConversationId, file.User1Id)
		// create dir
		err := os.MkdirAll(destinationPath, os.ModePerm)
		if err != nil {
			return err
		}
		dst, err := os.Create(filepath.Join(destinationPath, filepath.Base(file.FileHeader.Filename)))
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, *file.File)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *DefaultTempFileChatProvider) GetFiles(conversationId string, user1Id string, workspacePath string) (io.Reader, string, error) {
	destinationPath := filepath.Join(workspacePath, "file-chat", conversationId, user1Id)

	// list all files in the directory
	// for each file, add to files
	entries, err := os.ReadDir(destinationPath)
	if err != nil {
		return nil, "", err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, entry := range entries {
		filePath := filepath.Join(destinationPath, entry.Name())
		part, err := writer.CreateFormFile(entry.Name(), filePath)
		if err != nil {
			return nil, "", err
		}

		// Open the file
		f, err := os.Open(filePath)
		if err != nil {
			return nil, "", err
		}

		defer f.Close()

		// Copy the file content to the form field
		_, err = io.Copy(part, f)
		if err != nil {
			return nil, "", err
		}

	}

	return body, writer.FormDataContentType(), nil
}

func (p *DefaultTempFileChatProvider) DeleteFile(conversationId string, user1Id string, workspacePath string) error {
	destinationPath := filepath.Join(workspacePath, "file-chat", conversationId, user1Id)
	err := os.RemoveAll(destinationPath)
	if err != nil {
		return err
	}
	return nil
}
