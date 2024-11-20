package train

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	core "core-api/pkg/north/api/user/core/v1"
	"core-api/pkg/util/common"
	customErr "core-api/pkg/util/error"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	coreApiLog "core-api/pkg/logger"

	"github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testUsersInGroup = map[string]struct{ Users []User }{
	"test1id": {Users: []User{
		{ID: "test1id", Name: "test1", Password: "test1", Email: "", Enabled: true},
	}},
	"test2id": {Users: []User{
		{ID: "test2id", Name: "test2", Password: "test2", Email: "", Enabled: true},
		{ID: "test3id", Name: "test3", Password: "test3", Email: "", Enabled: true},
	}},
}

var testGroupUser = core.CoreUser{
	Id:     "testTestGroupUser",
	Name:   "testTestGroupUser",
	Groups: []core.CoreGroup{{Id: "test1id"}, {Id: "test2id"}},
}

var testGroupList = GroupContainer{[]Group{
	{ID: "test1id", Name: "test1", Description: "test1", Links: &Links{}, CoreGroup: &core.CoreGroup{
		Id:          "test1id",
		Name:        "test1",
		Description: "test1",
		TagColor:    "test1Red",
	}},
	{ID: "test2id", Name: "test2", Description: "test2", Links: &Links{}, CoreGroup: &core.CoreGroup{
		Id:          "test2id",
		Name:        "test2",
		Description: "test2",
		TagColor:    "test2Green",
	}},
	{ID: "adminId", Name: "admin", Description: "admin", Links: &Links{}, CoreGroup: &core.CoreGroup{
		Id:          "adminId",
		Name:        "admin",
		Description: "admin",
	}},
	{ID: "serviceId", Name: "service", Description: "service", Links: &Links{}},
},
}
var groupCreated *Group

var testUserList = UserContainer{[]User{
	{ID: "test1id", Name: "test1", Password: "test1", Email: "test1@test.com", Enabled: true, CoreUser: &core.CoreUser{
		Id:          "test1id",
		Name:        "test1",
		Email:       "test1@test.com",
		Description: "test1",
		Password:    "test1",
		Roles:       []core.CoreRole{{Id: "test1id", Name: "test1", Permission: map[string]uint64{"test": 1}}},
		Groups:      []core.CoreGroup{{Id: "test1id"}},
	},
	},
	{ID: "test2id", Name: "test2", Password: "test2", Enabled: true, CoreUser: &core.CoreUser{
		Name:        "test2",
		Email:       "test2@test.com",
		Description: "test2",
		Password:    "test2",
		Roles:       []core.CoreRole{{Id: "test2id", Name: "test2", Permission: map[string]uint64{"dataset": 2}}},
		Groups:      []core.CoreGroup{{Id: "test1id"}, {Id: "test2id"}},
	}},
	{ID: "test3id", Name: "test3", Password: "test3", Email: "test3@test.com", Enabled: true, CoreUser: &core.CoreUser{
		Name:        "test3",
		Email:       "test3@test.com",
		Description: "test3",
		Password:    "test3",
		Roles:       []core.CoreRole{{Id: "test1id", Name: "test1"}, {Id: "test2id", Name: "test2"}}},
	},
	{ID: "adminid", Name: "admin", Password: "admin", Email: "admin@test.com", Enabled: true},
	{ID: "test4id", Name: "test4", Password: "test4", Email: "test40@test.com", Enabled: true, CoreUser: &core.CoreUser{
		Name:        "test4",
		Email:       "test4@test.com",
		Description: "test4",
		Password:    "test4",
		Roles:       []core.CoreRole{{Id: "test1id", Name: "test1"}, {Id: "test2id", Name: "test2"}}},
	},
	{ID: "login", Name: "login", Password: "login", Email: "login@test.com", Enabled: true, CoreUser: &core.CoreUser{
		Name:        "login",
		Email:       "login@test.com",
		Description: "login",
		Password:    "login",
	},
	},
},
}

var testRoleList = RoleContainer{[]Role{
	{ID: "test1id", Name: "test1", Description: "test1", Links: &Links{}, CoreRole: &core.CoreRole{
		Id:          "test1id",
		Name:        "test1",
		Description: "test1",
		Permission: map[string]uint64{
			"course": 1,
		},
	}},
	{ID: "test2id", Name: "test2", Description: "test2", Links: &Links{}, CoreRole: &core.CoreRole{
		Id:          "test2id",
		Name:        "test1",
		Description: "test1",
		Permission: map[string]uint64{
			"dataset": 2,
		},
	}},
	{ID: "adminId", Name: "admin", Description: "admin", Links: &Links{}},
	{ID: "serviceId", Name: "service", Description: "service", Links: &Links{}},
},
}

var roleCreated *Role
var userCreated *User

var testRouter = func(ws *restful.WebService) {
	ws.Route(ws.POST("/v3/auth/tokens").To(func(request *restful.Request, response *restful.Response) {
		body, err := io.ReadAll(request.Request.Body)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		var auth AuthRequest
		err = json.Unmarshal(body, &auth)
		if err != nil {
			response.WriteErrorString(http.StatusBadRequest, "failed to parse request body")
			return
		}

		if auth.Auth.Identity.Password.User.Name == "login" && auth.Auth.Identity.Password.User.Password == "login" {
			response.AddHeader("X-Subject-Token", "test-token")
			response.WriteHeader(http.StatusCreated)
			response.WriteAsJson(&TokenResponse{
				Token: Token{
					Methods:   []string{"password"},
					User:      User{Name: "login", ID: "login"},
					AuditIds:  []string{"login"},
					ExpiresAt: "2021-08-10T14:00:00Z",
					IssuedAt:  "2021-08-10T13:00:00Z",
					Domain:    Domain{Name: "default"},
					Roles:     []Role{{ID: "test", Name: "test"}},
					Catalog:   []Catalog{{Endpoints: []Endpoint{{ID: "test", Interface: "public", RegionID: "test", URL: "http://test", Region: "test"}}, ID: "test", Type: "test", Name: "test"}},
				},
			})
			return
		}

		if auth.Auth.Identity.Password.User.Name == "failedToLogin" && auth.Auth.Identity.Password.User.Password == "failedToLogin" {
			response.WriteHeader(http.StatusUnauthorized)
		}

		if auth.Auth.Identity.Password.User.Name == "noToken" && auth.Auth.Identity.Password.User.Password == "noToken" {
			response.WriteHeader(http.StatusCreated)
			response.WriteAsJson(&TokenResponse{
				Token: Token{
					Methods:   []string{"password"},
					User:      User{Name: "login", ID: "login"},
					AuditIds:  []string{"login"},
					ExpiresAt: "2021-08-10T14:00:00Z",
					IssuedAt:  "2021-08-10T13:00:00Z",
					Domain:    Domain{Name: "default"},
					Roles:     []Role{{ID: "test", Name: "test"}},
					Catalog:   []Catalog{{Endpoints: []Endpoint{{ID: "test", Interface: "public", RegionID: "test", URL: "http://test", Region: "test"}}, ID: "test", Type: "test", Name: "test"}},
				},
			})
			return
		}

		for _, user := range testUserList.Users {
			if auth.Auth.Identity.Password.User.Name == user.Name && auth.Auth.Identity.Password.User.Password == user.Password {
				response.AddHeader("X-Subject-Token", "test-token")
				response.WriteHeader(http.StatusCreated)
				response.WriteAsJson(&TokenResponse{
					Token: Token{
						Methods:   []string{"password"},
						User:      User{Name: user.Name, ID: user.ID},
						AuditIds:  []string{"test"},
						ExpiresAt: "2021-08-10T14:00:00Z",
						IssuedAt:  "2021-08-10T13:00:00Z",
						Domain:    Domain{Name: "default"},
						Roles:     []Role{{ID: "test", Name: "test"}},
						Catalog:   []Catalog{{Endpoints: []Endpoint{{ID: "test", Interface: "public", RegionID: "test", URL: "http://test", Region: "test"}}, ID: "test", Type: "test", Name: "test"}},
					},
				})
				return
			}
		}
		fmt.Printf("no match %s", auth.Auth.Identity.Password.User.Name)
		fmt.Println(testUserList)
		response.WriteHeader(http.StatusUnauthorized)
	}))
	ws.Route(ws.GET("/v3/users").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		tempUserList := testUserList
		tempUserList.Users = append(tempUserList.Users, User{
			ID:       testGroupUser.Id,
			Name:     testGroupUser.Name,
			CoreUser: &testGroupUser,
		})

		usersJson, err := json.Marshal(tempUserList)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		response.Write(usersJson)
	}))
	ws.Route(ws.GET("/v3/roles").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		rolesJson, err := json.Marshal(testRoleList)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		response.Write(rolesJson)
	}))
	ws.Route(ws.GET("/v3/groups").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		groupsJson, err := json.Marshal(testGroupList)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		response.Write(groupsJson)
	}))
	ws.Route(ws.GET("/v3/groups/{id}/users").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		groupID := request.PathParameter("id")
		if groupID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		group, found := testUsersInGroup[groupID]
		if !found {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		groupJson, err := json.Marshal(group)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		response.Write(groupJson)
	}))
	ws.Route(ws.GET("/v3/users/{user_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID := request.PathParameter("user_id")
		if userID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		if userID == "test1id" {
			userJson, err := json.Marshal(&struct {
				User *User `json:"user"`
			}{User: &testUserList.Users[0]})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(userJson)
			return
		}
		if userID == "test2id" {
			userJson, err := json.Marshal(&struct {
				User *User `json:"user"`
			}{User: &testUserList.Users[1]})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(userJson)
			return
		}
		if userID == "test3id" {
			userJson, err := json.Marshal(&struct {
				User *User `json:"user"`
			}{User: &testUserList.Users[2]})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(userJson)
			return
		}
		if userID == "test4id" {
			userJson, err := json.Marshal(&struct {
				User *User `json:"user"`
			}{User: &testUserList.Users[4]})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(userJson)
			return
		}
		if userID == "adminid" {
			userJson, err := json.Marshal(&struct {
				User *User `json:"user"`
			}{User: &User{Name: "admin", ID: "admin", Password: "admin"}})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(userJson)
			return
		}
		if userID == "testTestGroupUser" {
			userJson, err := json.Marshal(&struct {
				User *User `json:"user"`
			}{User: &User{
				ID:       testGroupUser.Id,
				Name:     testGroupUser.Name,
				CoreUser: &testGroupUser,
			}})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(userJson)
			return
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	ws.Route(ws.GET("/v3/roles/{role_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		roleId := request.PathParameter("role_id")
		if roleId == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		if roleId == "test1id" {
			roleJson, err := json.Marshal(&struct {
				Role *Role `json:"role"`
			}{Role: &testRoleList.Roles[0]})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(roleJson)
			return
		}
		if roleId == "test2id" {
			roleJson, err := json.Marshal(&struct {
				Role *Role `json:"role"`
			}{Role: &testRoleList.Roles[1]})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(roleJson)
			return
		}
		if roleId == "adminId" {
			roleJson, err := json.Marshal(&struct {
				Role *Role `json:"role"`
			}{Role: &Role{Name: "admin", ID: "admin", Description: "admin"}})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(roleJson)
			return
		}
		if roleId == "serviceId" {
			roleJson, err := json.Marshal(&struct {
				Role *Role `json:"role"`
			}{Role: &Role{Name: "service", ID: "service", Description: "service"}})
			if err != nil {
				response.WriteErrorString(http.StatusInternalServerError, err.Error())
				return
			}
			response.Write(roleJson)
			return
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	ws.Route(ws.GET("/v3/groups/{group_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		groupID := request.PathParameter("group_id")
		if groupID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		for _, group := range testGroupList.Groups {
			if group.ID == groupID {
				groupJson, err := json.Marshal(&struct {
					Group *Group `json:"group"`
				}{Group: &group})
				if err != nil {
					response.WriteErrorString(http.StatusInternalServerError, err.Error())
					return
				}
				response.Write(groupJson)
				return
			}
		}

		response.WriteHeader(http.StatusNotFound)
	}))
	ws.Route(ws.POST("/v3/users").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(request.Request.Body)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		var userContainer struct {
			User *User `json:"user"`
		}
		err = json.Unmarshal(body, &userContainer)
		if err != nil {
			response.WriteErrorString(http.StatusBadRequest, "failed to parse request body")
			return
		}

		if userContainer.User.Name == "test1" || userContainer.User.Name == "test2" || userContainer.User.Name == "test3" {
			response.WriteErrorString(http.StatusConflict, "user already exists")
			return
		}

		userCreated = userContainer.User

		response.WriteHeader(http.StatusCreated)
	}))
	ws.Route(ws.POST("/v3/roles").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(request.Request.Body)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		var roleContainer struct {
			Role *Role `json:"role"`
		}
		err = json.Unmarshal(body, &roleContainer)
		if err != nil {
			response.WriteErrorString(http.StatusBadRequest, "failed to parse request body")
			return
		}

		for _, role := range testRoleList.Roles {
			if role.Name == roleContainer.Role.Name {
				response.WriteErrorString(http.StatusConflict, "role already exists")
				return
			}
		}

		roleCreated = roleContainer.Role

		response.WriteHeader(http.StatusCreated)
		response.WriteAsJson(roleContainer)
	}))
	ws.Route(ws.POST("/v3/groups").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(request.Request.Body)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		var groupContainer struct {
			Group *Group `json:"group"`
		}
		err = json.Unmarshal(body, &groupContainer)
		if err != nil {
			response.WriteErrorString(http.StatusBadRequest, "failed to parse request body")
			return
		}

		for _, group := range testGroupList.Groups {
			if group.Name == groupContainer.Group.Name {
				response.WriteErrorString(http.StatusConflict, "group already exists")
				return
			}
		}

		groupCreated = groupContainer.Group
		response.WriteHeader(http.StatusCreated)
		response.WriteAsJson(groupContainer)
	}))
	ws.Route(ws.DELETE("/v3/users/{user_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID := request.PathParameter("user_id")
		if userID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusOK)
	}))
	ws.Route(ws.DELETE("/v3/roles/{role_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		roleID := request.PathParameter("role_id")
		if roleID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusOK)
	}))
	ws.Route(ws.DELETE("/v3/groups/{group_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		groupID := request.PathParameter("group_id")
		if groupID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusOK)
	}))
	ws.Route(ws.PATCH("/v3/users/{user_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID := request.PathParameter("user_id")
		if userID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(request.Request.Body)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		var userContainer struct {
			User *User `json:"user"`
		}
		err = json.Unmarshal(body, &userContainer)
		if err != nil {
			response.WriteErrorString(http.StatusBadRequest, "failed to parse request body")
			return
		}

		if userID == "testTestGroupUser" {
			testGroupUser.Groups = userContainer.User.CoreUser.Groups
			response.WriteHeader(http.StatusOK)
			return
		} else {
			for index, user := range testUserList.Users {
				if userID == user.ID {
					testUserList.Users[index].Password = userContainer.User.Password
					response.WriteHeader(http.StatusOK)
					return
				}
			}
		}

		response.WriteHeader(http.StatusNotFound)
	}))
	ws.Route(ws.PATCH("/v3/roles/{role_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		roleID := request.PathParameter("role_id")
		if roleID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(request.Request.Body)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		var roleContainer struct {
			Role *Role `json:"role"`
		}
		err = json.Unmarshal(body, &roleContainer)
		if err != nil {
			response.WriteErrorString(http.StatusBadRequest, "failed to parse request body")
			return
		}

		for index, role := range testRoleList.Roles {
			if roleID == role.ID {
				testRoleList.Roles[index].Description = roleContainer.Role.Description
				response.WriteHeader(http.StatusOK)
				return
			}
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	ws.Route(ws.PATCH("/v3/groups/{group_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		groupID := request.PathParameter("group_id")
		if groupID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(request.Request.Body)
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
		var groupContainer struct {
			Group *Group `json:"group"`
		}
		err = json.Unmarshal(body, &groupContainer)
		if err != nil {
			response.WriteErrorString(http.StatusBadRequest, "failed to parse request body")
			return
		}

		for index, group := range testGroupList.Groups {
			if groupID == group.ID {
				testGroupList.Groups[index].Description = groupContainer.Group.Description
				response.WriteHeader(http.StatusOK)
				return
			}
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	ws.Route(ws.PUT("/v3/groups/{group_id}/users/{user_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		groupID := request.PathParameter("group_id")
		userID := request.PathParameter("user_id")
		if groupID == "" || userID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		testGroupUser.Groups = append(testGroupUser.Groups, core.CoreGroup{Id: groupID})
		response.WriteHeader(http.StatusOK)
	}))
	ws.Route(ws.DELETE("/v3/groups/{group_id}/users/{user_id}").To(func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("X-Auth-Token")
		if token != "test-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}

		groupID := request.PathParameter("group_id")
		userID := request.PathParameter("user_id")
		if groupID == "" || userID == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		_, found := testUsersInGroup[groupID]
		if !found {
			response.WriteHeader(http.StatusNotFound)
			return
		}

		for index, user := range testUsersInGroup[groupID].Users {
			var result []User
			if user.ID == userID {
				result = append(testUsersInGroup[groupID].Users[:index], testUsersInGroup[groupID].Users[index+1:]...)
				testUsersInGroup[groupID] = struct{ Users []User }{Users: result}
				response.WriteHeader(http.StatusOK)
				return
			}
		}
		response.WriteHeader(http.StatusNotFound)
	}))
}

var _ = Describe("getRoleMap tests", func() {
	var serverConfig *config.Config
	BeforeEach(func() {
		coreApiLog.InitLogger("DEBUG")
		serverConfig = &config.Config{
			AuthConfig: &config.AuthConfig{
				Keystone: &config.KeystoneConfig{
					Endpoint:  "http://localhost:20001",
					Username:  "admin",
					Password:  "admin",
					DomainId:  "default",
					ProjectId: "default",
				},
			},
		}
	})

	Describe("getRoleMap test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20001"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20001, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			roles, err := getRoleMap(serverConfig, "test-token", nil)
			Expect(err).To(BeNil())
			Expect(len(roles)).To(Equal(2))
		})
		It("should be expected without filter out keystone role", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20001"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20001, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			roles, err := getRoleMap(serverConfig, "test-token", map[string]struct{}{LoadUnRelatedKeystoneObjects: {}})
			Expect(err).To(BeNil())
			Expect(len(roles)).To(Equal(len(testRoleList.Roles)))
		})
	})

	Describe("getGroupMap test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20001"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20001, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			groups, err := getGroupMap(serverConfig, "test-token", nil)
			Expect(err).To(BeNil())
			Expect(len(groups)).To(Equal(3))
		})
		It("should be expected without filter out keystone group", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20001"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20001, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			groups, err := getGroupMap(serverConfig, "test-token", map[string]struct{}{LoadUnRelatedKeystoneObjects: {}})
			Expect(err).To(BeNil())
			Expect(len(groups)).To(Equal(len(testGroupList.Groups)))
		})
	})
})

var _ = Describe("keystone common test", func() {
	var keystoneUser, keystoneUser2, keystoneUser3, keystoneUser4 *User
	var keystoneRole, keystoneRole2, keystoneRole3 *Role
	var AllRoles map[string]core.CoreRole
	BeforeEach(func() {
		keystoneUser = &User{
			ID:       "test1id",
			Name:     "test1",
			Password: "test1",
			Email:    "test1@test.com",
			Enabled:  true,
			CoreUser: &core.CoreUser{
				Id:          "core-test1id",
				Name:        "core-test1",
				Email:       "core-test1@test.com",
				Description: "core-test1",
				Password:    "core-test1",
				Roles:       []core.CoreRole{{Name: "core-test1", Permission: map[string]uint64{"core-test1": 1}}},
			},
		}

		keystoneUser2 = &User{
			ID:       "test2id",
			Name:     "test2",
			Password: "test2",
			Email:    "test2@test.com",
			Enabled:  true,
		}

		keystoneUser3 = &User{
			ID:       "adminId",
			Name:     "admin",
			Password: "admin",
			Email:    "admin@test.com",
			Enabled:  true,
		}

		keystoneUser4 = &User{
			ID:       "serviceId",
			Name:     "service",
			Password: "service",
			Email:    "service@test.com",
			Enabled:  true,
		}

		keystoneRole = &Role{
			ID:          "test1id",
			Name:        "test1",
			Description: "test1",
			Links:       &Links{},
			CoreRole: &core.CoreRole{
				Id:          "test1id",
				Name:        "test1",
				Description: "test1",
				Permission: map[string]uint64{
					"course": 1,
				},
			},
		}

		keystoneRole2 = &Role{
			ID:    "test2id",
			Name:  "test2",
			Links: &Links{},
			CoreRole: &core.CoreRole{
				Id:          "test2id",
				Name:        "test2",
				Description: "core-test2",
				Permission: map[string]uint64{
					"course":  0,
					"dataset": 2,
				},
			},
		}

		keystoneRole3 = &Role{
			ID:    "adminId",
			Name:  "admin",
			Links: &Links{},
			CoreRole: &core.CoreRole{
				Id:          "adminId",
				Name:        "admin",
				Description: "core-admin",
				Permission:  privileges.ModulesFullPermission(),
			},
		}

		AllRoles = map[string]core.CoreRole{
			keystoneRole.ID:  *keystoneRole.CoreRole,
			keystoneRole2.ID: *keystoneRole2.CoreRole,
			keystoneRole3.ID: *keystoneRole3.CoreRole,
		}
	})
	Describe("sumPermisson test", func() {
		It("should be expected", func() {
			testRoles := []core.CoreRole{
				*keystoneRole.CoreRole,
				*keystoneRole2.CoreRole,
			}
			sumedPermission := sumPermission(testRoles, AllRoles)
			Expect(sumedPermission).To(Equal(map[string]uint64{
				"course":            1,
				"dataset":           2,
				"device":            0,
				"user":              0,
				"role":              0,
				"rag":               0,
				"setting":           0,
				"group":             0,
				"flavor":            0,
				"indexView":         0,
				"courseStudentView": 0,
				"deviceStudentView": 0,
				"model":             0,
			}))
		})
		It("should be expected with full module permisson", func() {
			testRoles := []core.CoreRole{
				*keystoneRole.CoreRole,
				*keystoneRole2.CoreRole,
				*keystoneRole3.CoreRole,
			}
			sumedPermission := sumPermission(testRoles, AllRoles)
			Expect(sumedPermission).To(Equal(privileges.ModulesFullPermission()))
		})
	})
	Describe("ConvertKeystoneUserToCoreUser test", func() {
		It("should be match core user without password", func() {
			result := ConvertKeystoneUserToCoreUser(keystoneUser, true, nil)
			Expect(result.Id).To(Equal(keystoneUser.CoreUser.Id))
			Expect(result.Name).To(Equal(keystoneUser.CoreUser.Name))
			Expect(result.Email).To(Equal(keystoneUser.CoreUser.Email))
			Expect(result.Password).To(Equal(keystoneUser.CoreUser.Password))
			Expect(result.Roles).To(Equal(keystoneUser.CoreUser.Roles))
		})
		It("should be match keystone user without password", func() {
			result := ConvertKeystoneUserToCoreUser(keystoneUser2, true, nil)
			Expect(result.Id).To(Equal(keystoneUser2.ID))
			Expect(result.Name).To(Equal(keystoneUser2.Name))
			Expect(result.Email).To(Equal(keystoneUser2.Email))
			Expect(result.Password).To(Equal(""))
		})
		It("should be match keystone user with password", func() {
			result := ConvertKeystoneUserToCoreUser(keystoneUser2, false, nil)
			Expect(result.Id).To(Equal(keystoneUser2.ID))
			Expect(result.Name).To(Equal(keystoneUser2.Name))
			Expect(result.Email).To(Equal(keystoneUser2.Email))
			Expect(result.Password).To(Equal(keystoneUser2.Password))
		})
		It("should be match core user with uneditable", func() {
			result := ConvertKeystoneUserToCoreUser(keystoneUser3, true, nil)
			Expect(result.UnEditable).To(Equal(true))
		})
		It("should be match core user with uneditable", func() {
			result := ConvertKeystoneUserToCoreUser(keystoneUser4, true, nil)
			Expect(result.UnEditable).To(Equal(true))
		})
		It("should be match core user with permission case1", func() {
			result := ConvertKeystoneUserToCoreUser(keystoneUser, true, privileges.ModulesNoPermission())
			Expect(result.Permission).To(Equal(privileges.ModulesNoPermission()))
		})
		It("should be match core user with permission case2", func() {
			permission := privileges.ModulesNoPermission()
			permission["course"] = privileges.PermissionCourseList | privileges.PermissionCourseCreate
			result := ConvertKeystoneUserToCoreUser(keystoneUser, true, permission)
			Expect(result.Permission).To(Equal(permission))
		})
		It("should be match core user with permission case3", func() {
			result := ConvertKeystoneUserToCoreUser(keystoneUser, true, nil)
			Expect(result.Permission).To(Equal(privileges.ModulesNoPermission()))
		})
	})

	Describe("ConvertKeystoneRoleToCoreRole test", func() {
		It("should be match core role", func() {
			result := ConvertKeystoneRoleToCoreRole(keystoneRole)
			Expect(result.Id).To(Equal(keystoneRole.CoreRole.Id))
			Expect(result.Name).To(Equal(keystoneRole.CoreRole.Name))
			Expect(result.Permission).To(Equal(keystoneRole.CoreRole.Permission))
		})

		It("should be match keystone role", func() {
			result := ConvertKeystoneRoleToCoreRole(keystoneRole2)
			Expect(result.Id).To(Equal(keystoneRole2.ID))
			Expect(result.Name).To(Equal(keystoneRole2.Name))
		})
	})
})

var _ = Describe("keystone role tests", func() {
	var serverConfig *config.Config
	var roleProvider *RoleProvider
	BeforeEach(func() {
		coreApiLog.InitLogger("DEBUG")
		serverConfig = &config.Config{
			AuthConfig: &config.AuthConfig{
				Keystone: &config.KeystoneConfig{
					Endpoint:  "http://localhost:20081",
					Username:  "admin",
					Password:  "admin",
					DomainId:  "default",
					ProjectId: "default",
				},
			},
		}
		roleProvider = &RoleProvider{
			Config: serverConfig,
		}
	})

	Describe("GetRoles test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20081"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20081, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			roles, err := roleProvider.GetRoles(map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(len(roles)).To(Equal(2))
		})
		It("should be expected without filter out keystone role", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20081"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20081, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			roles, err := roleProvider.GetRoles(map[string]struct{}{LoadUnRelatedKeystoneObjects: {}})
			Expect(err).To(BeNil())
			Expect(len(roles)).To(Equal(len(testRoleList.Roles)))
		})
	})

	Describe("GetRawKeystoneRoleList test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20001"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20001, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			roles, err := roleProvider.GetRawKeystoneRoleList()
			Expect(err).To(BeNil())
			Expect(len(roles.Roles)).To(Equal(len(testRoleList.Roles)))
		})
	})

	Describe("GetRole test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20002"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20002, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			role, err := roleProvider.GetRole("test1id", map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(role.Id).To(Equal("test1id"))
			Expect(role.Name).To(Equal("test1"))
			Expect(role.Description).To(Equal("test1"))
		})
	})

	Describe("UpdateRole test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20003"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20003, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := roleProvider.UpdateRole(&core.CoreRole{
				Id:          "test1id",
				Name:        "test1",
				Description: "new-description",
			}, map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(testRoleList.Roles[0].Description).To(Equal("new-description"))
		})
		It("should be rejected due to role admin cannot be modified", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20004"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20004, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := roleProvider.UpdateRole(&core.CoreRole{
				Id:          "adminId",
				Name:        "admin",
				Description: "new-description",
			}, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
		It("should be rejected due to role service cannot be modified", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20005"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20005, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := roleProvider.UpdateRole(&core.CoreRole{
				Id:          "serviceId",
				Name:        "service",
				Description: "new-description",
			}, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("DeleteRole test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20006"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20006, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := roleProvider.DeleteRole("test1id", map[string]struct{}{})
			Expect(err).To(BeNil())
		})
		It("should be rejected due to role admin cannot be deleted", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20007"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20007, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := roleProvider.DeleteRole("adminId", map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
		It("should be rejected due to role service cannot be deleted", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20008"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20008, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := roleProvider.DeleteRole("serviceId", map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("SearchRoleByName test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20009"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20009, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			role, err := roleProvider.SearchRoleByName("test1", map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(role.Id).To(Equal("test1id"))
			Expect(role.Name).To(Equal("test1"))
		})
	})

	Describe("CreateRole test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20009"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20009, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			_, err := roleProvider.CreateRole(&core.CoreRole{
				Name:        "test4",
				Description: "test4",
			}, map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(roleCreated.Name).To(Equal("test4"))
			Expect(roleCreated.Description).To(Equal("test4"))
		})
		It("should be rejected due to role already exists", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20009"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20010, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			_, err := roleProvider.CreateRole(&core.CoreRole{
				Name:        "test1",
				Description: "test1",
			}, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
	})
})

var _ = Describe("keystone group tests", func() {
	var serverConfig *config.Config
	var keystone *GroupProvider
	BeforeEach(func() {
		coreApiLog.InitLogger("DEBUG")
		serverConfig = &config.Config{
			AuthConfig: &config.AuthConfig{
				Keystone: &config.KeystoneConfig{
					Endpoint:  "http://localhost:20081",
					Username:  "admin",
					Password:  "admin",
					DomainId:  "default",
					ProjectId: "default",
				},
			},
		}
		keystone = &GroupProvider{
			Config: serverConfig,
		}
	})
	Describe("GetRawKeystoneGroupList test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20001"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20001, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			groups, err := keystone.GetRawKeystoneGroupList()
			Expect(err).To(BeNil())
			Expect(len(groups.Groups)).To(Equal(len(testGroupList.Groups)))
		})
	})
	Describe("GetGroup test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20002"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20002, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			group, err := keystone.GetGroup("test1id", map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(group.Id).To(Equal("test1id"))
			Expect(group.Name).To(Equal("test1"))
			Expect(group.Description).To(Equal("test1"))
			Expect(group.TagColor).To(Equal("test1Red"))
		})
		It("should be expected without filter out keystone group", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20002"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20002, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			group, err := keystone.GetGroup("adminId", map[string]struct{}{LoadUnRelatedKeystoneObjects: {}})
			Expect(err).To(BeNil())
			Expect(group.Id).To(Equal("adminId"))
			Expect(group.Name).To(Equal("admin"))
			Expect(group.TagColor).To(Equal(""))
		})
		It("should be expected filter out keystone group", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20002"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20002, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			group, err := keystone.GetGroup("serviceId", map[string]struct{}{})
			Expect(err).NotTo(BeNil())
			Expect(group).To(BeNil())
		})
	})
	Describe("GetGroups test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20003"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20003, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			groups, err := keystone.GetGroups(map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(len(groups)).To(Equal(3))
		})
		It("should be expected without filter out keystone group", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20003"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20003, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			groups, err := keystone.GetGroups(map[string]struct{}{LoadUnRelatedKeystoneObjects: {}})
			Expect(err).To(BeNil())
			Expect(len(groups)).To(Equal(len(testGroupList.Groups)))
		})
	})
	Describe("UpdateGroup test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20004"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20004, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.UpdateGroup(&core.CoreGroup{
				Id:          "test1id",
				Name:        "test1",
				Description: "new-description",
			}, map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(testGroupList.Groups[0].Description).To(Equal("new-description"))
		})
		It("should be rejected due to group admin cannot be modified", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20005"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20005, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.UpdateGroup(&core.CoreGroup{
				Id:          "adminId",
				Name:        "admin",
				Description: "new-description",
			}, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
	})
	Describe("DeleteGroup test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20006"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20006, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.DeleteGroup("test1id", map[string]struct{}{})
			Expect(err).To(BeNil())
		})
		It("should be rejected due to group admin cannot be deleted", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20007"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20007, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.DeleteGroup("adminId", map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("CreateGroup test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20008"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20008, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			_, err := keystone.CreateGroup(&core.CoreGroup{
				Name:        "test4",
				Description: "test4",
				TagColor:    "test4Red",
			}, map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(groupCreated.Name).To(Equal("test4"))
			Expect(groupCreated.Description).To(Equal("test4"))
			Expect(groupCreated.CoreGroup.TagColor).To(Equal("test4Red"))
		})
	})

	Describe("SearchGroupByName test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20008"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20008, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			group, err := keystone.SearchGroupByName("test1", map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(group.Id).To(Equal("test1id"))
			Expect(group.Name).To(Equal("test1"))
		})
	})

	Describe("GetGroupUsers test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20009"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20009, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetGroupUsers("test1id", map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(len(users)).To(Equal(2))
		})
		It("should be expected with 2 users", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20009"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20009, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetGroupUsers("test2id", map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(len(users)).To(Equal(1))
		})
		It("should be expected reverse the result", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20009"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20009, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetGroupUsers("test2id", map[string]struct{}{ReverseGetGroupUsers: {}})
			Expect(err).To(BeNil())
			Expect(len(users)).To(Equal(4))
		})
	})
	Describe("AddUserToGroup test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20010"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20010, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.AddUserToGroup("testTestGroupUser", "adminId")
			Expect(err).To(BeNil())
			Expect(len(testGroupUser.Groups)).To(Equal(3))
		})
		It("should be not changed due to user already in that group", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20010"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20010, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.AddUserToGroup("testTestGroupUser", "test2id")
			Expect(err).To(BeNil())
			Expect(len(testGroupUser.Groups)).To(Equal(3))
		})
		It("should be rejected due to group not found", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20010"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20011, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.AddUserToGroup("test3id", "test9id")
			Expect(err).NotTo(BeNil())
		})
		It("should be rejected due to user not found", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20010"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20012, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.AddUserToGroup("test9id", "test2id")
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("RemoveUserFromGroup test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20010"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20010, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.RemoveUserFromGroup("testTestGroupUser", "test1id")
			Expect(err).To(BeNil())
			Expect(len(testUsersInGroup["test2id"].Users)).To(Equal(2))
		})
		It("should be rejected due to group not found", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20010"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20010, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.RemoveUserFromGroup("test3id", "test9id")
			Expect(err).NotTo(BeNil())
		})
		It("should be rejected due to user not found", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20010"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20010, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.RemoveUserFromGroup("test9id", "test2id")
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("GetGroupSummary test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20011"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20011, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			summary, err := keystone.GetGroupSummary(map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(len(summary.Counts)).To(Equal(3))
			// note fake http server using map which means it's not order guarantee
			// that's why we need to loop to find the expected value
			for _, count := range summary.Counts {
				if count.Name == "test1" {
					Expect(count.Count).To(Equal(2))
				}
				if count.Name == "test2" {
					Expect(count.Count).To(Equal(1))
				}
				if count.Name == "admin" {
					Expect(count.Count).To(Equal(0))
				}
			}
		})
	})
})

var _ = Describe("keystone user tests", func() {
	var serverConfig *config.Config
	var keystone *UserProvider
	BeforeEach(func() {
		coreApiLog.InitLogger("DEBUG")
		serverConfig = &config.Config{
			AuthConfig: &config.AuthConfig{
				Keystone: &config.KeystoneConfig{
					Endpoint:  "http://localhost:20081",
					Username:  "admin",
					Password:  "admin",
					DomainId:  "default",
					ProjectId: "default",
				},
			},
		}
		keystone = &UserProvider{
			Config: serverConfig,
		}
	})
	Describe("RequestToken test", func() {
		It("should be expected", func() {
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20081, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			token, tokenResp, err := RequestToken(serverConfig.AuthConfig.Keystone.Username, serverConfig.AuthConfig.Keystone.Password, serverConfig.AuthConfig.Keystone.DomainId, serverConfig.AuthConfig.Keystone.Endpoint, serverConfig.AuthConfig.Keystone.TokenKeyInResponse, true)
			Expect(err).To(BeNil())
			Expect(token).To(Equal("test-token"))
			Expect(tokenResp.Token.User.Name).To(Equal("admin"))

		})
	})

	Describe("getDomain test", func() {
		It("should be the default value", func() {
			Expect(getDomainOrDefault(serverConfig.AuthConfig.Keystone.DomainId)).To(Equal("default"))
		})

		It("should be equal to test", func() {
			serverConfig.AuthConfig.Keystone.DomainId = "test"
			Expect(getDomainOrDefault(serverConfig.AuthConfig.Keystone.DomainId)).To(Equal("test"))
		})
	})

	Describe("getToken test", func() {
		It("should be expected", func() {
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20081, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			token, err := keystone.getToken()
			Expect(err).To(BeNil())
			Expect(token).To(Equal("test-token"))
			token, err = keystone.getToken()
			Expect(err).To(BeNil())
			Expect(token).To(Equal("test-token"))
		})
	})

	Describe("buildPath test", func() {
		It("should be expected", func() {
			Expect(common.BuildPath(serverConfig.AuthConfig.Keystone.Endpoint, "/v3/auth/tokens")).To(Equal("http://localhost:20081/v3/auth/tokens"))
			Expect(common.BuildPath(serverConfig.AuthConfig.Keystone.Endpoint, "v3/users")).To(Equal("http://localhost:20081/v3/users"))
		})
	})

	Describe("commentRequestAutoRenewToken test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20082"
			keystone.token = "wrong token"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20082, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			_, _, httpCode, err := commentRequestAutoRenewToken("/v3/users", http.MethodGet, serverConfig.AuthConfig.Keystone, keystone.getToken, keystone.setToken, nil)
			Expect(err).To(BeNil())
			Expect(httpCode).To(Equal(http.StatusOK))

		})
	})

	Describe("GetRawKeystoneUserList test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20088"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20088, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetRawKeystoneUserList()
			Expect(err).To(BeNil())
			Expect(len(users.Users)).To(Equal(len(testUserList.Users) + 1))
		})
	})

	Describe("GetUserIdFromName test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20089"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20089, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			id, err := keystone.GetUserIdFromName("test1")
			Expect(err).To(BeNil())
			Expect(id).To(Equal("test1id"))
		})
	})

	Describe("UpdateUser test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20090"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20090, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.UpdateUser(&core.CoreUser{
				Id:       "test4id",
				Password: "new-password",
			}, map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(testUserList.Users[4].Password).To(Equal("new-password"))

			err = keystone.UpdateUser(&core.CoreUser{
				Id:       "test99",
				Password: "new-password",
			}, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("ListUsers test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20083"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20083, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetUsers(map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(len(users)).To(Equal(len(testUserList.Users) - 1))
			Expect(users[0].Id).To(Equal(testUserList.Users[0].ID))
			Expect(users[0].Name).To(Equal(testUserList.Users[0].Name))
			Expect(users[0].Email).To(Equal(testUserList.Users[0].Email))
			Expect(users[0].Roles[0].Name).To(Equal("test1"))
			Expect(users[1].Name).To(Equal(testUserList.Users[1].Name))
			Expect(users[1].Password).To(Equal(""))
			Expect(users[1].Roles[0].Name).To(Equal("test2"))
			Expect(users[1].Email).To(Equal(testUserList.Users[1].CoreUser.Email))
			Expect(users[2].Name).To(Equal(testUserList.Users[2].Name))
			Expect(users[2].Email).To(Equal(testUserList.Users[2].Email))
		})
		It("should be expected without filter out keystone objects", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20083"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20083, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetUsers(map[string]struct{}{LoadUnRelatedKeystoneObjects: {}})
			Expect(err).To(BeNil())
			Expect(len(users)).To(Equal(len(testUserList.Users)))
		})

		It("should be expected with no permission loaded", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20080"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20080, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetUsers(map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(len(users)).To(Equal(len(testUserList.Users) - 1))
			Expect(users[0].Permission).To(Equal(privileges.ModulesNoPermission()))
			Expect(users[1].Permission).To(Equal(privileges.ModulesNoPermission()))
			Expect(users[2].Permission).To(Equal(privileges.ModulesNoPermission()))
		})

		It("should be expected with permission loaded", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20079"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20079, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			users, err := keystone.GetUsers(map[string]struct{}{LoadPermission: {}})
			Expect(err).To(BeNil())
			Expect(len(users)).To(Equal(len(testUserList.Users) - 1))
			Expect(users[0].Permission).To(Equal(map[string]uint64{
				"course":            privileges.PermissionCourseViewPage,
				"dataset":           0,
				"device":            0,
				"user":              0,
				"role":              0,
				"rag":               0,
				"setting":           0,
				"group":             0,
				"flavor":            0,
				"indexView":         0,
				"courseStudentView": 0,
				"deviceStudentView": 0,
				"model":             0,
			}))
			Expect(users[1].Permission).To(Equal(map[string]uint64{
				"course":            0,
				"dataset":           privileges.PermissionDatasetList,
				"device":            0,
				"user":              0,
				"role":              0,
				"rag":               0,
				"setting":           0,
				"group":             0,
				"flavor":            0,
				"indexView":         0,
				"courseStudentView": 0,
				"deviceStudentView": 0,
				"model":             0,
			}))
			Expect(users[2].Permission).To(Equal(map[string]uint64{
				"course":            privileges.PermissionCourseViewPage,
				"dataset":           privileges.PermissionDatasetList,
				"device":            0,
				"user":              0,
				"role":              0,
				"rag":               0,
				"setting":           0,
				"group":             0,
				"flavor":            0,
				"indexView":         0,
				"courseStudentView": 0,
				"deviceStudentView": 0,
				"model":             0,
			}))
		})
	})

	Describe("GetUser test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20084"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20084, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			user, err := keystone.GetUser("test1id", map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(user.Id).To(Equal(testUserList.Users[0].ID))
			Expect(user.Name).To(Equal(testUserList.Users[0].Name))
			Expect(user.Roles).To(Equal(testUserList.Users[0].CoreUser.Roles))
			Expect(user.Email).To(Equal(testUserList.Users[0].CoreUser.Email))
		})
		It("should be expected with permisson", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20084"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20084, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			user, err := keystone.GetUser("test3id", map[string]struct{}{LoadPermission: {}})
			Expect(err).To(BeNil())
			Expect(user.Permission).To(Equal(map[string]uint64{
				"course":            privileges.PermissionCourseViewPage,
				"dataset":           privileges.PermissionDatasetList,
				"device":            0,
				"user":              0,
				"role":              0,
				"rag":               0,
				"setting":           0,
				"group":             0,
				"flavor":            0,
				"indexView":         0,
				"courseStudentView": 0,
				"deviceStudentView": 0,
				"model":             0,
			}))
		})
	})

	Describe("CreateUser test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20085"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20085, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			userToCreate := &core.CoreUser{
				Name:     "test4",
				Email:    "",
				Password: "test4",
			}
			_, err := keystone.CreateUser(userToCreate, map[string]struct{}{})
			Expect(err).To(BeNil())
			// test conflict user will be rejected
			userToCreate.Name = "test1"
			_, err = keystone.CreateUser(userToCreate, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
			Expect(userCreated.Name).To(Equal("test4"))
		})

		It("should be error due to role not found", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20085"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20085, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			userToCreate := &core.CoreUser{
				Name:     "test4",
				Email:    "",
				Password: "test4",
				Roles:    []core.CoreRole{{Id: "test4id"}},
			}
			_, err := keystone.CreateUser(userToCreate, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
			_, ok := err.(*customErr.NotFound)
			Expect(ok).To(BeTrue())
			Expect(err.Error()).To(Equal("Error code: 404, message: Role test4id not found"))
		})

		It("should be error due to group not found", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20085"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20085, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			userToCreate := &core.CoreUser{
				Name:     "test4",
				Email:    "",
				Password: "test4",
				Groups:   []core.CoreGroup{{Id: "test4id"}},
			}
			_, err := keystone.CreateUser(userToCreate, map[string]struct{}{})
			Expect(err).NotTo(BeNil())
			_, ok := err.(*customErr.NotFound)
			Expect(ok).To(BeTrue())
			Expect(err.Error()).To(Equal("Error code: 404, message: Group test4id not found"))
		})

		It("should be ok with role and group", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20085"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20085, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			userToCreate := &core.CoreUser{
				Name:     "test4",
				Email:    "",
				Password: "test4",
				Roles:    []core.CoreRole{{Id: "test1id"}},
				Groups:   []core.CoreGroup{{Id: "test1id"}},
			}
			_, err := keystone.CreateUser(userToCreate, map[string]struct{}{})
			Expect(err).To(BeNil())
			Expect(userCreated.Name).To(Equal("test4"))
			Expect(userCreated.CoreUser.Roles[0].Id).To(Equal("test1id"))
			Expect(userCreated.CoreUser.Groups[0].Id).To(Equal("test1id"))
		})
	})

	Describe("DeleteUser test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20086"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20086, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.DeleteUser("test1id", map[string]struct{}{})
			Expect(err).To(BeNil())
		})

		It("should be rejected due to admin account not allowed to delete", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20088"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20088, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.DeleteUser("adminid", map[string]struct{}{})
			Expect(err).To(HaveOccurred())
		})

		It("should be rejected due to service account not allowed to delete", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20089"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20089, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			err := keystone.DeleteUser("adminid", map[string]struct{}{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("LoginUser test", func() {
		It("should be expected", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20087"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20087, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			user, err := keystone.LoginUser("login", "login")
			Expect(err).To(BeNil())
			Expect(user.Name).To(Equal("login"))
			Expect(user.Password).To(Equal(""))
			Expect(user.Permission).To(Equal(map[string]uint64{
				"course":            0,
				"dataset":           0,
				"device":            0,
				"user":              0,
				"role":              0,
				"rag":               0,
				"setting":           0,
				"group":             0,
				"flavor":            0,
				"indexView":         0,
				"courseStudentView": 0,
				"deviceStudentView": 0,
				"model":             0,
			}))
		})
		It("should be rejected due status is not 201", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20087"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20087, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			_, err := keystone.LoginUser("failedToLogin", "failedToLogin")
			Expect(err).To(HaveOccurred())
			// check err is unauthorized
			_, ok := err.(*customErr.Unauthorized)
			Expect(ok).To(BeTrue())
		})
		It("should be rejected due to token not in header", func() {
			serverConfig.AuthConfig.Keystone.Endpoint = "http://localhost:20087"
			stopChan := make(chan struct{}, 1)
			go common.StartMockServer(20087, testRouter, stopChan)
			time.Sleep(1 * time.Second)
			defer close(stopChan)
			_, err := keystone.LoginUser("noToken", "noToken")
			Expect(err).To(HaveOccurred())
			// check err is unauthorized
			_, ok := err.(*customErr.Unauthorized)
			Expect(ok).To(BeTrue())
		})
	})
})
