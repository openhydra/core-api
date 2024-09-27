package custom_middleware

import (
	"context"
	"core-api/pkg/core/auth"
	keystone "core-api/pkg/core/auth/provider/keystone/train"
	"core-api/pkg/core/privileges"
	v1 "core-api/pkg/north/api/user/core/v1"
	customError "core-api/pkg/util/error"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	coreApiLog "core-api/pkg/logger"

	northApiRoute "core-api/pkg/north/api/route"

	"github.com/go-chi/chi/v5"
)

type defaultCoreBasicAuth struct {
	PrivilegesProvider      privileges.IPrivilegeProvider
	UserProvider            auth.IUserProvider
	userAuthenticationCache *sync.Map
	stopChan                <-chan struct{}
	innerStopChan           chan struct{}
	routeProvider           northApiRoute.IRouteProvider
	whiteList               map[string]struct{}
}

func getRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if pattern := rctx.RoutePattern(); pattern != "" {
		// Pattern is already available
		return pattern
	}

	routePath := r.URL.Path
	if r.URL.RawPath != "" {
		routePath = r.URL.RawPath
	}

	tctx := chi.NewRouteContext()
	if !rctx.Routes.Match(tctx, r.Method, routePath) {
		// No matching pattern, so just return the request path.
		// Depending on your use case, it might make sense to
		// return an empty string or error here instead
		return routePath
	}

	// tctx has the updated pattern, since Match mutates it
	return tctx.RoutePattern()
}

func (cba *defaultCoreBasicAuth) BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// we use closures to pass the request and response to the next handler
		// and mainly we can use the request and response to do some operations

		// check if the route is in the white list
		selectedRoute := getRoutePattern(r)
		if _, found := cba.whiteList[selectedRoute]; found {
			next.ServeHTTP(w, r)
			return
		}

		// first we authenticate the user
		code, user, err := cba.authentication(r)
		if err != nil {
			coreApiLog.Logger.Error("failed to authenticate", "error", err)
			http.Error(w, err.Error(), code)
			return
		}

		// then we authorize the user
		code, err = cba.authorization(user, selectedRoute, r.Method)
		if err != nil {
			coreApiLog.Logger.Error("failed to authorize", "error", err)
			http.Error(w, err.Error(), code)
			return
		}

		ctx := context.WithValue(r.Context(), "core-user", user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (cba *defaultCoreBasicAuth) RunBackgroundCache() {
	if cba.stopChan == nil {
		coreApiLog.Logger.Warn("stop channel is nil, background cache will not run")
		return
	}

	go func() {
		cba.renewUserAuthenticationCache()
		ticker := time.Tick(10 * time.Second)
		for range ticker {
			// clear all cache
			coreApiLog.Logger.Debug("background worker renewing user cache")
			cba.renewUserAuthenticationCache()
		}
	}()

	select {
	case <-cba.stopChan:
		coreApiLog.Logger.Info("stop channel is closed, stopping background cache")
		return
	case <-cba.innerStopChan:
		coreApiLog.Logger.Info("inner stop channel is closed, stopping background cache")
		return
	}

	// we can run background cache here
}

func (cba *defaultCoreBasicAuth) StopBackgroundCache() {
	if cba.stopChan == nil {
		coreApiLog.Logger.Warn("stop channel is nil, background cache not running")
		return
	}
	close(cba.innerStopChan)
}

// if stopChan is provided, we will run the background cache
// we update a full list of users every 10 seconds from the user provider api
func (cba *defaultCoreBasicAuth) renewUserAuthenticationCache() {
	users, err := cba.UserProvider.GetUsers(map[string]struct{}{keystone.LoadPasswd: {}, keystone.LoadPermission: {}})
	if err != nil {
		coreApiLog.Logger.Error("failed to renew user cache", "error", err)
		return
	}

	coreApiLog.Logger.Debug("Attempting to renewing user cache with", "total", len(users))
	tempSyncMap := &sync.Map{}
	count := 0
	for index, u := range users {
		tempSyncMap.Store(u.Name, &users[index])
		count++
	}

	coreApiLog.Logger.Debug("renew user cache success with", "total", count)
	// replace the old cache with the new one
	cba.userAuthenticationCache = tempSyncMap
}

func (cba *defaultCoreBasicAuth) authentication(r *http.Request) (int, *v1.CoreUser, error) {
	// get header Bear token from request
	token := r.Header.Get("Authorization")
	if token == "" {
		return http.StatusUnauthorized, nil, fmt.Errorf("no token found in header with authorization enabled")
	}

	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
		// base64 decode the token
		decodedToken, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			coreApiLog.Logger.Error("failed to decode token", "error", err)
			return http.StatusUnauthorized, nil, fmt.Errorf("failed to decode token")
		}
		authSet := strings.Split(string(decodedToken), ":")
		if len(authSet) != 2 {
			return http.StatusUnauthorized, nil, fmt.Errorf("invalid token format")
		}

		// first we go with local cache to speed up the process
		if user, ok := cba.userAuthenticationCache.Load(authSet[0]); ok {
			coreApiLog.Logger.Info("hitting user info in cache go for it", "user", user)
			coreUser := user.(*v1.CoreUser)
			if coreUser.Password == authSet[1] {
				return http.StatusOK, coreUser, nil
			} else {
				return http.StatusUnauthorized, nil, fmt.Errorf("password or username is incorrect")
			}
		} else {
			// go with userProvider to query api
			user, err := cba.UserProvider.LoginUser(authSet[0], authSet[1])
			if err != nil {
				// assert error is not found error
				if _, ok := err.(*customError.NotFound); ok {
					return http.StatusNotFound, nil, fmt.Errorf("user not found")
				} else {
					return http.StatusUnauthorized, nil, fmt.Errorf("failed to query user")
				}
			}
			return http.StatusOK, user, nil
		}
	} else {
		return http.StatusUnauthorized, nil, fmt.Errorf("invalid token")
	}
}

func (cba *defaultCoreBasicAuth) authorization(user *v1.CoreUser, selectedRoute, method string) (int, error) {
	routePermission := cba.routeProvider.GetRouteAuthorization()

	if _, found := routePermission[selectedRoute]; !found {
		return http.StatusForbidden, fmt.Errorf("access denied for route %s of user %s", selectedRoute, user.Name)
	} else {
		if _, methodFound := routePermission[selectedRoute][method]; !methodFound {
			return http.StatusForbidden, fmt.Errorf("access denied for method %s of route %s with user %s", method, selectedRoute, user.Name)
		} else {
			moduleSet := routePermission[selectedRoute][method]
			priProvider := &privileges.DefaultPrivilegeProvider{}
			canAccess, err := priProvider.CanAccess(user.Permission, moduleSet.Module, moduleSet.Permission)
			if err != nil {
				return http.StatusForbidden, fmt.Errorf("failed to check permission for method %s of route %s with user %s", method, selectedRoute, user.Name)
			}
			if !canAccess {
				return http.StatusForbidden, fmt.Errorf("access denied for method %s of route %s with user %s", method, selectedRoute, user.Name)
			}
		}
	}

	return http.StatusOK, nil
}

func (cba *defaultCoreBasicAuth) GetWhiteListedRoutes() []string {
	routes := make([]string, 0, len(cba.whiteList))
	for route := range cba.whiteList {
		routes = append(routes, route)
	}
	return routes
}

func (cba *defaultCoreBasicAuth) SetWhiteListedRoutes(routes []string) {
	cba.whiteList = make(map[string]struct{}, len(routes))
	for _, route := range routes {
		cba.whiteList[route] = struct{}{}
	}
}
