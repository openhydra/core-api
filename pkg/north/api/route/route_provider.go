package route

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type DefaultRouteProvider struct {
	root          *chi.Mux
	PermissionMap map[string]map[string]ModuleAndPermission
}

func NewDefaultRootRoute() *DefaultRouteProvider {
	provider := &DefaultRouteProvider{
		root: chi.NewRouter(),
	}
	return provider
}

func (r *DefaultRouteProvider) RegisterRoute(path string, routeBuilder *ChiRouteBuilder) {
	route, permission := routeBuilder.Build()
	r.root.Route(path, route)
	if r.PermissionMap == nil {
		r.PermissionMap = permission
	} else {
		// merge the permission
		for k, v := range permission {
			r.PermissionMap[k] = v
		}
	}
}

func (r *DefaultRouteProvider) AddCommonMiddlewares() {
	r.root.Use(middleware.RequestID)
	r.root.Use(middleware.RealIP)
	r.root.Use(middleware.Logger)
	r.root.Use(middleware.Recoverer)
	r.root.Use(middleware.SetHeader("Content-Type", "application/json"))
}

func (r *DefaultRouteProvider) AddGlobalMiddlewares(middlewares ...func(http.Handler) http.Handler) {
	r.root.Use(middlewares...)
}

func (r *DefaultRouteProvider) GetRouteAuthorization() map[string]map[string]ModuleAndPermission {
	return r.PermissionMap
}

func (r *DefaultRouteProvider) GetRoot() *chi.Mux {
	return r.root
}
