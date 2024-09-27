package custom_middleware

import (
	"core-api/pkg/core/auth"
	"core-api/pkg/core/privileges"
	northApiRoute "core-api/pkg/north/api/route"
	"fmt"
	"net/http"
	"sync"
)

var defaultCoreBasicAuthInstance *defaultCoreBasicAuth

type IBasicAuth interface {
	BasicAuth(next http.Handler) http.Handler
	RunBackgroundCache()
	StopBackgroundCache()
	GetWhiteListedRoutes() []string
	SetWhiteListedRoutes(routes []string)
}

type CoreBaseAuthType string

const (
	DefaultCoreBaseAuth CoreBaseAuthType = "default"
)

func NewCoreBaseAuth(priProvider privileges.IPrivilegeProvider, authProvider auth.IUserProvider, baType CoreBaseAuthType, routeProvider northApiRoute.IRouteProvider, stopChan <-chan struct{}) (IBasicAuth, error) {
	if priProvider == nil {
		return nil, fmt.Errorf("privilege provider is nil")
	}

	if authProvider == nil {
		return nil, fmt.Errorf("auth provider is nil")
	}

	switch baType {
	case DefaultCoreBaseAuth:
		return newOrGetDefaultCoreBasicAuth(priProvider, authProvider, routeProvider, stopChan), nil
	}

	return nil, fmt.Errorf("unknown core base auth type: '%s'", baType)
}

// make it singleton
func newOrGetDefaultCoreBasicAuth(priProvider privileges.IPrivilegeProvider, authProvider auth.IUserProvider, routeProvider northApiRoute.IRouteProvider, stopChan <-chan struct{}) *defaultCoreBasicAuth {
	if defaultCoreBasicAuthInstance == nil {
		defaultCoreBasicAuthInstance = &defaultCoreBasicAuth{
			PrivilegesProvider:      priProvider,
			UserProvider:            authProvider,
			userAuthenticationCache: &sync.Map{},
			routeProvider:           routeProvider,
			stopChan:                stopChan,
			whiteList:               make(map[string]struct{}),
		}
	}

	return defaultCoreBasicAuthInstance
}
