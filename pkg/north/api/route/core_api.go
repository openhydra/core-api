package route

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	"fmt"
	"net/http"

	auth "core-api/pkg/core/auth"
)

var userProvider auth.IUserProvider
var groupProvider auth.IGroupProvider
var roleProvider auth.IRoleProvider

var coreApiFullPathPrefix = fmt.Sprintf("%s/%s/%s", corePrefix, coreGroup, coreAPIVersion)

func initOrGetUserProvider(serverConfig *config.Config) (auth.IUserProvider, error) {
	if userProvider == nil {
		userProvider, _ = auth.CreateUserProvider(serverConfig, auth.KeystoneAuthProvider)
	}
	return userProvider, nil
}

func initOrGetGroupProvider(serverConfig *config.Config) (auth.IGroupProvider, error) {
	if groupProvider == nil {
		groupProvider, _ = auth.CreateGroupProvider(serverConfig, auth.KeystoneAuthProvider)
	}
	return groupProvider, nil
}

func initOrGetRoleProvider(serverConfig *config.Config) (auth.IRoleProvider, error) {
	if roleProvider == nil {
		roleProvider, _ = auth.CreateRoleProvider(serverConfig, auth.KeystoneAuthProvider)
	}
	return roleProvider, nil
}

const (
	corePrefix     = "/apis"
	coreAPIVersion = "v1"
	coreGroup      = "core-api.openhydra.io"
)

func GetCoreApiRoute(config *config.Config) *ChiRouteBuilder {
	// init handler by passing the config
	return &ChiRouteBuilder{
		PathPrefix: coreApiFullPathPrefix,
		MethodHandlers: []ChiSubRouteBuilder{
			{
				Method:  http.MethodGet,
				Pattern: "/users/{userId}",
				Handler: CreateGetUserHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "user",
					Permission: privileges.PermissionUserList,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/users",
				Handler: CreateGetUsersHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "user",
					Permission: privileges.PermissionUserList,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/users/login",
				Handler: CreateLoginHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "user",
					Permission: 0,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/users",
				Handler: CreateCreateUserHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "user",
					Permission: privileges.PermissionUserCreate,
				},
			},
			// delete user
			{
				Method:  http.MethodDelete,
				Pattern: "/users/{userId}",
				Handler: CreateDeleteUserHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "user",
					Permission: privileges.PermissionUserDelete,
				},
			},
			// update user
			{
				Method:  http.MethodPut,
				Pattern: "/users/{userId}",
				Handler: CreateUpdateUserHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "user",
					Permission: privileges.PermissionUserUpdate,
				},
			},
			// upload user from csv or txt
			{
				Method:  http.MethodPost,
				Pattern: "/users/upload",
				Handler: CreateUploadUsersHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "user",
					Permission: privileges.PermissionUserCreate,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/groups/{groupId}",
				Handler: CreateGetGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupList,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/groups",
				Handler: CreateGetGroupsHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupList,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/groups",
				Handler: CreateCreateGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupCreate,
				},
			},
			{
				Method:  http.MethodPut,
				Pattern: "/groups/{groupId}",
				Handler: CreateUpdateGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupUpdate,
				},
			},
			{
				Method:  http.MethodDelete,
				Pattern: "/groups/{groupId}",
				Handler: CreateDeleteGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupDelete,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/groups/{groupId}/users",
				Handler: CreateGetGroupUsersHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupList,
				},
			},
			// add users to group
			{
				Method:  http.MethodPut,
				Pattern: "/groups/{groupId}/users",
				Handler: CreateAddUsersToGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupUpdate,
				},
			},
			// add user to group api
			{
				Method:  http.MethodPut,
				Pattern: "/groups/{groupId}/users/{userId}",
				Handler: CreateAddUserToGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupUpdate,
				},
			},
			// remove user from group api
			{
				Method:  http.MethodDelete,
				Pattern: "/groups/{groupId}/users/{userId}",
				Handler: CreateRemoveUserFromGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupDelete,
				},
			},
			// count users in each group
			{
				Method:  http.MethodGet,
				Pattern: "/groups/summary/all/count",
				Handler: CreateGetGroupSummaryHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupList,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/groups/summary/count",
			},
			{
				Method:  http.MethodGet,
				Pattern: "/groups/{groupId}/users/not-in-group/list",
				Handler: CreateGetGroupUsersNotInGroupHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "group",
					Permission: privileges.PermissionGroupList,
				},
			},
			// get role lists
			{
				Method:  http.MethodGet,
				Pattern: "/roles",
				Handler: CreateGetRolesHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "role",
					Permission: privileges.PermissionRoleList,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/roles/{roleId}",
				Handler: CreateGetRoleHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "role",
					Permission: privileges.PermissionRoleList,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/roles",
				Handler: CreateCreateRoleHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "role",
					Permission: privileges.PermissionRoleCreate,
				},
			},
			{
				Method:  http.MethodPut,
				Pattern: "/roles/{roleId}",
				Handler: CreateUpdateRoleHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "role",
					Permission: privileges.PermissionRoleUpdate,
				},
			},
			{
				Method:  http.MethodDelete,
				Pattern: "/roles/{roleId}",
				Handler: CreateDeleteRoleHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "role",
					Permission: privileges.PermissionRoleDelete,
				},
			},
			// flavor apis, only have
			{
				Method:  http.MethodGet,
				Pattern: "/flavors/{flavorId}",
				Handler: CreateGetFlavorHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "flavor",
					Permission: 0,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/sandboxes/{userId}",
				Handler: CreateGetOpenhydraSandboxHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "flavor",
					Permission: 0,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/summary/{summaryType}",
				Handler: CreateGetResourceSummaryHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "indexView",
					Permission: privileges.PermissionIndexPageViewResourceOverall,
				},
			},
			// get version info
			{
				Method:  http.MethodGet,
				Pattern: "/versions/{versionId}",
				Handler: CreateGetVersionHandler(config),
				ModuleAndPermission: ModuleAndPermission{
					Module:     "",
					Permission: 0,
				},
			},
		},
	}
}
