package train

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	coreApiLog "core-api/pkg/logger"
	core "core-api/pkg/north/api/user/core/v1"
	customErr "core-api/pkg/util/error"
	"encoding/json"
	"fmt"
	"net/http"
)

type UserProvider struct {
	Config *config.Config
	token  string
}

// implement IUserProvider

func (up *UserProvider) GetUsers(options map[string]struct{}) ([]core.CoreUser, error) {
	userCollection, err := up.GetRawKeystoneUserList()
	if err != nil {
		coreApiLog.Logger.Error("Failed to get raw keystone user list", "error", err)
		return []core.CoreUser{}, err
	}

	var mapRoles map[string]core.CoreRole
	if _, found := options[LoadPermission]; found {
		mapRoles, err = getRoleMap(up.Config, up.token, options)
		if err != nil {
			coreApiLog.Logger.Error("Failed to get role map", "error", err)
			return []core.CoreUser{}, err
		}
	}

	var userList []core.CoreUser

	for _, user := range userCollection.Users {
		if !user.Enabled {
			// skip disabled user
			continue
		}

		// filter out that origin keystone user if LoadUnRelatedKeystoneObjects is not enabled
		if _, found := options[LoadUnRelatedKeystoneObjects]; !found && user.CoreUser == nil {
			continue
		}

		initPermission := privileges.ModulesNoPermission()
		if len(mapRoles) > 0 && user.CoreUser != nil {
			initPermission = sumPermission(user.CoreUser.Roles, mapRoles)
		}

		_, found := options[LoadPasswd]
		userList = append(userList, *ConvertKeystoneUserToCoreUser(&user, !found, initPermission))
	}

	return userList, nil
}

func (up *UserProvider) SearchUserByName(name string, options map[string]struct{}) (*core.CoreUser, error) {
	users, err := up.GetUsers(options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get users", "error", err)
		return nil, err
	}

	for _, user := range users {
		if user.Name == name {
			return &user, nil
		}
	}

	return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("User %s not found", name))
}

func (up *UserProvider) GetUser(id string, options map[string]struct{}) (*core.CoreUser, error) {
	body, _, code, err := commentRequestAutoRenewToken(fmt.Sprintf("/v3/users/%s", id), http.MethodGet, up.Config.AuthConfig.Keystone, up.getToken, up.setToken, nil)
	if code == http.StatusNotFound {
		return nil, customErr.NewNotFound(code, fmt.Sprintf("User %s not found", id))
	}
	if err != nil {
		coreApiLog.Logger.Error("Failed to get user", "error", err)
		return nil, err
	}

	var userContainer struct {
		User *User `json:"user"`
	}
	err = json.Unmarshal(body, &userContainer)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal user", "error", err)
		return nil, err
	}

	// filter out that origin keystone user if LoadUnRelatedKeystoneObjects is not enabled
	if _, found := options[LoadUnRelatedKeystoneObjects]; !found && userContainer.User.CoreUser == nil {
		return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("User %s not found", id))
	}

	var mapRoles map[string]core.CoreRole
	if _, found := options[LoadPermission]; found {
		mapRoles, err = getRoleMap(up.Config, up.token, options)
		if err != nil {
			coreApiLog.Logger.Error("Failed to get role map", "error", err)
			return nil, err
		}
	}

	initPermission := privileges.ModulesNoPermission()
	if len(mapRoles) > 0 && userContainer.User.CoreUser != nil {
		initPermission = sumPermission(userContainer.User.CoreUser.Roles, mapRoles)
	}

	_, found := options[LoadPasswd]
	return ConvertKeystoneUserToCoreUser(userContainer.User, !found, initPermission), nil
}

func (up *UserProvider) UpdateUser(user *core.CoreUser, options map[string]struct{}) error {

	// get old user
	if options == nil {
		options = map[string]struct{}{LoadPasswd: {}}
	}

	oldUser, err := up.GetUser(user.Id, options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get user", "error", err)
		return err
	}

	changed := false
	userPost := &User{}
	if user.Password != "" {
		// update password
		changed = true
		oldUser.Password = user.Password
		userPost.Password = user.Password
	}

	if user.Email != "" {
		// update email
		changed = true
		oldUser.Email = user.Email
		userPost.Email = user.Email
	}

	if user.Roles != nil {
		changed = true
		oldUser.Roles = user.Roles
	}

	if user.Groups != nil {
		changed = true
		oldUser.Groups = user.Groups
	}

	if user.Description != "" {
		changed = true
		oldUser.Description = user.Description
	}

	if !changed {
		coreApiLog.Logger.Debug("No change found skip update", "user", user.Id)
		return nil
	}

	// update user in database or wherever
	userPost.CoreUser = oldUser

	postBody, err := json.Marshal(&struct {
		User *User `json:"user"`
	}{User: userPost})
	if err != nil {
		coreApiLog.Logger.Error("Failed to marshal user", "error", err)
		return err
	}

	_, _, _, err = commentRequestAutoRenewToken(fmt.Sprintf("/v3/users/%s", user.Id), http.MethodPatch, up.Config.AuthConfig.Keystone, up.getToken, up.setToken, postBody)
	if err != nil {
		coreApiLog.Logger.Error("Failed to create user", "error", err)
		return err
	}

	return nil
}

func (up *UserProvider) DeleteUser(id string, options map[string]struct{}) error {

	user, err := up.GetUser(id, options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get user", "error", err)
		return err
	}

	if user.Name == "admin" || user.Name == "service" {
		return fmt.Errorf("build in user can not be deleted")
	}

	_, _, _, err = commentRequestAutoRenewToken(fmt.Sprintf("/v3/users/%s", id), http.MethodDelete, up.Config.AuthConfig.Keystone, up.getToken, up.setToken, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to delete user", "error", err)
		return err
	}
	return nil
}

// create user
func (up *UserProvider) CreateUser(user *core.CoreUser, options map[string]struct{}) (*core.CoreUser, error) {

	// get role map
	mapRoles, err := getRoleMap(up.Config, up.token, options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get role map", "error", err)
		return nil, err
	}

	// check if all roles are valid
	for _, role := range user.Roles {
		if _, found := mapRoles[role.Id]; !found {
			coreApiLog.Logger.Error("Role not found", "role", role.Id, "user", user.Name)
			return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("Role %s not found", role.Id))
		}
	}

	// get group map
	mapGroups, err := getGroupMap(up.Config, up.token, options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get group map", "error", err)
		return nil, err
	}

	// check if all groups are valid
	for _, group := range user.Groups {
		if _, found := mapGroups[group.Id]; !found {
			coreApiLog.Logger.Error("Group not found", "group", group.Id, "user", user.Name)
			return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("Group %s not found", group.Id))
		}
	}

	userPost := &User{
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
		Enabled:  true,
		CoreUser: user,
		Options: &Options{
			IgnorePasswordExpiry:             true,
			IgnoreChangePasswordUponFirstUse: true,
			IgnoreLockoutFailureAttempts:     true,
		},
	}

	postBody, err := json.Marshal(&struct {
		User *User `json:"user"`
	}{User: userPost})
	if err != nil {
		coreApiLog.Logger.Error("Failed to marshal user", "error", err)
		return nil, err
	}

	retBodyBytes, _, retCode, err := commentRequestAutoRenewToken("/v3/users", http.MethodPost, up.Config.AuthConfig.Keystone, up.getToken, up.setToken, postBody)
	if err != nil {
		coreApiLog.Logger.Error("Failed to create user", "error", err)
		return nil, err
	}

	if retCode != http.StatusCreated && retCode != http.StatusOK {
		coreApiLog.Logger.Error("Failed to create user", "code", retCode, "body", string(retBodyBytes))
		return nil, fmt.Errorf("failed to create user")
	}

	var userContainer struct {
		User *User `json:"user"`
	}

	err = json.Unmarshal(retBodyBytes, &userContainer)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal user", "error", err)
		return nil, err
	}

	fmt.Println(userContainer)

	user.Id = userContainer.User.ID

	return user, nil
}

func (up *UserProvider) getToken() (string, error) {
	if up.token == "" {
		token, _, err := RequestToken(up.Config.AuthConfig.Keystone.Username, up.Config.AuthConfig.Keystone.Password, getDomainOrDefault(up.Config.AuthConfig.Keystone.DomainId), up.Config.AuthConfig.Keystone.Endpoint, up.Config.AuthConfig.Keystone.TokenKeyInResponse, true)
		if err != nil {
			return "", err
		}
		up.token = token
	}
	return up.token, nil
}

func (up *UserProvider) setToken(token string) {
	up.token = token
}

func (up *UserProvider) GetRawKeystoneUserList() (UserContainer, error) {
	body, _, _, err := commentRequestAutoRenewToken("/v3/users", http.MethodGet, up.Config.AuthConfig.Keystone, up.getToken, up.setToken, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to list users", "error", err)
		return UserContainer{}, err
	}
	//fmt.Println(string(body))

	var userCollection UserContainer
	err = json.Unmarshal(body, &userCollection)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal users", "error", err)
		return UserContainer{}, err
	}
	return userCollection, nil
}

// due to keystone do not have api to query user with name
// we have to list all users and find the user id by name
func (up *UserProvider) GetUserIdFromName(name string) (string, error) {
	userCollection, err := up.GetRawKeystoneUserList()
	if err != nil {
		coreApiLog.Logger.Error("Failed to get raw keystone user list", "error", err)
		return "", err
	}

	for _, user := range userCollection.Users {
		if user.Name == name {
			return user.ID, nil
		}
	}

	return "", fmt.Errorf(fmt.Sprintf("user %s not found", name))
}

// Login a user
func (up *UserProvider) LoginUser(name, password string) (*core.CoreUser, error) {

	_, _, err := RequestToken(name, password, getDomainOrDefault(up.Config.AuthConfig.Keystone.DomainId), up.Config.AuthConfig.Keystone.Endpoint, up.Config.AuthConfig.Keystone.TokenKeyInResponse, false)
	if err != nil {
		coreApiLog.Logger.Error("Failed to login user", "error", err)
		return nil, err
	}

	// now user detail by name
	return up.SearchUserByName(name, map[string]struct{}{LoadPermission: {}})
}
