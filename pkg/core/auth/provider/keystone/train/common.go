package train

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	coreApiLog "core-api/pkg/logger"
	core "core-api/pkg/north/api/user/core/v1"
	"core-api/pkg/util/common"
	customError "core-api/pkg/util/error"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	KeystoneDefaultDomainId = "default"
)

func ConvertKeystoneUserToCoreUser(user *User, hidePassword bool, permission map[string]uint64) *core.CoreUser {

	var result *core.CoreUser

	if user.CoreUser != nil {
		result = user.CoreUser
		result.Id = user.ID
	} else {
		result = &core.CoreUser{
			Id:       user.ID,
			Name:     user.Name,
			Email:    user.Email,
			Password: user.Password,
		}
	}

	if hidePassword {
		result.Password = ""
	}

	if user.Name == "admin" || user.Name == "service" {
		result.UnEditable = true
	}

	if permission == nil {
		permission = privileges.ModulesNoPermission()
	}

	result.Permission = permission

	return result
}

func ConvertKeystoneRoleToCoreRole(role *Role) *core.CoreRole {
	if role.CoreRole != nil {
		role.CoreRole.Id = role.ID
		return role.CoreRole
	}

	return &core.CoreRole{
		Id:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permission:  map[string]uint64{},
	}
}

func ConvertKeystoneGroupToCoreGroup(group *Group) *core.CoreGroup {
	if group.CoreGroup != nil {
		group.CoreGroup.Id = group.ID
		return group.CoreGroup
	}

	return &core.CoreGroup{
		Id:          group.ID,
		Name:        group.Name,
		Description: group.Description,
	}
}

func RequestToken(name, password, DomainId, keystoneEndpoint, tokenInResponse string, enableScoped bool) (string, *TokenResponse, error) {
	authReq := &AuthRequest{
		Auth: AuthDetails{
			Identity: IdentityDetails{
				Methods: []string{"password"},
				Password: PasswordDetails{
					User: User{
						Name:     name,
						Password: password,
						Domain: &Domain{
							Id: DomainId,
						},
					},
				},
			},
		},
	}

	if enableScoped {
		authReq.Auth.Scope = &ScopeDetails{
			Domain: DomainDetails{
				Id: DomainId,
			},
		}
	}

	postBody, err := json.Marshal(authReq)
	if err != nil {
		coreApiLog.Logger.Error("Failed to marshal auth request", "error", err)
		return "", nil, err
	}

	coreApiLog.Logger.Debug("Requesting token from keystone", "body", string(postBody))

	resp, header, respCode, err := common.CommonRequest(common.BuildPath(keystoneEndpoint, "/v3/auth/tokens"), http.MethodPost, "", postBody, nil, true, false, 3*time.Second)
	if err != nil {
		coreApiLog.Logger.Error("Failed to request token", "error", err)
		return "", nil, customError.NewUnauthorized(http.StatusUnauthorized, "Failed to request token")
	}

	if respCode != http.StatusCreated {
		coreApiLog.Logger.Error("Failed to request token", "response_code", respCode)
		return "", nil, customError.NewUnauthorized(respCode, "Failed to request token")
	}

	key := common.GetStringValueOrDefault(tokenInResponse, "X-Subject-Token")

	token := header.Get(key)
	if token == "" {
		coreApiLog.Logger.Error("No token in response header")
		return "", nil, customError.NewUnauthorized(http.StatusUnauthorized, "No token in response header")
	}

	tokenResp := &TokenResponse{}
	err = json.Unmarshal(resp, tokenResp)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal token response", "error", err)
		return "", nil, err
	}

	return token, tokenResp, nil
}

func commentRequestAutoRenewToken(path, method string, keystoneConfig *config.KeystoneConfig, getToken func() (string, error), setToken func(token string), body json.RawMessage) ([]byte, http.Header, int, error) {
	tokenHeaderKey := common.GetStringValueOrDefault(keystoneConfig.TokenKeyInRequest, "X-Auth-Token")
	token, err := getToken()
	if err != nil {
		coreApiLog.Logger.Error("Failed to get token", "error", err)
		return nil, nil, -1, err
	}
	reqURL := common.BuildPath(keystoneConfig.Endpoint, path)
	result, header, code, err := common.CommonRequest(reqURL, method, "", body, map[string][]string{tokenHeaderKey: {token}}, true, false, 3*time.Second)

	if code == http.StatusUnauthorized {
		coreApiLog.Logger.Warn("Token may expired, attempt to renew the token and retry for one shot")
		// token may expired, request a new one
		newToken, _, err := RequestToken(keystoneConfig.Username, keystoneConfig.Password, keystoneConfig.DomainId, keystoneConfig.Endpoint, keystoneConfig.TokenKeyInResponse, true)
		if err != nil {
			coreApiLog.Logger.Error("Failed to renew token", "error", err)
			return nil, nil, -1, err
		}
		setToken(newToken)
		return common.CommonRequest(reqURL, method, "", body, map[string][]string{tokenHeaderKey: {newToken}}, true, false, 3*time.Second)
	}

	if err != nil {
		coreApiLog.Logger.Error(fmt.Sprintf("Failed to request %s", reqURL), "error", err)
		return nil, nil, -1, err
	}

	if code != http.StatusOK && code != http.StatusCreated && code != http.StatusNoContent {
		coreApiLog.Logger.Error(fmt.Sprintf("Failed to request %s, response code: %d", reqURL, code))
		return nil, nil, code, fmt.Errorf("failed to request %s, response code: %d", reqURL, code)
	}

	// if everything is fine, return the result
	return result, header, code, nil
}

func getDomainOrDefault(domainId string) string {
	return common.GetStringValueOrDefault(domainId, KeystoneDefaultDomainId)
}

func sumPermission(userRelatedRoles []core.CoreRole, allRoles map[string]core.CoreRole) map[string]uint64 {
	initPermission := privileges.ModulesNoPermission()
	for _, role := range userRelatedRoles {
		if r, found := allRoles[role.Id]; found {
			for module, permission := range r.Permission {
				if _, found := r.Permission[module]; found {
					initPermission[module] |= permission
				}
			}
		} else {
			coreApiLog.Logger.Warn("Role not found", "role", role.Id)
		}
	}
	return initPermission
}

func getRoleMap(serverConfig *config.Config, token string, options map[string]struct{}) (map[string]core.CoreRole, error) {

	mapRoles := make(map[string]core.CoreRole)
	// load roles
	roleProvider := &RoleProvider{Config: serverConfig}
	roleProvider.setToken(token)
	roles, err := roleProvider.GetRoles(options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get roles", "error", err)
		return nil, err
	}

	// use space complexity to reduce time complexity
	for _, role := range roles {
		mapRoles[role.Id] = role
	}
	return mapRoles, nil
}

func getGroupMap(serverConfig *config.Config, token string, options map[string]struct{}) (map[string]core.CoreGroup, error) {

	mapGroups := make(map[string]core.CoreGroup)
	// load groups
	groupProvider := &GroupProvider{Config: serverConfig}
	groupProvider.setToken(token)
	groups, err := groupProvider.GetGroups(options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get groups", "error", err)
		return nil, err
	}

	// use space complexity to reduce time complexity
	for _, group := range groups {
		mapGroups[group.Id] = group
	}
	return mapGroups, nil
}
