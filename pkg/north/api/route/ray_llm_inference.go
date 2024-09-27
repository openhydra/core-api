package route

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	"core-api/pkg/south"
	"fmt"
	"net/http"
)

const (
	rayLLMInferencePrefix     = "/apis"
	rayLLMInferenceAPIVersion = "v1"
	rayLLMInferenceGroup      = "ray-llm-inference.openhydra.io"
)

var rayLLMInferenceFullPathPrefix = fmt.Sprintf("%s/%s/%s", rayLLMInferencePrefix, rayLLMInferenceGroup, rayLLMInferenceAPIVersion)

func GetRayLLMInferenceRoute(config *config.Config) *ChiRouteBuilder {
	// init handler by passing the config
	rayLLMHandler := south.NewRayLLMInferenceSouthAPIHandler(config)
	return &ChiRouteBuilder{
		PathPrefix: rayLLMInferenceFullPathPrefix,
		MethodHandlers: []ChiSubRouteBuilder{
			{
				Method:  http.MethodGet,
				Pattern: "/models",
				Handler: rayLLMHandler.GetModels,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/models",
				Handler: rayLLMHandler.CreateModel,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagCreate,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/models/{modelId}",
				Handler: rayLLMHandler.GetModel,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagList,
				},
			},
			{
				Method:  http.MethodDelete,
				Pattern: "/models/{modelId}",
				Handler: rayLLMHandler.DeleteModel,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "rag",
					Permission: privileges.PermissionRagDelete,
				},
			},
		},
	}
}
