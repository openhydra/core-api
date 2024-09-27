package v1

type XInferenceModel struct {
	Id           string   `json:"id,omitempty"`
	Object       string   `json:"object,omitempty"`
	Created      int      `json:"created,omitempty"`
	OwnedBy      string   `json:"owned_by,omitempty"`
	Address      string   `json:"address,omitempty"`
	Accelerators []string `json:"accelerators,omitempty"`
	*XInferenceCommon
	Dimensions    int      `json:"dimensions,omitempty"`
	MaxTokens     int      `json:"max_tokens,omitempty"`
	Language      []string `json:"language,omitempty"`
	ModelLang     []string `json:"model_lang,omitempty"`
	ModelAbility  []string `json:"model_ability,omitempty"`
	ModelRevision string   `json:"model_revision,omitempty"`

	ModelDescription string `json:"model_description,omitempty"`

	ModelFamily string `json:"model_family,omitempty"`

	ModelHub      string `json:"model_hub,omitempty"`
	Revision      string `json:"revision,omitempty"`
	ContextLength int    `json:"context_length,omitempty"`
}

type XInferenceModelList struct {
	Object string            `json:"object,omitempty"`
	Data   []XInferenceModel `json:"data,omitempty"`
}

type XInferenceCommon struct {
	ModelFormat  string `json:"model_format,omitempty"`
	Quantization string `json:"quantization,omitempty"`
	ModelType    string `json:"model_type,omitempty"`
	Replica      int    `json:"replica,omitempty"`
	// should be always less equal than 1
	GpuMemoryUtilization float32 `json:"gpu_memory_utilization,omitempty"`
	ModelName            string  `json:"model_name,omitempty"`
	ModelSizeInBillions  int     `json:"model_size_in_billions,omitempty"`
}

type XInferenceModelLauncher struct {
	ModelUid         string `json:"model_uid,omitempty"`
	ModelEngine      string `json:"model_engine,omitempty"`
	NumberOfGpuToUse string `json:"n_gpu,omitempty"`
	RequestLimits    int    `json:"request_limits,omitempty"`
	PeftModelConfig  string `json:"peft_model_config,omitempty"`
	WorkerIp         string `json:"worker_ip,omitempty"`
	GpuIdx           string `json:"gpu_idx,omitempty"`
	DownloadHub      string `json:"download_hub,omitempty"`
	*XInferenceCommon
}

type XInferenceModelFontLauncher struct {
	ModelName string `json:"model_name,omitempty"`
	Template  string `json:"template,omitempty"`
}
