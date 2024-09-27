package south

import (
	"core-api/cmd/core-api-server/app/config"
	commonHelper "core-api/pkg/util/common"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	xInferenceV1 "core-api/pkg/north/api/xinference/core/v1"
	httpHelper "core-api/pkg/util/http"

	"github.com/go-chi/chi/v5"
)

var ModelTemplate map[string]map[string]xInferenceV1.XInferenceModelLauncher = map[string]map[string]xInferenceV1.XInferenceModelLauncher{
	"qwen1.5-chat": {
		"small": {
			ModelUid:         "qwen1.5-chat",
			ModelEngine:      "vLLM",
			NumberOfGpuToUse: "auto",
			DownloadHub:      "modelscope",
			XInferenceCommon: &xInferenceV1.XInferenceCommon{
				GpuMemoryUtilization: 0.5,
				ModelFormat:          "gptq",
				ModelType:            "LLM",
				Quantization:         "Int4",
				Replica:              1,
				ModelName:            "qwen1.5-chat",
				ModelSizeInBillions:  4,
			},
		},
		"medium": {
			ModelUid:         "qwen1.5-chat",
			ModelEngine:      "vLLM",
			NumberOfGpuToUse: "auto",
			DownloadHub:      "modelscope",
			XInferenceCommon: &xInferenceV1.XInferenceCommon{
				GpuMemoryUtilization: 0.7,
				ModelFormat:          "gptq",
				ModelType:            "LLM",
				Quantization:         "Int4",
				Replica:              1,
				ModelName:            "qwen1.5-chat",
				ModelSizeInBillions:  4,
			},
		},
		"large": {
			ModelUid:         "qwen1.5-chat",
			ModelEngine:      "vLLM",
			NumberOfGpuToUse: "auto",
			DownloadHub:      "modelscope",
			XInferenceCommon: &xInferenceV1.XInferenceCommon{
				GpuMemoryUtilization: 0.9,
				ModelFormat:          "gptq",
				ModelType:            "LLM",
				Quantization:         "Int4",
				Replica:              1,
				ModelName:            "qwen1.5-chat",
				ModelSizeInBillions:  4,
			},
		},
	},
	"bge-base-zh-v1.5": {
		"default": {
			ModelUid:         "bge-base-zh-v1.5",
			ModelEngine:      "embedding",
			NumberOfGpuToUse: "auto",
			DownloadHub:      "modelscope",
			XInferenceCommon: &xInferenceV1.XInferenceCommon{
				ModelType: "embedding",
				ModelName: "bge-base-zh-v1.5",
				Replica:   1,
			},
		},
	},
	"bge-large-zh-v1.5": {
		"default": {
			ModelUid:         "bge-large-zh-v1.5",
			ModelEngine:      "embedding",
			NumberOfGpuToUse: "auto",
			DownloadHub:      "modelscope",
			XInferenceCommon: &xInferenceV1.XInferenceCommon{
				ModelType: "embedding",
				ModelName: "bge-large-zh-v1.5",
				Replica:   1,
			},
		},
	},
}

type XInferenceSouthAPIHandler struct {
	config *config.Config
}

func NewXInferenceSouthAPIHandler(config *config.Config) *XInferenceSouthAPIHandler {
	return &XInferenceSouthAPIHandler{
		config: config,
	}
}

func (handler *XInferenceSouthAPIHandler) ListAllModels(w http.ResponseWriter, r *http.Request) {
	result, _, retCode, err := commonHelper.CommonRequest(fmt.Sprintf("%s/v1/models", handler.config.XInference.Endpoint), http.MethodGet, "", nil, r.Header, true, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to list all models", http.StatusInternalServerError, "", err)
		return
	}

	if retCode != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to list all models", retCode, "", fmt.Errorf("unexpected http response code"))
		return
	}

	models := &xInferenceV1.XInferenceModelList{}
	err = json.Unmarshal(result, models)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to list all models", http.StatusInternalServerError, "", err)
		return
	}

	httpHelper.WriteResponseEntity(w, models)
}

func (handler *XInferenceSouthAPIHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	result, _, retCode, err := commonHelper.CommonRequest(fmt.Sprintf("%s/v1/models/%s", handler.config.XInference.Endpoint, chi.URLParam(r, "modelId")), r.Method, "", nil, r.Header, true, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get model", http.StatusInternalServerError, "", err)
		return
	}

	if retCode != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get model", retCode, "", fmt.Errorf("unexpected http response code"))
		return
	}

	model := &xInferenceV1.XInferenceModel{}
	err = json.Unmarshal(result, model)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get model", http.StatusInternalServerError, "", err)
		return
	}

	httpHelper.WriteResponseEntity(w, model)
}

func (handler *XInferenceSouthAPIHandler) CreateModel(w http.ResponseWriter, r *http.Request) {
	// parse request body to []byte
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "", err)
		return
	}

	fontReq := &xInferenceV1.XInferenceModelFontLauncher{}
	err = json.Unmarshal(body, fontReq)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusInternalServerError, "", err)
		return
	}

	_, supported := ModelTemplate[fontReq.ModelName]
	if !supported {
		httpHelper.WriteCustomErrorAndLog(w, "Model not supported", http.StatusBadRequest, "", fmt.Errorf("model not supported"))
		return
	}

	var launcher xInferenceV1.XInferenceModelLauncher

	if fontReq.ModelName == "bge-base-zh-v1.5" || fontReq.ModelName == "bge-large-zh-v1.5" {
		launcher = ModelTemplate[fontReq.ModelName]["default"]
	} else {
		launcherFound, templateFound := ModelTemplate[fontReq.ModelName][fontReq.Template]
		if !templateFound {
			httpHelper.WriteCustomErrorAndLog(w, "Model template not found", http.StatusBadRequest, "", fmt.Errorf("model template not found"))
			return
		}
		launcher = launcherFound
	}

	postBody, err := json.Marshal(&launcher)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal model launcher", http.StatusInternalServerError, "", err)
		return
	}

	result, _, retCode, err := commonHelper.CommonRequest(fmt.Sprintf("%s/v1/models?wait_ready=false", handler.config.XInference.Endpoint), r.Method, "", postBody, r.Header, true, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create model", http.StatusInternalServerError, "", err)
		return
	}

	if retCode != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create model", retCode, "", fmt.Errorf("unexpected http response code %s", string(result)))
		return
	}

	model := &xInferenceV1.XInferenceModelLauncher{}
	err = json.Unmarshal(result, model)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create model", http.StatusInternalServerError, "", err)
		return
	}

	httpHelper.WriteResponseEntity(w, model)
}

// delete model
func (handler *XInferenceSouthAPIHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	result, _, retCode, err := commonHelper.CommonRequest(fmt.Sprintf("%s/v1/models/%s", handler.config.XInference.Endpoint, chi.URLParam(r, "modelId")), r.Method, "", nil, r.Header, true, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete model", http.StatusInternalServerError, "", err)
		return
	}

	if retCode != http.StatusNoContent {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete model", retCode, "", fmt.Errorf("unexpected http response code with error %s", string(result)))
		return
	}

	model := &xInferenceV1.XInferenceModelLauncher{
		ModelUid: chi.URLParam(r, "modelId"),
	}

	httpHelper.WriteResponseEntity(w, model)
}
