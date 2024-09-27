package apiserver

import (
	"core-api/cmd/core-api-server/app/config"
	northApiRoute "core-api/pkg/north/api/route"
	"net/http"
)

func GetRootRouteProvider(serverConfig *config.Config, stopChan <-chan struct{}, middlewares ...func(http.Handler) http.Handler) northApiRoute.IRouteProvider {
	// Run the server
	routeBuilder := northApiRoute.NewDefaultRootRoute()
	// load all common middlewares
	routeBuilder.AddCommonMiddlewares()
	routeBuilder.AddGlobalMiddlewares(middlewares...)

	// now we build up route
	defaultRouteRegister := northApiRoute.NewDefaultRouteRegister(serverConfig, stopChan)
	// build all route here
	// you can define a struct to override the default route register
	defaultRouteRegister.RegisterRoute(routeBuilder)
	return routeBuilder
}
