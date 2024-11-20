package route

import (
	"core-api/cmd/core-api-server/app/config"
	keystone "core-api/pkg/core/auth/provider/keystone/train"
	"core-api/pkg/core/privileges"
	coreApiLog "core-api/pkg/logger"
	chatV1 "core-api/pkg/north/api/chat/core/v1"
	conversationV1 "core-api/pkg/north/api/conversation/core/v1"
	coreFlavorV1 "core-api/pkg/north/api/flavor/core/v1"
	knowledgeBaseV1 "core-api/pkg/north/api/knowledge_base/core/v1"
	licenseV1 "core-api/pkg/north/api/license/core/v1"
	openhydraExtend "core-api/pkg/north/api/openhydra/extend/v1"
	summaryCoreV1 "core-api/pkg/north/api/summary/core/v1"
	coreUserV1 "core-api/pkg/north/api/user/core/v1"
	versionV1 "core-api/pkg/north/api/version/core/v1"
	xInferenceV1 "core-api/pkg/north/api/xinference/core/v1"
	"core-api/pkg/south"
	customErr "core-api/pkg/util/error"
	httpHelper "core-api/pkg/util/http"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	openhydraConfig "open-hydra-server-api/cmd/open-hydra-server/app/config"
	courseV1 "open-hydra-server-api/pkg/apis/open-hydra-api/course/core/v1"
	datasetV1 "open-hydra-server-api/pkg/apis/open-hydra-api/dataset/core/v1"
	deviceV1 "open-hydra-server-api/pkg/apis/open-hydra-api/device/core/v1"
	summaryV1 "open-hydra-server-api/pkg/apis/open-hydra-api/summary/core/v1"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var _ = deviceV1.Device{}
var _ = courseV1.Course{}
var _ = datasetV1.Dataset{}
var _ = coreFlavorV1.Flavor{}
var _ = summaryCoreV1.Summary{}
var _ = conversationV1.Conversation{}
var _ = knowledgeBaseV1.KnowledgeBase{}
var _ = openhydraConfig.OpenHydraServerConfig{}
var _ = chatV1.ChatPost{}
var _ = licenseV1.SystemInfo{}
var _ = versionV1.VersionInfo{}
var _ = xInferenceV1.XInferenceModelList{}

type CustomErrorUsersAddToGroup struct {
	Successes []coreUserV1.CoreUser `json:"successes,omitempty"`
	Failed    []coreUserV1.CoreUser `json:"failed,omitempty"`
}

/*
note we have to put all handler here
due to swag init command cannot merge annotation from multiple files
*/

// GET user detail
// @tags user
// @Summary show user detail
// @Description show user detail
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Param loadPermission query string false "load permission or not e.g. ?loadPermission=1"
// @Param loadPassword query string false "load password or not e.g. ?loadPassword=1"
// @Success 200 {object} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/users/{id}  [get]
func CreateGetUserHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// get user id from request path
		userId := chi.URLParam(r, "userId")
		if userId == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		userProvider, err := initOrGetUserProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}

		loadPermission := r.URL.Query().Get("loadPermission")
		var options map[string]struct{}
		if loadPermission != "" {
			options = map[string]struct{}{keystone.LoadPermission: {}}
		}

		loadPassword := r.URL.Query().Get("loadPassword")
		if loadPassword != "" {
			options = map[string]struct{}{keystone.LoadPasswd: {}}
		}

		if !config.CoreApiConfig.DisableAuth {
			// is allow show other user password
			_, _, err := south.SouthAuthorizationControlWithUser(r, w, "user", privileges.PermissionUserManageOtherUserResource, userId, "user")
			if err != nil {
				return
			}
		}

		var user *coreUserV1.CoreUser

		user, err = userProvider.GetUser(userId, options)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, user)
	}
}

// DELETE user
// @tags user
// @Summary delete user with given id
// @Description delete user with given id
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Success 200 {object} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/users/{userId}  [delete]
func CreateDeleteUserHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userId := chi.URLParam(r, "userId")
		if userId == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		userProvider, err := initOrGetUserProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}

		err = userProvider.DeleteUser(userId, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to delete user", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, &coreUserV1.CoreUser{
			Id: userId,
		})
	}
}

// PUT update user
// @tags user
// @Summary update user
// @Description update user
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Param request body coreUserV1.CoreUser true "user post"
// @Success 200 {object} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/users/{userId}  [put]
func CreateUpdateUserHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userId := chi.URLParam(r, "userId")
		if userId == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		userProvider, err := initOrGetUserProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}

		userPost := &coreUserV1.CoreUser{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		userPost.Id = userId

		err = json.Unmarshal(result, userPost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		userFound, err := userProvider.GetUser(userId, nil)
		if err != nil {
			// check is not found error
			if _, ok := err.(*customErr.NotFound); !ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusNotFound, "", err)
				return
			}
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusInternalServerError, "", err)
			return
		}

		if userPost.Password != "" {
			// ensure new changed password is not the same as the old one
			if userPost.Password == userFound.Password {
				httpHelper.WriteCustomErrorAndLog(w, "Password is the same as the old one", http.StatusBadRequest, "", fmt.Errorf("password is the same as the old one"))
				return
			}
		}

		err = userProvider.UpdateUser(userPost, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to update user", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, userPost)
	}
}

// GET group list
// @tags group
// @Summary list all groups
// @Description list all groups
// @Accept  json
// @Produce  json
// @Success 200 {array} coreUserV1.CoreGroup
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups  [get]
func CreateGetGroupsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}
		groups, err := groupProvider.GetGroups(nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get groups", http.StatusInternalServerError, "", err)
			return
		}

		if groups == nil {
			groups = []coreUserV1.CoreGroup{}
		}

		httpHelper.WriteResponseEntity(w, groups)
	}
}

// GET group detail
// @tags group
// @Summary get detail of group with given username
// @Description get detail of group with given username
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Success 200 {object} coreUserV1.CoreGroup
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}  [get]
func CreateGetGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		group, err := groupProvider.GetGroup(groupId, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get group", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, group)
	}
}

// POST create group
// @tags group
// @Summary create group
// @Description create group
// @Accept  json
// @Produce  json
// @Param request body coreUserV1.CoreGroup true "group post"
// @Success 200 {object} coreUserV1.CoreGroup
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups  [post]
func CreateCreateGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupPost := &coreUserV1.CoreGroup{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		err = json.Unmarshal(result, groupPost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		groupCreated, err := groupProvider.CreateGroup(groupPost, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, groupCreated)
	}
}

// PUT update group
// @tags group
// @Summary update group
// @Description update group
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Param request body coreUserV1.CoreGroup true "group post"
// @Success 200 {object} coreUserV1.CoreGroup
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}  [put]
func CreateUpdateGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupPost := &coreUserV1.CoreGroup{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		err = json.Unmarshal(result, groupPost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		err = groupProvider.UpdateGroup(groupPost, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to update group", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, groupPost)
	}
}

// DELETE delete group
// @tags group
// @Summary delete group
// @Description delete group
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Success 200 {object} coreUserV1.CoreGroup
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}  [delete]
func CreateDeleteGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		err = groupProvider.DeleteGroup(groupId, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to delete group", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, nil)
	}
}

// GET user in group
// @tags group
// @Summary list all users in group
// @Description list all users in group
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Success 200 {array} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}/users  [get]
func CreateGetGroupUsersHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		users, err := groupProvider.GetGroupUsers(groupId, nil)
		if err != nil {
			if _, ok := err.(*customErr.NotFound); ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get users from group due to resource not found", http.StatusNotFound, "", err)
				return
			}
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get group users", http.StatusInternalServerError, "", err)
			return
		}

		if users == nil {
			users = []coreUserV1.CoreUser{}
		}

		httpHelper.WriteResponseEntity(w, users)
	}
}

// Get user not in group
// @tags group
// @Summary list all users not in group
// @Description list all users not in group
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Success 200 {array} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}/users/not-in-group/list  [get]
func CreateGetGroupUsersNotInGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		users, err := groupProvider.GetGroupUsers(groupId, map[string]struct{}{keystone.ReverseGetGroupUsers: {}})
		if err != nil {
			if _, ok := err.(*customErr.NotFound); ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get users not in group due to resource not found", http.StatusNotFound, "", err)
				return
			}
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get group users not in", http.StatusInternalServerError, "", err)
			return
		}

		if users == nil {
			users = []coreUserV1.CoreUser{}
		}

		httpHelper.WriteResponseEntity(w, users)
	}
}

// PUT add user to group
// @tags group
// @Summary add user to group
// @Description add user to group
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Param userId path string true "user id"
// @Success 200 {object} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}/users/{userId}  [put]
func CreateAddUserToGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		userId := chi.URLParam(r, "userId")
		if userId == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		err = groupProvider.AddUserToGroup(userId, groupId)
		if err != nil {
			if _, ok := err.(*customErr.NotFound); ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to add user to group due to resource not found", http.StatusNotFound, "", err)
				return
			}
			httpHelper.WriteCustomErrorAndLog(w, "Failed to add user to group", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, nil)
	}
}

// DELETE remove user from group
// @tags group
// @Summary remove user from group
// @Description remove user from group
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Param userId path string true "user id"
// @Success 200 {object} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}/users/{userId}  [delete]
func CreateRemoveUserFromGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		userId := chi.URLParam(r, "userId")
		if userId == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		err = groupProvider.RemoveUserFromGroup(userId, groupId)
		if err != nil {
			if _, ok := err.(*customErr.NotFound); ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to remove user from group due to resource not found", http.StatusNotFound, "", err)
				return
			}
			httpHelper.WriteCustomErrorAndLog(w, "Failed to remove user from group", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, nil)
	}
}

// GET group summary
// @tags group
// @Summary show group summary
// @Description show group summary
// @Accept  json
// @Produce  json
// @Success 200 {object} coreUserV1.CoreGroupSummary
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/summary/all/count  [get]
func CreateGetGroupSummaryHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		summary, err := groupProvider.GetGroupSummary(nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get group summary", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, summary)
	}
}

// PUT add users to group
// @tags group
// @Summary add users to group
// @Description add users to group
// @Accept  json
// @Produce  json
// @Param groupId path string true "group id"
// @Param request body []coreUserV1.CoreUser true "users to add to group"
// @Success 200 {object} CustomErrorUsersAddToGroup
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/groups/{groupId}/users  [put]
func CreateAddUsersToGroupHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		groupId := chi.URLParam(r, "groupId")
		if groupId == "" {
			http.Error(w, "missing group id", http.StatusBadRequest)
			return
		}

		users := []coreUserV1.CoreUser{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		err = json.Unmarshal(result, &users)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		successes, failed, err := groupProvider.AddUsersToGroup(groupId, users)
		if err != nil {
			coreApiLog.Logger.Error("Failed to add users to group but will return 200", "error", err)
		}

		httpHelper.WriteResponseEntity(w, &CustomErrorUsersAddToGroup{
			Successes: successes,
			Failed:    failed,
		})
	}
}

// GET user list
// @tags user
// @Summary list all users
// @Description list all users
// @Accept  json
// @Produce  json
// @Param name query string false "filter with username e.g. ?name=ZhangSan"
// @Param group query string false "filter with group e.g. ?group=id1&group=id2"
// @Success 200 {array} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/users  [get]
func CreateGetUsersHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userProvider, err := initOrGetUserProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}

		var users []coreUserV1.CoreUser

		// get query name
		name := r.URL.Query().Get("name")
		if name != "" {
			userFound, err := userProvider.SearchUserByName(name, nil)
			if err != nil {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to search user by name", http.StatusInternalServerError, "", err)
				return
			}
			users = []coreUserV1.CoreUser{*userFound}
		} else {
			usersFound, err := userProvider.GetUsers(nil)
			if err != nil {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get users", http.StatusInternalServerError, "", err)
				return
			}
			users = usersFound
		}

		// get query groups
		// filter users by group
		groups := r.URL.Query()["group"]
		if groups != nil {
			flatGroups := map[string]struct{}{}
			for _, group := range groups {
				flatGroups[group] = struct{}{}
			}
			var filterGroups []coreUserV1.CoreUser
			for _, user := range users {
				for _, group := range user.Groups {
					if _, ok := flatGroups[group.Id]; ok {
						filterGroups = append(filterGroups, user)
						break
					}
				}
			}
			users = filterGroups
		}

		if len(users) == 0 {
			users = []coreUserV1.CoreUser{}
		}

		httpHelper.WriteResponseEntity(w, users)
	}
}

// GET device list
// @tags device
// @Summary show device list
// @Description show device list
// @Accept  json
// @Produce  json
// @Success 200 {array} deviceV1.Device
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/devices  [get]
func CreateOpenhydraGetDevicesHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// GET device detail
// @tags device
// @Summary get detail of device with given username
// @Description get detail of device with given username
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Success 200 {object} deviceV1.Device
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/devices/{userId}  [get]
func CreateOpenhydraGetDeviceHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// POST create device
// @tags device
// @Summary create device
// @Description create device
// @Accept  json
// @Produce  json
// @Param request body deviceV1.Device true "device post"
// @Success 200 {object} deviceV1.Device
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/devices  [post]
func CreateOpenhydraCreateDeviceHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// DELETE delete device
// @tags device
// @Summary delete device with given username
// @Description delete device with given username
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Success 200 {object} deviceV1.Device
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/devices/{userId}  [delete]
func CreateOpenhydraDeleteDeviceHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// GET course list
// @tags course
// @Summary show course list
// @Description show course list
// @Accept  json
// @Produce  json
// @Success 200 {array} courseV1.Course
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/courses  [get]
func CreateOpenhydraGetCoursesHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// GET course detail
// @tags course
// @Summary show course detail
// @Description show course detail
// @Accept  json
// @Produce  json
// @Param courseId path string true "course id"
// @Success 200 {object} courseV1.Course
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/courses/{courseId}  [get]
func CreateOpenhydraGetCourseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// POST create course
// @tags course
// @Summary create course
// @Description create course
// @Accept  json
// @Produce  json
// @Param request body courseV1.Course true "course post"
// @Success 200 {object} courseV1.Course
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/courses  [post]
func CreateOpenhydraCreateCourseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// DELETE delete course
// @tags course
// @Summary delete course
// @Description delete course
// @Accept  json
// @Produce  json
// @Param courseId path string true "course id"
// @Success 200 {object} courseV1.Course
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/courses/{courseId}  [delete]
func CreateOpenhydraDeleteCourseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// GET dataset list
// @tags dataset
// @Summary show dataset list
// @Description show dataset list
// @Accept  json
// @Produce  json
// @Success 200 {array} datasetV1.Dataset
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/datasets  [get]
func CreateOpenhydraGetDatasetsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// GET dataset detail
// @tags dataset
// @Summary show dataset detail
// @Description show dataset detail
// @Accept  json
// @Produce  json
// @Param datasetId path string true "dataset id"
// @Success 200 {object} datasetV1.Dataset
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/datasets/{datasetId}  [get]
func CreateOpenhydraGetDatasetHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// POST create dataset
// @tags dataset
// @Summary create dataset
// @Description create dataset
// @Accept  json
// @Produce  json
// @Param request body datasetV1.Dataset true "login params"
// @Success 200 {object} datasetV1.Dataset
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/datasets  [post]
func CreateOpenhydraCreateDatasetHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// DELETE delete dataset
// @tags dataset
// @Summary delete dataset
// @Description delete dataset
// @Accept  json
// @Produce  json
// @Param datasetId path string true "dataset id"
// @Success 200 {object} datasetV1.Dataset
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/v1/datasets/{datasetId}  [delete]
func CreateOpenhydraDeleteDatasetHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	// for calling to south api we only define a fake handler
	// to generate swagger doc
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

// POST user login
// @tags user
// @Summary user login
// @Description user login
// @Accept  json
// @Produce  json
// @Param request body coreUserV1.CoreUser true "login params"
// @Success 200 {object} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/users/login  [post]
func CreateLoginHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userProvider, err := initOrGetUserProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}
		userPost := &coreUserV1.CoreUser{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		err = json.Unmarshal(result, userPost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		if userPost.Name == "" || userPost.Password == "" {
			httpHelper.WriteCustomErrorAndLog(w, "Missing user name or password", http.StatusBadRequest, "", nil)
			return
		}

		user, err := userProvider.LoginUser(userPost.Name, userPost.Password)
		if err != nil {
			// check err is unauthorized
			if _, ok := err.(*customErr.Unauthorized); ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to login user due to unauthorized", http.StatusUnauthorized, "", err)
				return
			}
			httpHelper.WriteCustomErrorAndLog(w, "Failed to login user", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, user)
	}
}

// POST user create
// @tags user
// @Summary create user
// @Description create user
// @Accept  json
// @Produce  json
// @Param request body coreUserV1.CoreUser true "user post"
// @Success 200 {object} coreUserV1.CoreUser
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/users  [post]
func CreateCreateUserHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userProvider, err := initOrGetUserProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}

		userPost := &coreUserV1.CoreUser{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		err = json.Unmarshal(result, userPost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		retUserPost, err := userProvider.CreateUser(userPost, nil)
		if err != nil {
			// check error is not found error
			if _, ok := err.(*customErr.NotFound); ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to create user due to role not found", http.StatusNotFound, "", err)
				return
			}
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, retUserPost)
	}
}

// GET flavor detail
// @tags flavor
// @Summary show flavor detail
// @Description show flavor detail
// @Accept  json
// @Produce  json
// @Param flavorId path string true "flavor id"
// @Success 200 {object} coreFlavorV1.Flavor
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/flavors/{flavorId}  [get]
func CreateGetFlavorHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// get flavor id from request path
		// due to openhydra api issue  this flavor id is not used
		// but we still need it as a get detail standard
		flavorId := chi.URLParam(r, "flavorId")
		if flavorId == "" {
			http.Error(w, "missing flavor id", http.StatusBadRequest)
			return
		}

		plugin, err := GetOpenhydraPlugin()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get openhydra plugin", http.StatusInternalServerError, "", err)
			return
		}

		// get sumup list from openhydra sumup
		sumup, err := south.RequestKubeApiserverWithServiceAccountAndParseToT[summaryV1.SumUp](config, south.SumupGVR, "", nil, http.MethodGet, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get sumup list", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, &coreFlavorV1.Flavor{
			Plugin: plugin,
			SumUps: sumup,
		})
	}
}

// GET role list
// @tags role
// @Summary show role list
// @Description show role list
// @Accept  json
// @Produce  json
// @Success 200 {array} coreUserV1.CoreRole
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/roles  [get]
func CreateGetRolesHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		roleProvider, err := initOrGetRoleProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create role provider", http.StatusInternalServerError, "", err)
			return
		}
		roles, err := roleProvider.GetRoles(nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get roles", http.StatusInternalServerError, "", err)
			return
		}

		if roles == nil {
			roles = []coreUserV1.CoreRole{}
		}

		httpHelper.WriteResponseEntity(w, roles)
	}
}

// GET role detail
// @tags role
// @Summary show role detail
// @Description show role detail
// @Accept  json
// @Produce  json
// @Param roleId path string true "role id"
// @Success 200 {object} coreUserV1.CoreRole
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/roles/{roleId}  [get]
func CreateGetRoleHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		roleProvider, err := initOrGetRoleProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create role provider", http.StatusInternalServerError, "", err)
			return
		}

		roleId := chi.URLParam(r, "roleId")
		if roleId == "" {
			http.Error(w, "missing role id", http.StatusBadRequest)
			return
		}

		role, err := roleProvider.GetRole(roleId, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get role", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, role)
	}
}

// POST create role
// @tags role
// @Summary create role
// @Description create role
// @Accept  json
// @Produce  json
// @Param request body coreUserV1.CoreRole true "role post"
// @Success 200 {object} coreUserV1.CoreRole
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/roles  [post]
func CreateCreateRoleHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		roleProvider, err := initOrGetRoleProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create role provider", http.StatusInternalServerError, "", err)
			return
		}

		rolePost := &coreUserV1.CoreRole{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		err = json.Unmarshal(result, rolePost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		role, err := roleProvider.CreateRole(rolePost, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create role", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, role)
	}
}

// PUT update role
// @tags role
// @Summary update role
// @Description update role
// @Accept  json
// @Produce  json
// @Param roleId path string true "role id"
// @Param request body coreUserV1.CoreRole true "role post"
// @Success 200 {object} coreUserV1.CoreRole
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/roles/{roleId}  [put]
func CreateUpdateRoleHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		roleProvider, err := initOrGetRoleProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create role provider", http.StatusInternalServerError, "", err)
			return
		}

		rolePost := &coreUserV1.CoreRole{}
		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusBadRequest, "", err)
			return
		}

		err = json.Unmarshal(result, rolePost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		roleId := chi.URLParam(r, "roleId")
		if roleId == "" {
			http.Error(w, "missing role id", http.StatusBadRequest)
			return
		}

		rolePost.Id = roleId

		err = roleProvider.UpdateRole(rolePost, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to update role", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, rolePost)
	}
}

// DELETE delete role
// @tags role
// @Summary delete role
// @Description delete role
// @Accept  json
// @Produce  json
// @Param roleId path string true "role id"
// @Success 200 {object} coreUserV1.CoreRole
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/roles/{roleId}  [delete]
func CreateDeleteRoleHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		roleProvider, err := initOrGetRoleProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create role provider", http.StatusInternalServerError, "", err)
			return
		}

		roleId := chi.URLParam(r, "roleId")
		if roleId == "" {
			http.Error(w, "missing role id", http.StatusBadRequest)
			return
		}

		err = roleProvider.DeleteRole(roleId, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to delete role", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, nil)
	}
}

// GET ray_llm models
// @tags ray-llm-inference
// @Summary show ray_llm models
// @Description show ray_llm models
// @Accept  json
// @Produce  json
// @Param llmType query string false "either llm_models or embedding_models or all e.g. ?llmType=llm_models"
// @Success 200 {array} string
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/ray-llm-inference.openhydra.io/v1/models  [get]
func CreateGetRayLlmModelsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET ray_llm model detail
// @tags ray-llm-inference
// @Summary show ray_llm model detail
// @Description show ray_llm model detail
// @Accept  json
// @Produce  json
// @Param modelId path string true "model id"
// @Success 200 {object} south.RayDeployment
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/ray-llm-inference.openhydra.io/v1/models/{modelId}  [get]
func CreateGetRayLlmModelHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// DELETE ray_llm model
// @tags ray-llm-inference
// @Summary delete ray_llm model
// @Description delete ray_llm model
// @Accept  json
// @Produce  json
// @Param modelId path string true "model id"
// @Success 200 {object} south.RayDeployment
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/ray-llm-inference.openhydra.io/v1/models/{modelId}  [delete]
func CreateDeleteRayLlmModelHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST create model
// @tags ray-llm-inference
// @Summary create model
// @Description create model
// @Accept  json
// @Produce  json
// @Param request body south.RayDeployment true "model post"
// @Success 200 {object} south.RayDeployment
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/ray-llm-inference.openhydra.io/v1/models  [post]
func CreateCreateRayLlmModelHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET openhydra sandbox list
// @tags flavor
// @Summary show openhydra sandbox list
// @Description show openhydra sandbox list
// @Accept  json
// @Produce  json
// @Param loadRunningStat query string false "whether to load running sandbox stat e.g. ?loadRunningStat=1"
// @Param userId path string true "user id"
// @Success 200 {object} openhydraExtend.WrapperSandbox
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/sandboxes/{userId}  [get]
func CreateGetOpenhydraSandboxHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// get user id from request path
		userId := chi.URLParam(r, "userId")
		if userId == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		device, err := south.RequestKubeApiserverWithServiceAccountAndParseToT[deviceV1.Device](config, south.DevicesGVR, userId, nil, http.MethodGet, r.Header)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get device", http.StatusInternalServerError, "", err)
			return
		}

		pluginList, err := GetOpenhydraPlugin()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get openhydra plugin", http.StatusInternalServerError, "", err)
			return
		}

		wrappedSandbox := &openhydraExtend.WrapperSandbox{
			Sandboxes: pluginList,
		}

		for sandboxName := range pluginList.Sandboxes {
			if sandboxName == device.Spec.SandboxName {
				wrappedSandbox.RunningSandboxName = sandboxName
				break
			}
		}
		httpHelper.WriteResponseEntity(w, wrappedSandbox)
	}
}

// GET resource summary
// @tags summary
// @Summary show resource summary
// @Description show resource summary
// @Accept  json
// @Produce  json
// @Param summaryType path string true "summary type not you can use any kind of string"
// @Success 200 {object} summaryCoreV1.Summary
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/summary/{summaryType}  [get]
func CreateGetResourceSummaryHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := GetSummary()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get summary", http.StatusInternalServerError, "", err)
			return
		}

		// get sumup list from openhydra sumup
		sumup, err := south.RequestKubeApiserverWithServiceAccountAndParseToT[summaryV1.SumUp](config, south.SumupGVR, "", nil, http.MethodGet, nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get sumup list", http.StatusInternalServerError, "", err)
			return
		}

		result.GpuResourceSumUp = sumup.Spec.GpuResourceSumUp

		openhydraConfig, err := GetOpenhydraConfigMap()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get openhydra config map", http.StatusInternalServerError, "", err)
			return
		}

		result.OpenHydraConfig = openhydraConfig

		shareStat := map[string]int{}
		for key := range result.GpuResourceSumUp {
			if key != "nvidia.com/gpu" {
				shareStat[key] = 1
			} else {
				result, err := GetNvidiaGpuShare()
				if err != nil {
					httpHelper.WriteCustomErrorAndLog(w, "Failed to get nvidia gpu share", http.StatusInternalServerError, "", err)
					return
				}
				shareStat[key] = result
			}
		}

		result.GpuResourceShare = shareStat

		// get config map with k8s helper

		httpHelper.WriteResponseEntity(w, result)
	}
}

// GET conversations
// @tags rag
// @Summary show conversations
// @Description show conversations
// @Accept  json
// @Produce  json
// @Param chatTypes query string true "chat type e.g. chatTypes=llm_chat | file_chat | kb_chat | all or chatTypes=llm_chat,file_chat"
// @Success 200 {object} []conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations  [get]
func CreateGetConversationsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST create conversation
// @tags rag
// @Summary create conversation
// @Description create conversation
// @Accept  json
// @Produce  json
// @Param request body conversationV1.Conversation true "conversation post"
// @Success 200 {object} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations  [post]
func CreateCreateConversationHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET conversation by id
// @tags rag
// @Summary show conversation by id
// @Description show conversation by id
// @Accept  json
// @Produce  json
// @Param conversationId path string true "conversation id"
// @Success 200 {object} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations/{conversationId}  [get]
func CreateGetConversationByIdHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// DELETE conversation by id
// @tags rag
// @Summary delete conversation by id
// @Description delete conversation by id
// @Accept  json
// @Produce  json
// @Param conversationId path string true "conversation id"
// @Success 200 {object} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations/{conversationId}  [delete]
func CreateDeleteConversationByIdHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// PATCH conversation by id
// @tags rag
// @Summary patch conversation by id
// @Description patch conversation by id
// @Accept  json
// @Produce  json
// @Param conversationId path string true "conversation id"
// @Param request body conversationV1.Conversation true "conversation patch"
// @Success 200 {object} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations/{conversationId}  [patch]
func CreatePatchConversationByIdHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET conversation of user
// @tags rag
// @Summary show conversation of user
// @Description show conversation of user
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Param chatTypes query string true "chat type e.g. chatTypes=llm_chat | file_chat | kb_chat | all or chatTypes=llm_chat,file_chat"
// @Success 200 {array} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations/users/{userId}  [get]
func CreateGetConversationOfUserHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// DELETE conversation of user
// @tags rag
// @Summary delete conversation of user
// @Description delete conversation of user
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Success 200 {array} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations/users/{userId}  [delete]
func CreateDeleteConversationOfUserHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST create knowledge base
// @tags rag
// @Summary create knowledge base
// @Description create knowledge base
// @Accept  json
// @Produce  json
// @Param request body knowledgeBaseV1.KnowledgeBase true "knowledge base post"
// @Success 200 {object} knowledgeBaseV1.KnowledgeBase
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases  [post]
func CreateCreateKnowledgeBaseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// DELETE knowledge base
// @tags rag
// @Summary delete knowledge base
// @Description delete knowledge base
// @Accept  json
// @Produce  json
// @Param knowledgeBaseName path string true "knowledge base name"
// @Success 200 {object} knowledgeBaseV1.KnowledgeBase
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases/{knowledgeBaseId}  [delete]
func CreateDeleteKnowledgeBaseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// PATCH knowledge base
// @tags rag
// @Summary patch knowledge base
// @Description patch knowledge base
// @Accept  json
// @Produce  json
// @Param knowledgeBaseName path string true "knowledge base name"
// @Param request body knowledgeBaseV1.KnowledgeBase true "knowledge base patch"
// @Success 200 {object} knowledgeBaseV1.KnowledgeBase
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases/{knowledgeBaseId}  [patch]
func CreatePatchKnowledgeBaseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET knowledge bases of user
// @tags rag
// @Summary show knowledge bases of user
// @Description show knowledge bases of user
// @Accept  json
// @Produce  json
// @Param userId path string true "user id"
// @Param appendKB query string false "either groupedKB or publicKB or publicKB,groupedKB e.g. ?appendKB=publicKB,groupedKB"
// @Success 200 {array} knowledgeBaseV1.KnowledgeBase
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases/users/{userId}  [get]
func CreateGetKnowledgeBasesHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET public knowledge bases
// @tags rag
// @Summary show public knowledge bases
// @Description show public knowledge bases
// @Accept  json
// @Produce  json
// @Success 200 {array} knowledgeBaseV1.KnowledgeBase
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases  [get]
func CreateGetPublicKnowledgeBasesHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST upload file to knowledge base
// @tags rag
// @Summary upload file to knowledge base
// @Description upload file to knowledge base
// @Accept  multipart/form-data
// @Produce  json
// @Param knowledgeBaseId path string true "knowledge base name"
// @Param kb_id body string true "knowledge base id which will direct forward to south api"
// @Param kb_name body string true "knowledge base name which will direct forward to south api"
// @Param files formData file true "array of files to upload, support multiple files"
// @Success 200 {object} knowledgeBaseV1.KnowledgeBase
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases/{knowledgeBaseId}/upload  [post]
func CreateUploadFileToKnowledgeBaseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET settings
// @tags settings
// @Summary get settings detail
// @Description get settings detail
// @Accept  json
// @Produce  json
// @Param settingId path string true "setting id"
// @Success 200 {object} openhydraConfig.OpenHydraServerConfig
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/extendV1/settings/{settingId}  [get]
func CreateGetSettingsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		openhydraConfig, err := GetOpenhydraConfigMap()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get openhydra config map", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, openhydraConfig)
	}
}

// GET settings list
// @tags settings
// @Summary get settings list
// @Description get settings list
// @Accept  json
// @Produce  json
// @Success 200 {array} openhydraConfig.OpenHydraServerConfig
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/extendV1/settings  [get]
func CreateGetSettingsListHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		currentConfig, err := GetOpenhydraConfigMap()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get openhydra config map", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, []openhydraConfig.OpenHydraServerConfig{*currentConfig})
	}
}

// PATCH settings
// @tags settings
// @Summary patch settings
// @Description patch settings
// @Accept  json
// @Produce  json
// @Param settingId path string true "setting id"
// @Param request body openhydraConfig.OpenHydraServerConfig true "setting patch"
// @Param saveSection query string false "either storage,runtimeResource,serverIp,gpuType  e.g. ?saveSection=storage"
// @Success 200 {object} openhydraConfig.OpenHydraServerConfig
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/open-hydra-server.openhydra.io/extendV1/settings/{settingId}  [patch]
func CreatePatchSettingsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		settingId := chi.URLParam(r, "settingId")
		if settingId == "" {
			http.Error(w, "missing setting id", http.StatusBadRequest)
			return
		}

		saveSection := r.URL.Query().Get("saveSection")
		if saveSection == "" {
			httpHelper.WriteCustomErrorAndLog(w, "Missing save section", http.StatusBadRequest, "", nil)
			return
		}

		if saveSection != "storage" && saveSection != "runtimeResource" && saveSection != "serverIp" && saveSection != "gpuType" {
			httpHelper.WriteCustomErrorAndLog(w, "Invalid save section", http.StatusBadRequest, "", nil)
			return
		}

		// convert request body to string
		result, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read request body", http.StatusInternalServerError, "", err)
			return
		}

		postConfig := &openhydraConfig.OpenHydraServerConfig{}
		err = json.Unmarshal(result, postConfig)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal request body", http.StatusBadRequest, "", err)
			return
		}

		err = ValidateOpenhydraConfig(postConfig, OpenhydraSettingSection(saveSection))
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to validate openhydra config", http.StatusBadRequest, "", err)
			return
		}

		currentConfig, err := GetOpenhydraConfigMap()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get openhydra config map", http.StatusInternalServerError, "", err)
			return
		}

		switch OpenhydraSettingSection(saveSection) {
		case OpenhydraSettingSectionStorage:
			currentConfig.PublicDatasetBasePath = postConfig.PublicDatasetBasePath
			currentConfig.PublicCourseBasePath = postConfig.PublicCourseBasePath
			currentConfig.WorkspacePath = postConfig.WorkspacePath
		case OpenhydraSettingSectionRuntimeResource:
			currentConfig.CpuOverCommitRate = postConfig.CpuOverCommitRate
			currentConfig.MemoryOverCommitRate = postConfig.MemoryOverCommitRate
			currentConfig.DefaultGpuPerDevice = postConfig.DefaultGpuPerDevice
			currentConfig.DefaultCpuPerDevice = postConfig.DefaultCpuPerDevice
			currentConfig.DefaultRamPerDevice = postConfig.DefaultRamPerDevice
		case OpenhydraSettingSectionServerIp:
			currentConfig.ServerIP = postConfig.ServerIP
		case OpenhydraSettingSectionGPUType:
			currentConfig.GpuResourceKeys = postConfig.GpuResourceKeys
		}

		err = UpdateOpenhydraConfigMap(currentConfig)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to update openhydra config map", http.StatusInternalServerError, "", err)
			return
		}

		httpHelper.WriteResponseEntity(w, currentConfig)
	}
}

// POST create chats
// @tags rag
// @Summary create chats
// @Description create chats
// @Accept  json
// @Produce  json
// @Param request body chatV1.ChatPost true "chat post"
// @Success 200 {array} string
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/chats  [post]
func CreateCreateChatsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET version info
// @tags version
// @Summary show version info
// @Description show version info
// @Accept  json
// @Produce  json
// @Param versionId path string true "version id"
// @Success 200 {object} versionV1.VersionInfo
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/core-api.openhydra.io/v1/versions/{versionId}  [get]
func CreateGetVersionHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		versionInfo := versionV1.VersionInfo{
			ReleaseVersion: config.CoreApiConfig.ReleaseVersion,
			GitVersion:     config.CoreApiConfig.GitVersion,
		}

		// random 1-100
		randomNumber := rand.Intn(100) + 1
		if randomNumber <= 15 {
			funMap := map[string]string{
				"爱骑自行车的 🚴🏾Frank🚴🏾 saids:":   "车卡路里了。。。只能做地铁上班",
				"不务正业的 🐑羊成群🐑 saids:":        "nvidia 的显卡太难伺候了，我都没时间写 bug 了",
				"洗剪吹一次只要 38 的首席 ✀诗梦✀ saids": "这页面大部分都是我设计的，如果有不好看的肯定是他们改了设计",
				"爱喝奶茶的 🧋玥璇🧋 saids":          "产品经理不怎么好玩,我先撤了，家人们保重",
				"来不急做界面的 😕辉银😕 saids":        "我就是个后端，不会写界面",
			}
			if randomNumber < 10 {
				funMap["爱打游戏的 🎮小沈🎮 saids"] = "最近黑猴子通关了，我就出去跑跑客户吧"
				funMap["始作俑者 👨🏻‍🏫谢老师👨🏻‍🏫 saids"] = "小沈来我这里有商机"
				funMap["被部署搞的头很的 🫥凌杰🫥 saids"] = "再不好好测试部署脚本，我就...我就..."
				funMap["啰嗦 🤔jn🤔 saids"] = "好好写代码我会来 review 的"
				funMap["还有一个 🤖机器人🤖 saids"] = "我是机器人，我是机器人，我是机器人"
				funMap["以及其他等等人 saids"] = "谁是等等啊，我 Eason 也经常宣传 openhydra 的好嘛"
			}
			versionInfo.Fun = funMap
		}

		httpHelper.WriteResponseEntity(w, versionInfo)
	}
}

// GET xinference models
// @tags xinference
// @Summary show xinference models
// @Description show xinference models
// @Accept  json
// @Produce  json
// @Success 200 {array} xInferenceV1.XInferenceModelList
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/xinference.openhydra.io/v1/models  [get]
func CreateGetXinferenceModelsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET xinference model detail
// @tags xinference
// @Summary show xinference model detail
// @Description show xinference model detail
// @Accept  json
// @Produce  json
// @Param modelId path string true "model id"
// @Success 200 {object} xInferenceV1.XInferenceModel
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/xinference.openhydra.io/v1/models/{modelId}  [get]
func CreateGetXinferenceModelHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST create xinference model
// @tags xinference
// @Summary create xinference model
// @Description create xinference model
// @Accept  json
// @Produce  json
// @Param request body xInferenceV1.XInferenceModelFontLauncher true "model post"
// @Success 200 {object} xInferenceV1.XInferenceModelFontLauncher
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/xinference.openhydra.io/v1/models  [post]
func CreateCreateXinferenceModelHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// DELETE xinference model
// @tags xinference
// @Summary delete xinference model
// @Description delete xinference model
// @Accept  json
// @Produce  json
// @Param modelId path string true "model id"
// @Success 200 {object} xInferenceV1.XInferenceModel
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/xinference.openhydra.io/v1/models/{modelId}  [delete]
func CreateDeleteXinferenceModelHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET messages of a conversation
// @tags rag
// @Summary show messages of a conversation
// @Description show messages of a conversation
// @Accept  json
// @Produce  json
// @Param conversationId path string true "conversation id"
// @Success 200 {array} conversationV1.ConversationMessage
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations/{conversationId}/messages  [get]
func CreateGetMessagesOfConversationHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET messages by message id
// @tags rag
// @Summary show messages by message id
// @Description show messages by message id
// @Accept  json
// @Produce  json
// @Param messageId path string true "message id"
// @Param conversationId path string true "conversation id"
// @Success 200 {object} conversationV1.ConversationMessage
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/conversations/{conversationId}/messages/{messageId}  [get]
func CreateGetMessagesByMessageIdHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET chat quick starts
// @tags rag
// @Summary show chat quick starts
// @Description show chat quick starts
// @Accept  json
// @Produce  json
// @Success 200 {object} chatV1.ChatQuickStarts
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/chat_quick_starts  [get]
func CreateGetChatQuickStartsHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST create a quick file chat
// @tags rag
// @Summary create a quick file chat
// @Description create a quick file chat
// @Accept  json
// @Produce  json
// @Param request body conversationV1.Conversation true "conversation post"
// @Success 200 {object} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router 	/apis/rag.openhydra.io/v1/quick_file_chat  [post]
func CreateQuickFileChatHandler(config *config.Config, stopChan <-chan struct{}) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST create a file chat
// @tags rag
// @Summary create a file chat alone with conversation and file
// @Description create a file chat alone with conversation and file
// @Accept  multipart/form-data
// @Produce  json
// @Param userId path string true "user id"
// @Param user_id body string true "user id"
// @Param files body string true "file uploads"
// @Success 200 {object} conversationV1.Conversation
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router 	/apis/rag.openhydra.io/v1/file_chat/{userId}  [post]
func CreateFileChatHandler(config *config.Config, stopChan <-chan struct{}) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST file chat
// @tags rag
// @Summary chat with a file
// @Description chat with a file
// @Accept  json
// @Produce  json
// @Param request body chatV1.FileChatPost true "message post"
// @Success 200 {array} string
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router 	/apis/rag.openhydra.io/v1/file_chat  [post]
func CreateFileChatPostHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST kb chat
// @tags rag
// @Summary chat with a knowledge base
// @Description chat with a knowledge base
// @Accept  json
// @Produce  json
// @Param request body chatV1.KbChatPost true "message post"
// @Success 200 {array} string
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router 	/apis/rag.openhydra.io/v1/knowledge_bases/{knowledgeBaseId}/kb_chat  [post]
func CreateKnowledgeBaseChatPostHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET kb detail
// @tags rag
// @Summary show knowledge base detail
// @Description show knowledge base detail
// @Accept  json
// @Produce  json
// @Param knowledgeBaseName path string true "knowledge base name"
// @Success 200 {object} knowledgeBaseV1.KnowledgeBase
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases/{knowledgeBaseId}  [get]
func CreateGetKnowledgeBaseHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// GET knowledge base files
// @tags rag
// @Summary show knowledge base files
// @Description show knowledge base files
// @Accept  json
// @Produce  json
// @Param knowledgeBaseName path string true "knowledge base name"
// @Success 200 {object} knowledgeBaseV1.KnowledgeBaseFileList
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases/{knowledgeBaseId}/files  [get]
func CreateGetKnowledgeBaseFilesHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// DELETE knowledge base file
// @tags rag
// @Summary delete knowledge base file
// @Description delete knowledge base file
// @Accept  json
// @Produce  json
// @Param knowledgeBaseName path string true "knowledge base name"
// @Param fileIds body knowledgeBaseV1.KnowledgeBaseFilesToDelete true "files to be delete"
// @Success 200 {object} knowledgeBaseV1.KnowledgeBaseCommonResult
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 404 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router /apis/rag.openhydra.io/v1/knowledge_bases/{knowledgeBaseId}/files  [delete]
func CreateDeleteKnowledgeBaseFileHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// POST upload users
// @tags user
// @Summary upload users from csv or txt split by comma
// @Description upload users from csv or txt split by comma
// @Accept  multipart/form-data
// @Produce  json
// @Param file body string true "file uploads"
// @Success 200 {array} string
// @Failure 400 {object} httpHelper.CustomError
// @Failure 403 {object} httpHelper.CustomError
// @Failure 500 {object} httpHelper.CustomError
// @Router 	/apis/core-api.openhydra.io/v1/users/upload  [post]
func CreateUploadUsersHandler(config *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// this is a very expensive operation, we need to check the user,role,group at the same time
		// this operation should be called at the very begin of system init
		// all resource load from csv file will be created if not exist
		// all resource load from csv will be ignored if already exist

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get file from request", http.StatusBadRequest, "", err)
			return
		}

		fileExt := filepath.Ext(fileHeader.Filename)

		if fileExt != ".csv" && fileExt != ".txt" {
			httpHelper.WriteCustomErrorAndLog(w, "Invalid file extension expected to one of csv or txt", http.StatusBadRequest, "", fmt.Errorf("invalid file extension %s", fileExt))
			return
		}

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to parse CSV file", http.StatusInternalServerError, "", err)
			return
		}

		// get all users
		userProvider, err := initOrGetUserProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create user provider", http.StatusInternalServerError, "", err)
			return
		}

		// get all roles
		roleProvider, err := initOrGetRoleProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create role provider", http.StatusInternalServerError, "", err)
			return
		}

		// get all group
		groupProvider, err := initOrGetGroupProvider(config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to create group provider", http.StatusInternalServerError, "", err)
			return
		}

		allUsers, err := userProvider.GetUsers(nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get all users", http.StatusInternalServerError, "", err)
			return
		}

		allRoles, err := roleProvider.GetRoles(nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get all roles", http.StatusInternalServerError, "", err)
			return
		}

		allGroup, err := groupProvider.GetGroups(nil)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get all groups", http.StatusInternalServerError, "", err)
			return
		}

		flatUser := make(map[string]coreUserV1.CoreUser)
		for _, user := range allUsers {
			flatUser[user.Name] = user
		}

		flatRoles := make(map[string]coreUserV1.CoreRole)
		for _, role := range allRoles {
			flatRoles[role.Name] = role
		}

		flatGroup := make(map[string]coreUserV1.CoreGroup)
		for _, group := range allGroup {
			flatGroup[group.Name] = group
		}

		var result []string
		for index, record := range records {
			if index == 0 {
				// skip column name
				continue
			}

			// 0 = username
			// 1 = password
			// 2 = role
			// 3 = group
			// 4 = description
			var recordIssue []string

			// data security check
			// do not create a user if user set role to aes-admin
			if record[2] == "aes-admin" {
				coreApiLog.Logger.Warn("skip this user due to role is aes-admin is forbidden")
				continue
			}

			if len(record) != 5 {
				httpHelper.WriteCustomErrorAndLog(w, fmt.Sprintf("Invalid record format expected column to be 5 got %d", len(record)), http.StatusBadRequest, "", fmt.Errorf("invalid record format expected column to be 5 got %d", len(record)))
				return
			}

			if record[0] == "" {
				recordIssue = append(recordIssue, fmt.Sprintf("user %s failed to create due to:", record[0]))
				recordIssue = append(recordIssue, "username is empty")
			}

			if record[1] == "" {

				if len(recordIssue) == 0 {
					recordIssue = append(recordIssue, fmt.Sprintf("user %s failed to create due to:", record[0]))
				}
				recordIssue = append(recordIssue, "password is empty")
			}

			if record[2] == "" {
				if len(recordIssue) == 0 {
					recordIssue = append(recordIssue, fmt.Sprintf("user %s failed to create due to:", record[0]))
				}
				recordIssue = append(recordIssue, "role is empty")
			}

			if record[3] == "" {
				if len(recordIssue) == 0 {
					recordIssue = append(recordIssue, fmt.Sprintf("user %s failed to create due to:", record[0]))
				}
				recordIssue = append(recordIssue, "group is empty")
			}

			// check group exist
			if _, ok := flatGroup[record[3]]; !ok {
				// create group then
				group, err := groupProvider.CreateGroup(&coreUserV1.CoreGroup{
					Name:        record[3],
					Description: fmt.Sprintf("Group created by %s at %s", "upload api", time.Now().Format("2006-01-02 15:04:05")),
				}, nil)
				if err != nil {
					if len(recordIssue) == 0 {
						recordIssue = append(recordIssue, fmt.Sprintf("user %s failed to create due to:", record[0]))
					}
					recordIssue = append(recordIssue, fmt.Sprintf("failed to create group %s", record[3]))
					result = append(result, strings.Join(recordIssue, ","))
					coreApiLog.Logger.Error(fmt.Sprintf("Failed to create group %s", record[3]), "error", err)
					continue
				}
				coreApiLog.Logger.Debug(fmt.Sprintf("Created group %s", record[3]))
				// add back to cache
				flatGroup[record[3]] = *group
			} else {
				coreApiLog.Logger.Debug(fmt.Sprintf("Group %s already exist skip it", record[3]))
			}

			// check role
			if _, ok := flatRoles[record[2]]; !ok {
				// create role then
				role, err := roleProvider.CreateRole(&coreUserV1.CoreRole{
					Name:        record[2],
					Description: fmt.Sprintf("Role created by %s at %s", "upload api", time.Now().Format("2006-01-02 15:04:05")),
					Permission: map[string]uint64{
						"indexView":         1,
						"courseStudentView": 1,
						"deviceStudentView": 31,
						"rag":               30,
					},
				}, nil)
				if err != nil {
					if len(recordIssue) == 0 {
						recordIssue = append(recordIssue, fmt.Sprintf("user %s failed to create due to:", record[0]))
					}
					recordIssue = append(recordIssue, fmt.Sprintf("failed to create role %s", record[2]))
					result = append(result, strings.Join(recordIssue, ","))
					coreApiLog.Logger.Error(fmt.Sprintf("Failed to create role %s", record[2]), "error", err)
					continue
				}
				coreApiLog.Logger.Debug(fmt.Sprintf("Created role %s", record[2]))
				// add back to cache
				flatRoles[record[2]] = *role
			} else {
				coreApiLog.Logger.Debug(fmt.Sprintf("Role %s already exist skip it", record[2]))
			}

			// check user
			if _, ok := flatUser[record[0]]; !ok {
				// create user then
				roleId := flatRoles[record[2]].Id
				groupId := flatGroup[record[3]].Id
				_, err := userProvider.CreateUser(&coreUserV1.CoreUser{
					Name:     record[0],
					Password: record[1],
					Roles: []coreUserV1.CoreRole{
						{
							Id: roleId,
						},
					},
					Groups: []coreUserV1.CoreGroup{
						{
							Id: groupId,
						},
					},
					Description: record[4],
				}, nil)
				if err != nil {
					if len(recordIssue) == 0 {
						recordIssue = append(recordIssue, fmt.Sprintf("user %s failed to create due to:", record[0]))
					}
					recordIssue = append(recordIssue, fmt.Sprintf("failed to create user %s", record[0]))
					result = append(result, strings.Join(recordIssue, ","))
					coreApiLog.Logger.Error(fmt.Sprintf("Failed to create user %s", record[0]), "error", err)
					continue
				}
				coreApiLog.Logger.Debug(fmt.Sprintf("Created user %s", record[0]))
				flatUser[record[0]] = coreUserV1.CoreUser{}
			} else {
				coreApiLog.Logger.Debug(fmt.Sprintf("User %s already exist skip it", record[0]))
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if len(result) > 0 {
			w.WriteHeader(http.StatusInternalServerError)
			httpHelper.WriteResponseEntity(w, result)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[]"))
		}

	}
}
