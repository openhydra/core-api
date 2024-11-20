package auth

import (
	"core-api/cmd/core-api-server/app/config"
	keystone "core-api/pkg/core/auth/provider/keystone/train"
	core "core-api/pkg/north/api/user/core/v1"
	"fmt"
)

// use interface to transparently switch between different implementations

type IUserProvider interface {
	SearchUserByName(name string, options map[string]struct{}) (*core.CoreUser, error)
	GetUsers(options map[string]struct{}) ([]core.CoreUser, error)
	GetUser(id string, options map[string]struct{}) (*core.CoreUser, error)
	UpdateUser(user *core.CoreUser, options map[string]struct{}) error
	DeleteUser(id string, options map[string]struct{}) error
	CreateUser(user *core.CoreUser, options map[string]struct{}) (*core.CoreUser, error)
	LoginUser(username, password string) (*core.CoreUser, error)
}

type IRoleProvider interface {
	GetRoles(options map[string]struct{}) ([]core.CoreRole, error)
	GetRole(id string, options map[string]struct{}) (*core.CoreRole, error)
	UpdateRole(role *core.CoreRole, options map[string]struct{}) error
	DeleteRole(id string, options map[string]struct{}) error
	CreateRole(role *core.CoreRole, options map[string]struct{}) (*core.CoreRole, error)
	SearchRoleByName(name string, options map[string]struct{}) (*core.CoreRole, error)
}

type IGroupProvider interface {
	GetGroups(options map[string]struct{}) ([]core.CoreGroup, error)
	GetGroup(id string, options map[string]struct{}) (*core.CoreGroup, error)
	UpdateGroup(group *core.CoreGroup, options map[string]struct{}) error
	DeleteGroup(id string, options map[string]struct{}) error
	CreateGroup(group *core.CoreGroup, options map[string]struct{}) (*core.CoreGroup, error)
	SearchGroupByName(name string, options map[string]struct{}) (*core.CoreGroup, error)
	AddUserToGroup(userId, groupId string) error
	RemoveUserFromGroup(userId, groupId string) error
	GetGroupUsers(groupId string, options map[string]struct{}) ([]core.CoreUser, error)
	GetGroupSummary(options map[string]struct{}) (*core.CoreGroupSummary, error)
	AddUsersToGroup(groupId string, users []core.CoreUser) ([]core.CoreUser, []core.CoreUser, error)
}

type AuthProviderType string

const (
	KeystoneAuthProvider AuthProviderType = "keystone"
)

func CreateUserProvider(config *config.Config, authProviderType AuthProviderType) (IUserProvider, error) {

	switch authProviderType {
	case KeystoneAuthProvider:
		return &keystone.UserProvider{
			Config: config,
		}, nil
	}
	return nil, fmt.Errorf("%s is not a valid user provider type", authProviderType)
}

func CreateRoleProvider(config *config.Config, authProviderType AuthProviderType) (IRoleProvider, error) {
	switch authProviderType {
	case KeystoneAuthProvider:
		return &keystone.RoleProvider{
			Config: config,
		}, nil
	}
	return nil, fmt.Errorf("%s is not a valid role provider type", authProviderType)
}

func CreateGroupProvider(config *config.Config, authProviderType AuthProviderType) (IGroupProvider, error) {
	switch authProviderType {
	case KeystoneAuthProvider:
		return &keystone.GroupProvider{
			Config: config,
		}, nil
	}
	return nil, fmt.Errorf("%s is not a valid group provider type", authProviderType)
}
