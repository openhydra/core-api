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

type GroupProvider struct {
	Config *config.Config
	token  string
}

func (gp *GroupProvider) GetGroups(options map[string]struct{}) ([]core.CoreGroup, error) {
	// fetch groups from database or whatever
	groups, err := gp.GetRawKeystoneGroupList()
	if err != nil {
		coreApiLog.Logger.Error("Failed to get raw keystone group list", "error", err)
		return nil, err
	}

	var groupList []core.CoreGroup

	for _, group := range groups.Groups {

		// filter out that origin keystone group if LoadUnRelatedKeystoneGroups is not enabled
		if _, found := options[LoadUnRelatedKeystoneObjects]; !found && group.CoreGroup == nil {
			continue
		}

		groupList = append(groupList, *ConvertKeystoneGroupToCoreGroup(&group))
	}

	return groupList, nil
}

func (gp *GroupProvider) GetRawKeystoneGroupList() (*GroupContainer, error) {
	body, _, _, err := commentRequestAutoRenewToken("/v3/groups", http.MethodGet, gp.Config.AuthConfig.Keystone, gp.getToken, gp.setToken, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to list groups", "error", err)
		return nil, err
	}

	groupCollection := &GroupContainer{}
	err = json.Unmarshal(body, &groupCollection)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal groups", "error", err)
		return nil, err
	}
	return groupCollection, nil
}

func (gp *GroupProvider) getToken() (string, error) {
	if gp.token == "" {
		token, _, err := RequestToken(gp.Config.AuthConfig.Keystone.Username, gp.Config.AuthConfig.Keystone.Password, common.GetStringValueOrDefault(gp.Config.AuthConfig.Keystone.DomainId, KeystoneDefaultDomainId), gp.Config.AuthConfig.Keystone.Endpoint, gp.Config.AuthConfig.Keystone.TokenKeyInResponse, true)
		if err != nil {
			return "", err
		}
		gp.token = token
	}
	return gp.token, nil
}

func (gp *GroupProvider) setToken(token string) {
	gp.token = token
}

func (gp *GroupProvider) GetGroup(id string, options map[string]struct{}) (*core.CoreGroup, error) {
	body, _, code, err := commentRequestAutoRenewToken(fmt.Sprintf("/v3/groups/%s", id), http.MethodGet, gp.Config.AuthConfig.Keystone, gp.getToken, gp.setToken, nil)
	if code == http.StatusNotFound {
		return nil, customErr.NewNotFound(code, fmt.Sprintf("Group %s not found", id))
	}
	if err != nil {
		coreApiLog.Logger.Error("Failed to get group", "error", err)
		return nil, err
	}

	var groupContainer struct {
		Group *Group `json:"group"`
	}
	err = json.Unmarshal(body, &groupContainer)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal gropu", "error", err)
		return nil, err
	}

	// filter out that origin keystone group if LoadUnRelatedKeystoneGroups is not enabled
	if _, found := options[LoadUnRelatedKeystoneObjects]; !found && groupContainer.Group.CoreGroup == nil {
		return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("Group %s not found", id))
	}

	return ConvertKeystoneGroupToCoreGroup(groupContainer.Group), nil
}

func (gp *GroupProvider) UpdateGroup(group *core.CoreGroup, options map[string]struct{}) error {

	if group.Name == "admin" || group.Name == "service" {
		return fmt.Errorf("Group %s is a reserved group cannot be modified", group.Name)
	}

	groupPost := &Group{
		Name:        group.Name,
		ID:          group.Id,
		Description: group.Description,
		CoreGroup:   group,
	}

	postBody, err := json.Marshal(&struct {
		Group *Group `json:"group"`
	}{Group: groupPost})
	if err != nil {
		coreApiLog.Logger.Error("Failed to marshal group", "error", err)
		return err
	}

	_, _, _, err = commentRequestAutoRenewToken(fmt.Sprintf("/v3/groups/%s", group.Id), http.MethodPatch, gp.Config.AuthConfig.Keystone, gp.getToken, gp.setToken, postBody)
	if err != nil {
		coreApiLog.Logger.Error("Failed to update group", "error", err)
		return err
	}

	return nil
}

func (gp *GroupProvider) DeleteGroup(id string, options map[string]struct{}) error {
	group, err := gp.GetGroup(id, options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get group", "error", err)
		return err
	}

	if group.Name == "admin" || group.Name == "service" {
		return fmt.Errorf("build in group can not be deleted")
	}

	// now this is painful, we need to update users in the group to remove the group
	// note all following operation will be error ignored
	coreApiLog.Logger.Debug("Attempting to remove group from users", "group", id)
	// ge all users
	userProvider := &UserProvider{Config: gp.Config}
	userProvider.setToken(gp.token)
	users, err := userProvider.GetUsers(nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get users, failed to remove delete group for users", "error", err)
	} else {
		for _, user := range users {
			err = gp.RemoveUserFromGroup(user.Id, id)
			if err != nil {
				coreApiLog.Logger.Error("Failed to remove group from user", "error", err, "user", user.Id, "group", id)
			}
		}
	}

	_, _, _, err = commentRequestAutoRenewToken(fmt.Sprintf("/v3/groups/%s", id), http.MethodDelete, gp.Config.AuthConfig.Keystone, gp.getToken, gp.setToken, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to delete group", "error", err)
		return err
	}

	return nil
}

func (gp *GroupProvider) CreateGroup(group *core.CoreGroup, options map[string]struct{}) (*core.CoreGroup, error) {
	groupPost := &Group{
		Name:        group.Name,
		Description: group.Description,
		ID:          group.Id,
		DomainID:    gp.Config.AuthConfig.Keystone.DomainId,
		CoreGroup:   group,
	}

	groupBody, err := json.Marshal(&struct {
		Group *Group `json:"group"`
	}{Group: groupPost})
	if err != nil {
		coreApiLog.Logger.Error("Failed to marshal group", "error", err)
		return nil, err
	}

	body, _, status, err := commentRequestAutoRenewToken("/v3/groups", http.MethodPost, gp.Config.AuthConfig.Keystone, gp.getToken, gp.setToken, groupBody)
	if err != nil {
		coreApiLog.Logger.Error("Failed to create group", "error", err)
		return nil, err
	}

	if status != http.StatusCreated {
		coreApiLog.Logger.Error("Failed to create group", "status", status, "body", string(body))
		return nil, fmt.Errorf("failed to create group")
	}

	result := &struct {
		Group *Group `json:"group"`
	}{}
	err = json.Unmarshal(body, result)
	if err != nil {
		coreApiLog.Logger.Error("Failed to unmarshal group", "error", err)
		return nil, err
	}

	return ConvertKeystoneGroupToCoreGroup(result.Group), nil
}

func (gp *GroupProvider) SearchGroupByName(name string, options map[string]struct{}) (*core.CoreGroup, error) {
	groups, err := gp.GetGroups(options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get groups", "error", err)
		return nil, err
	}

	for _, group := range groups {
		if group.Name == name {
			return &group, nil
		}
	}

	return nil, customErr.NewNotFound(http.StatusNotFound, fmt.Sprintf("Group %s not found", name))
}

func (gp *GroupProvider) AddUserToGroup(userId, groupId string) error {
	// get user
	userProvider := &UserProvider{Config: gp.Config}
	userProvider.setToken(gp.token)
	user, err := userProvider.GetUser(userId, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get user", "error", err)
		return err
	}

	foundGroup, err := gp.GetGroup(groupId, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get group", "error", err)
		return err
	}

	// check see if user already in group
	for _, group := range user.Groups {
		if group.Id == groupId {
			coreApiLog.Logger.Warn("User already in group", "user", userId, "group", groupId)
			return nil
		}
	}

	// add group to user data
	user.Groups = append(user.Groups, core.CoreGroup{
		Id:   groupId,
		Name: foundGroup.Name,
	})

	// update user
	err = userProvider.UpdateUser(user, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to update user", "error", err)
		return err
	}
	return nil
}

func (gp *GroupProvider) RemoveUserFromGroup(userId, groupId string) error {
	// get user
	userProvider := &UserProvider{Config: gp.Config}
	userProvider.setToken(gp.token)
	user, err := userProvider.GetUser(userId, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get user", "error", err)
		return err
	}

	_, err = gp.GetGroup(groupId, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get group", "error", err)
		return err
	}

	// remove group from user data
	newGroups := []core.CoreGroup{}
	for _, group := range user.Groups {
		if group.Id != groupId {
			newGroups = append(newGroups, group)
		}
	}

	if len(newGroups) == len(user.Groups) {
		coreApiLog.Logger.Warn("User not in group", "user", userId, "group", groupId)
		return nil
	}

	user.Groups = newGroups

	coreApiLog.Logger.Debug("Removing group from user", "user", userId, "group", groupId)
	// update user
	err = userProvider.UpdateUser(&core.CoreUser{
		Id:     user.Id,
		Groups: newGroups,
	}, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to update user", "error", err)
		return err
	}
	return nil
}

func (gp *GroupProvider) GetGroupUsers(groupId string, options map[string]struct{}) ([]core.CoreUser, error) {
	_, err := gp.GetGroup(groupId, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get group", "error", err)
		return nil, err
	}

	// get all users
	userProvider := &UserProvider{Config: gp.Config}
	userProvider.setToken(gp.token)
	users, err := userProvider.GetUsers(nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get users", "error", err)
		return nil, err
	}

	var groupUsers []core.CoreUser
	for _, user := range users {
		if _, found := options[ReverseGetGroupUsers]; found {
			groupMatched := false
			if len(user.Groups) == 0 {
				groupUsers = append(groupUsers, user)
			} else {
				for _, group := range user.Groups {
					if group.Id == groupId {
						groupMatched = true
						break
					}
				}
				if !groupMatched {
					groupUsers = append(groupUsers, user)
				}
			}
		} else {
			for _, group := range user.Groups {
				if group.Id == groupId {
					groupUsers = append(groupUsers, user)
					break
				}
			}
		}
	}

	return groupUsers, nil
}

func (gp *GroupProvider) GetGroupSummary(options map[string]struct{}) (*core.CoreGroupSummary, error) {
	groups, err := gp.GetGroups(options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get groups", "error", err)
		return nil, err
	}

	flatGroupDetail := make(map[string]core.CoreGroupSummaryDetail)

	for _, group := range groups {
		if _, found := flatGroupDetail[group.Id]; !found {
			flatGroupDetail[group.Id] = core.CoreGroupSummaryDetail{
				Id:   group.Id,
				Name: group.Name,
			}
		}
	}

	// get all users
	userProvider := &UserProvider{Config: gp.Config}
	userProvider.setToken(gp.token)
	users, err := userProvider.GetUsers(options)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get users", "error", err)
		return nil, err
	}

	// loop all users group
	for _, user := range users {
		for _, group := range user.Groups {
			if _, found := flatGroupDetail[group.Id]; found {
				modifyGroup := flatGroupDetail[group.Id]
				modifyGroup.Count = modifyGroup.Count + 1
				flatGroupDetail[group.Id] = modifyGroup
			}
		}
	}

	result := &core.CoreGroupSummary{}
	for _, group := range flatGroupDetail {
		result.Counts = append(result.Counts, group)
	}
	return result, nil
}

func (gp *GroupProvider) AddUsersToGroup(groupId string, users []core.CoreUser) ([]core.CoreUser, []core.CoreUser, error) {
	var successes []core.CoreUser
	var failures []core.CoreUser
	var errSum error

	foundGroup, err := gp.GetGroup(groupId, nil)
	if err != nil {
		coreApiLog.Logger.Error("Failed to get group", "error", err)
		return nil, nil, err
	}

	userProvider := &UserProvider{Config: gp.Config}
	userProvider.setToken(gp.token)
	for index, user := range users {
		userFound, err := userProvider.GetUser(user.Id, nil)
		if err != nil {
			coreApiLog.Logger.Error("Failed to get user", "error", err)
			failures = append(failures, user)
			continue
		}
		for _, group := range user.Groups {
			if group.Id == groupId {
				coreApiLog.Logger.Warn("User already in group", "user", userFound.Id, "group", groupId)
				continue
			}
		}
		users[index].Groups = append(userFound.Groups, core.CoreGroup{
			Id:   groupId,
			Name: foundGroup.Name,
		})
		err = userProvider.UpdateUser(&core.CoreUser{
			Id:     users[index].Id,
			Groups: users[index].Groups,
		}, nil)
		if err != nil {
			coreApiLog.Logger.Error("Failed to update user", "error", err)
			failures = append(failures, user)
		} else {
			successes = append(successes, user)
		}
	}

	if len(failures) > 0 {
		errSum = fmt.Errorf("failed to add %d users to group", len(failures))
	}

	return successes, failures, errSum
}
