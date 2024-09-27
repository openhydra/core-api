package route

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	"core-api/pkg/south"
	"fmt"
	"net/http"
)

const (
	xinferencePrefix     = "/apis"
	xinferenceAPIVersion = "v1"
	xinferenceGroup      = "xinference.openhydra.io"
)

var xInferenceFullPath = fmt.Sprintf("%s/%s/%s", xinferencePrefix, xinferenceGroup, xinferenceAPIVersion)

func GetXInferenceRoute(config *config.Config) *ChiRouteBuilder {
	handler := south.NewXInferenceSouthAPIHandler(config)
	return &ChiRouteBuilder{
		PathPrefix: xInferenceFullPath,
		MethodHandlers: []ChiSubRouteBuilder{
			{
				Method:  http.MethodGet,
				Handler: handler.ListAllModels,
				Pattern: "/models",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "model",
					Permission: privileges.PermissionNotRequired,
				},
			},
			{
				Method:  http.MethodGet,
				Handler: handler.GetModel,
				Pattern: "/models/{modelId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "model",
					Permission: privileges.PermissionNotRequired,
				},
			},
			// create model
			{
				Method:  http.MethodPost,
				Handler: handler.CreateModel,
				Pattern: "/models",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "model",
					Permission: privileges.PermissionModelCreate,
				},
			},
			// delete model
			{
				Method:  http.MethodDelete,
				Handler: handler.DeleteModel,
				Pattern: "/models/{modelId}",
				ModuleAndPermission: ModuleAndPermission{
					Module:     "model",
					Permission: privileges.PermissionModelDelete,
				},
			},
		},
	}
}
