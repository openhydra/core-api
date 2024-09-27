package south

import (
	"core-api/cmd/core-api-server/app/config"
	coreApiLog "core-api/pkg/logger"
	commonHelper "core-api/pkg/util/common"
	httpHelper "core-api/pkg/util/http"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type RayLLMInferenceSouthAPIHandler struct {
	config *config.Config
}

type RayLLMInferencePostBody struct {
	ModelName         string         `json:"model_name,omitempty"`
	Status            string         `json:"status,omitempty"`
	Message           string         `json:"message,omitempty"`
	LastDeployedTimeS float64        `json:"last_deployed_time_s,omitempty"`
	Deployments       *RayDeployment `json:"deployments,omitempty"`
}

type RayDeployment struct {
	RayVLLMDeployment *RayVLLMDeployment `json:"VllmDeployment,omitempty"`
}

type RayVLLMDeployment struct {
	Status        string            `json:"status,omitempty"`
	StatusTrigger string            `json:"status_trigger,omitempty"`
	ReplicaStates *RayReplicaStates `json:"replica_states,omitempty"`
	Message       string            `json:"message,omitempty"`
}

type RayReplicaStates struct {
	Starting int `json:"STARTING,omitempty"`
	Running  int `json:"RUNNING,omitempty"`
}

func NewRayLLMInferenceSouthAPIHandler(config *config.Config) *RayLLMInferenceSouthAPIHandler {
	return &RayLLMInferenceSouthAPIHandler{
		config: config,
	}
}

func (h *RayLLMInferenceSouthAPIHandler) GetModels(w http.ResponseWriter, r *http.Request) {
	llmType := r.URL.Query().Get("llmType")
	if llmType != "" {
		if llmType != "llm_models" && llmType != "embedding_models" && llmType != "all" {
			httpHelper.WriteCustomErrorAndLog(w, "llmType must be either llm or embedding or all", http.StatusBadRequest, "InvalidLLMType", fmt.Errorf("llmType must be either llm or vllm"))
			return
		}
	}

	// if llmType is empty, set it to all
	if llmType == "" {
		llmType = "all"
	}

	var queryType []string
	if llmType == "all" {
		queryType = []string{"llm_models", "embedding_models"}
	} else {
		queryType = []string{llmType}
	}
	var resultModels []string
	for _, t := range queryType {
		result, _, _, err := commonHelper.CommonRequest(fmt.Sprintf("%s/deployment/%s", h.config.RayLLM.Endpoint, t), http.MethodGet, "", nil, r.Header, true, true, 3*time.Second)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get models", http.StatusInternalServerError, "FailedToGetModels", err)
			return
		}
		currentQueryResult := []string{}
		err = json.Unmarshal(result, &currentQueryResult)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal models", http.StatusInternalServerError, "FailedToUnmarshalModels", err)
			return
		}
		resultModels = append(resultModels, currentQueryResult...)
	}

	httpHelper.WriteResponseEntity(w, resultModels)
}

func (h *RayLLMInferenceSouthAPIHandler) GetModel(w http.ResponseWriter, r *http.Request) {

	modelId := chi.URLParam(r, "modelId")
	if modelId == "" {
		http.Error(w, "missing modelId id", http.StatusBadRequest)
		return
	}

	postData := &RayLLMInferencePostBody{
		ModelName: modelId,
	}

	bodyToPost, err := json.Marshal(postData)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal model", http.StatusInternalServerError, "FailedToMarshalModel", err)
		return
	}

	result, _, code, err := commonHelper.CommonRequest(fmt.Sprintf("%s/%s", h.config.RayLLM.Endpoint, "deployment"), http.MethodGet, "", bodyToPost, r.Header, true, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get model", http.StatusInternalServerError, "FailedToGetModel", err)
		return
	}
	if code != http.StatusOK {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get model", code, "FailedToGetModel", fmt.Errorf("status code: %d", code))
		return
	}

	status := &RayLLMInferencePostBody{}
	err = json.Unmarshal(result, status)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal model", http.StatusInternalServerError, "FailedToUnmarshalModel", err)
		return
	}

	status.ModelName = modelId

	httpHelper.WriteResponseEntity(w, status)
}

func (h *RayLLMInferenceSouthAPIHandler) CreateModel(w http.ResponseWriter, r *http.Request) {
	// convert request body to []byte
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "FailedToReadRequestBody", err)
		return
	}

	parseBody := &RayLLMInferencePostBody{}
	err = json.Unmarshal(body, parseBody)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusInternalServerError, "FailedToUnmarshalRequestBody", err)
		return
	}

	// this api call will hang during the model deployment
	// so i will make it a goroutine
	// noway we are going to wait for this
	go func() {
		result, _, code, err := commonHelper.CommonRequest(fmt.Sprintf("%s/%s", h.config.RayLLM.Endpoint, "deployment"), http.MethodPost, "", body, r.Header, true, true, 3*time.Second)
		if err != nil {
			coreApiLog.Logger.Error("Failed to create model", "error", err)
			return
		}
		if code != http.StatusCreated && code != http.StatusOK {
			coreApiLog.Logger.Error("Failed to create model", "error", fmt.Errorf("status code: %d", code), "message", string(result))
			return
		}
		coreApiLog.Logger.Debug("Model created by api async call", "model", parseBody.ModelName)
	}()

	httpHelper.WriteResponseEntity(w, parseBody)
}

// delete models
func (h *RayLLMInferenceSouthAPIHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	modelId := chi.URLParam(r, "modelId")
	if modelId == "" {
		http.Error(w, "missing modelId id", http.StatusBadRequest)
		return
	}

	postData := &RayLLMInferencePostBody{
		ModelName: modelId,
	}

	bodyToPost, err := json.Marshal(postData)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal model", http.StatusInternalServerError, "FailedToMarshalModel", err)
		return
	}

	_, _, _, err = commonHelper.CommonRequest(fmt.Sprintf("%s/%s", h.config.RayLLM.Endpoint, "deployment"), http.MethodDelete, "", bodyToPost, r.Header, true, true, 3*time.Second)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete model", http.StatusInternalServerError, "FailedToDeleteModel", err)
		return
	}

	status := &RayLLMInferencePostBody{
		ModelName: modelId,
	}

	httpHelper.WriteResponseEntity(w, status)
}
