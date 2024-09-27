package train

import (
	"core-api/cmd/core-api-server/app/config"
	coreApiLog "core-api/pkg/logger"
	core "core-api/pkg/north/api/user/core/v1"
	"core-api/pkg/util/common"
	customErr "core-api/pkg/util/error"
	"encoding/json"
	"fmt"
	"net/http"
)

type RoleProvider struct {
	Config *config.Config
	token  string
}

// implement IRoleProvider

func (rp *RoleProvider) GetRoles(options map[string]struct{}) ([]core.CoreRole, error) {
	// fetch roles from database or whatever
	roles, err := rp.GetRawKeystoneRoleList()
	if err != nil {
		coreApiLog.Logger.Error("Failed to get raw keystone role list", "error", err)
		return nil, err
	}

	var roleList []core.CoreRole

	for _, role := range roles.Roles {

		// filter out that origin keystone role if LoadUnRelatedKeystoneRoles is not enabled
		if _, found := options[LoadUnRelatedKeystoneObjects]; !found && role.CoreRole == nil {
			continue
		}

		roleList = append(roleList, *ConvertKeystoneRoleToCoreRole(&role))
	}

	return roleList, nil
}

func (rp *RoleProvider) GetRole(id string, options map[string]struct{}) (*core.CoreRole, error) {
	body, _, code, err := commentRequestAutoRenewToken(fmt.Sprintf("/v3/roles/%s", id), http.MethodGet, rp.Config.AuthConfig.Keystone, rp.getToken, rp.setToken, nil)
	if code == http.StatusNotFound {
		return nil, customErr.NewNotFound(code, fmt.Sprintf("Role %s not found", id))
	}
	if err != nil {
		coreApiLog.Logger.Error("Failed to get role", "error", err)
		return nil, err
	}

	var roleContainer struct {
		Role *Role `json:"role"`
	}
	err = json.Unmarshal(body, &roleContainer)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal role", "error", err)
		return nil, err
	}

	// filter out that origin keystone role if LoadUnRelatedKeystoneRoles is not enabled
	if _, found := options[LoadUnRelatedKeystoneObjects]; !found && roleContainer.Role.CoreRole == nil {
		return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("Role %s not found", id))
	}

	return ConvertKeystoneRoleToCoreRole(roleContainer.Role), nil
}

func (rp *RoleProvider) UpdateRole(role *core.CoreRole, options map[string]struct{}) error {

	if role.Name == "admin" || role.Name == "service" {
		return fmt.Errorf("Role %s is a reserved role cannot be modified", role.Name)
	}

	rolePost := &Role{
		Name:        role.Name,
		Description: role.Description,
		CoreRole:    role,
	}

	postBody, err := json.Marshal(&struct {
		Role *Role `json:"role"`
	}{Role: rolePost})
	if err != nil {
		coreApiLog.Logger.Error("Failed to marshal role", "error", err)
		return err
	}

	_, _, _, err = commentRequestAutoRenewToken(fmt.Sprintf("/v3/roles/%s", role.Id), http.MethodPatch, rp.Config.AuthConfig.Keystone, rp.getToken, rp.setToken, postBody)
	if err != nil {
		coreApiLog.Logger.Error("Failed to create user", "error", err)
		return err
	}

	return nil
}

func (rp *RoleProvider) DeleteRole(id string, options map[string]struct{}) error {
	role, err := rp.GetRole(id, options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get role", "error", err)
		return err
	}

	if role.Name == "admin" || role.Name == "service" || role.Name == "aes-admin" {
		return fmt.Errorf("build in role can not be deleted")
	}

	// get all users
	userProvider := &UserProvider{Config: rp.Config}
	userProvider.setToken(rp.token)
	users, err := userProvider.GetUsers(nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get users", "error", err)
	} else {
		for index, user := range users {
			for roleIndex, role := range user.Roles {
				if role.Id == id {
					if len(users[index].Roles) == 0 {
						continue
					}
					if len(users[index].Roles) == 1 {
						users[index].Roles = []core.CoreRole{}
					} else {
						users[index].Roles = append(users[index].Roles[:roleIndex], users[index].Roles[roleIndex+1:]...)
					}
					err = userProvider.UpdateUser(&users[index], nil)
					if err != nil {
						coreApiLog.Logger.Error("Failed to update user", "error", err)
					}
					break
				}
			}
		}
	}

	_, _, _, err = commentRequestAutoRenewToken(fmt.Sprintf("/v3/roles/%s", id), http.MethodDelete, rp.Config.AuthConfig.Keystone, rp.getToken, rp.setToken, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to delete role", "error", err)
		return err
	}
	return nil
}

// create role
func (rp *RoleProvider) CreateRole(role *core.CoreRole, options map[string]struct{}) (*core.CoreRole, error) {
	rolePost := &Role{
		Name:        role.Name,
		Description: role.Description,
		CoreRole:    role,
	}

	postBody, err := json.Marshal(&struct {
		Role *Role `json:"role"`
	}{Role: rolePost})
	if err != nil {
		coreApiLog.Logger.Error("Failed to marshal role", "error", err)
		return nil, err
	}

	result, _, _, err := commentRequestAutoRenewToken("/v3/roles", http.MethodPost, rp.Config.AuthConfig.Keystone, rp.getToken, rp.setToken, postBody)
	if err != nil {
		coreApiLog.Logger.Error("Failed to create role", "error", err)
		return nil, err
	}

	var returnedRole = &struct {
		Role *Role `json:"role"`
	}{}

	err = json.Unmarshal(result, returnedRole)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal role", "error", err)
		return nil, err
	}

	role.Id = returnedRole.Role.ID
	return role, nil
}

func (rp *RoleProvider) GetRawKeystoneRoleList() (*RoleContainer, error) {
	body, _, _, err := commentRequestAutoRenewToken("/v3/roles", http.MethodGet, rp.Config.AuthConfig.Keystone, rp.getToken, rp.setToken, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to list roles", "error", err)
		return nil, err
	}

	roleCollection := &RoleContainer{}
	err = json.Unmarshal(body, &roleCollection)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal roles", "error", err)
		return nil, err
	}
	return roleCollection, nil
}

func (rp *RoleProvider) getToken() (string, error) {
	if rp.token == "" {
		token, _, err := RequestToken(rp.Config.AuthConfig.Keystone.Username, rp.Config.AuthConfig.Keystone.Password, common.GetStringValueOrDefault(rp.Config.AuthConfig.Keystone.DomainId, KeystoneDefaultDomainId), rp.Config.AuthConfig.Keystone.Endpoint, rp.Config.AuthConfig.Keystone.TokenKeyInResponse, true)
		if err != nil {
			return "", err
		}
		rp.token = token
	}
	return rp.token, nil
}

func (rp *RoleProvider) setToken(token string) {
	rp.token = token
}

func (rp *RoleProvider) SearchRoleByName(name string, options map[string]struct{}) (*core.CoreRole, error) {
	roles, err := rp.GetRoles(options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get roles", "error", err)
		return nil, err
	}

	for _, role := range roles {
		if role.Name == name {
			return &role, nil
		}
	}

	return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("Role %s not found", name))
}
