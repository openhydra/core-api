package route

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	"fmt"
	"net/http"
)

var (
	openhydraExtendAPIVersion = "extendV1"
)

var openhydraExtendFullPathPrefix = fmt.Sprintf("%s/%s/%s", openhydraPrefix, openhydraGroup, openhydraExtendAPIVersion)

func GetOpenhydraExtendRoute(config *config.Config) *ChiRouteBuilder {
	// init handler by passing the config
	return &ChiRouteBuilder{
		PathPrefix: openhydraExtendFullPathPrefix,
		MethodHandlers: []ChiSubRouteBuilder{
			{
				Method:  http.MethodGet,
				Pattern: "/settings/{settingId}",
				Handler: CreateGetSettingsHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "setting",
					Permission: privileges.PermissionSettingList,
				},
			},
			// list settings
			{
				Method:  http.MethodGet,
				Pattern: "/settings",
				Handler: CreateGetSettingsListHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "setting",
					Permission: privileges.PermissionSettingList,
				},
			},
			// patch settings
			{
				Method:  http.MethodPatch,
				Pattern: "/settings/{settingId}",
				Handler: CreatePatchSettingsHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "setting",
					Permission: privileges.PermissionSettingUpdate,
				},
			},
		},
	}
}
