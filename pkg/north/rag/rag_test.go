package rag

import (
	"bytes"
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/util/common"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var workspacePath = "/tmp"

var testRouter = func(ws *restful.WebService) {
	fileChatPath := "/apis/rag.openhydra.io/v1/fileChat"
	ws.Route(ws.POST(fileChatPath).To(func(request *restful.Request, response *restful.Response) {
		// parse request body to TestBodyStruct
		// read from request.Request.Body
		file, fileHeader, err := request.Request.FormFile("file")
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}

		// change file path to avoid conflict with seed test file
		fileHeader.Filename = "/tmp/core-api-rag-test-file2"

		ragChatFileProvider := NewTempFileChatProvider(&config.Config{})
		err = ragChatFileProvider.SaveFiles([]TempChatChatFile{
			{
				ConversationId: "testConversationId",
				User1Id:        "testUser1Id",
				File:           &file,
				FileHeader:     fileHeader,
			}}, workspacePath)

		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
		response.WriteHeader(http.StatusCreated)
	}))
}

var _ = Describe("rag test", func() {
	var createMultiPartBody = func(txtData map[string]string, filePath string) (io.Reader, string, error) {
		var (
			buf = new(bytes.Buffer)
			w   = multipart.NewWriter(buf)
		)

		for k, v := range txtData {
			_ = w.WriteField(k, v)
		}

		part, err := w.CreateFormFile("file", "/tmp/core-api-rag-test-file")
		if err != nil {
			return nil, "", nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", err
		}

		_, err = part.Write(data)
		if err != nil {
			return nil, "", err
		}

		w.Close()
		return buf, w.FormDataContentType(), nil
	}
	Describe("Default provider test", func() {
		BeforeEach(func() {
			// write file content hello world to /tmp/core-api-rag-test-file
			err := os.WriteFile("/tmp/core-api-rag-test-file", []byte("hello world"), 0644)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			// remove /tmp/core-api-rag-test-file
			_ = os.Remove("/tmp/core-api-rag-test-file")
		})
		Context("Save file test", func() {
			It("should save file as expected", func() {
				stopChan := make(chan struct{})
				go common.StartMockServer(8080, testRouter, stopChan)
				defer close(stopChan)
				time.Sleep(1 * time.Second)
				filePath := "/tmp/core-api-rag-test-file"
				reader, contentType, err := createMultiPartBody(nil, filePath)
				Expect(err).To(BeNil())
				header := make(map[string][]string)
				header["Content-Type"] = []string{contentType}
				body, err := io.ReadAll(reader)
				Expect(err).To(BeNil())
				_, _, retCode, err := common.CommonRequest("http://localhost:8080/apis/rag.openhydra.io/v1/fileChat", http.MethodPost, "", body, header, false, false, 0)
				Expect(err).To(BeNil())
				Expect(retCode).To(Equal(http.StatusCreated))
			})
		})
		Context("Get file test", func() {
			It("should get file as expected", func() {
				ragChatFileProvider := NewTempFileChatProvider(&config.Config{})
				reader, contentType, err := ragChatFileProvider.GetFiles("testConversationId", "testUser1Id", workspacePath)
				Expect(err).To(BeNil())
				Expect(strings.ContainsAny(contentType, "multipart/form-data;")).To(BeTrue())
				Expect(len(reader.(*bytes.Buffer).Bytes())).To(Equal(263))
			})
		})
		Context("Delete file test", func() {
			It("should delete file as expected", func() {
				ragChatFileProvider := NewTempFileChatProvider(&config.Config{})
				err := ragChatFileProvider.DeleteFile("testConversationId", "testUser1Id", workspacePath)
				Expect(err).To(BeNil())
				// check if file is deleted
				_, err = os.Stat("/tmp/ragFileChat/testConversationId/testUser1Id/core-api-rag-test-file2")
				Expect(os.IsNotExist(err)).To(BeTrue())
			})
		})
	})
})
