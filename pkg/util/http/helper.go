package http

import (
	coreApiLog "core-api/pkg/logger"
	"encoding/json"
	"net/http"
)

type FileUploadFailedResponse struct {
	Id          string `json:"id"`
	FailedFiles []struct {
		FileName string `json:"filename"`
		Msg      string `json:"msg"`
	} `json:"failed_files"`
	SucceededFiles []struct {
		FileName string `json:"filename"`
	} `json:"succeeded_files"`
}

type CustomError struct {
	CustomErrCode           string                    `json:"customErrCode"`
	Message                 string                    `json:"message"`
	FileUploadFailedMessage *FileUploadFailedResponse `json:"data"`
}

func WriteHttpErrorAndLog(w http.ResponseWriter, message string, code int, err error) {
	coreApiLog.Logger.Error(message, "error", err)
	http.Error(w, message, code)
}

func WriteCustomErrorAndLog(w http.ResponseWriter, message string, code int, customErrCode string, err error) {
	coreApiLog.Logger.Error(message, "error", err)
	customErr := &CustomError{
		CustomErrCode: customErrCode,
		Message:       message,
	}

	response, err := json.Marshal(customErr)
	if err != nil {
		WriteHttpErrorAndLog(w, "Failed to marshal response", http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(code)
	w.Write(response)
}

func WriteCustomErrorAndLogHandler(w http.ResponseWriter, message string, code int, customErrCode string, err error, handler func(customError *CustomError)) {

	coreApiLog.Logger.Error(message, "error", err)
	customErr := &CustomError{
		CustomErrCode: customErrCode,
		Message:       message,
	}
	if handler != nil {
		handler(customErr)
	}

	response, err := json.Marshal(customErr)
	if err != nil {
		WriteHttpErrorAndLog(w, "Failed to marshal response", http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(code)
	w.Write(response)
}

func WriteResponseEntity(w http.ResponseWriter, entity interface{}) {
	response, err := json.Marshal(entity)
	if err != nil {
		WriteHttpErrorAndLog(w, "Failed to write entity due to json marshal failed", http.StatusInternalServerError, err)
		return
	}
	w.Write(response)
}

func ParseJsonBody[T any](r *http.Request, t *T) error {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(t)
	if err != nil {
		return err
	}
	return nil
}

func GetCommonHttpHeader(additionalHeader map[string][]string) map[string][]string {
	headers := map[string][]string{
		"Content-Type": {"application/json"},
	}
	for k, v := range additionalHeader {
		headers[k] = v
	}
	return headers
}
