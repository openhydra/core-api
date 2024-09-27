package test

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

func CreateMultiPartForm(files []struct{ FileName, FilePath string }, additionalProperty map[string]string) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, file := range files {
		part, err := writer.CreateFormFile(file.FileName, filepath.Base(file.FilePath))
		if err != nil {
			return nil, "", err
		}

		// Open the file
		f, err := os.Open(file.FilePath)
		if err != nil {
			return nil, "", err
		}
		defer f.Close()

		// Copy the file content to the form field
		_, err = io.Copy(part, f)
		if err != nil {
			return nil, "", err
		}
		// avoid property name conflict
		delete(additionalProperty, file.FileName)
	}

	for k, v := range additionalProperty {
		writer.WriteField(k, v)
	}

	err := writer.Close()
	if err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}
