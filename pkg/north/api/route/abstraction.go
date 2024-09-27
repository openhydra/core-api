package route

import (
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
)

type IRouteProvider interface {
	RegisterRoute(path string, routeBuilder *ChiRouteBuilder)
	AddCommonMiddlewares()
	AddGlobalMiddlewares(middlewares ...func(http.Handler) http.Handler)
	// path|method|permission
	// u no what i mean right?
	GetRouteAuthorization() map[string]map[string]ModuleAndPermission
	GetRoot() *chi.Mux
}

type IRouteRegister interface {
	RegisterRoute(provider IRouteProvider)
}

type RouteType string

const (
	DefaultRootRouteType RouteType = "default"
)

type ChiSubRouteBuilder struct {
	Method              string
	Pattern             string
	Handler             http.HandlerFunc
	ModuleAndPermission ModuleAndPermission
}

type ChiRouteBuilder struct {
	// if this func do not require any permission, set it to 0 or leave it blank
	PathPrefix     string
	MethodHandlers []ChiSubRouteBuilder
	With           []func(http.Handler) http.Handler
}

type ModuleAndPermission struct {
	Module     string
	Permission uint64
}

func (builder *ChiRouteBuilder) Build() (func(r chi.Router), map[string]map[string]ModuleAndPermission) {
	authorization := make(map[string]map[string]ModuleAndPermission)
	return func(r chi.Router) {
		for _, middlewares := range builder.With {
			r.Use(middlewares)
		}

		for _, methodHandler := range builder.MethodHandlers {
			r.MethodFunc(methodHandler.Method, methodHandler.Pattern, methodHandler.Handler)
			CombinePathPermission(builder, methodHandler, authorization)
		}
	}, authorization
}

func NewRouteProvider(routeType RouteType) IRouteProvider {
	switch routeType {
	case DefaultRootRouteType:
		return &DefaultRouteProvider{}
	default:
		return nil
	}
}

func CombinePathPermission(builder *ChiRouteBuilder, subBuilder ChiSubRouteBuilder, pathPermissionSet map[string]map[string]ModuleAndPermission) {
	reqUrl := path.Join(builder.PathPrefix, subBuilder.Pattern)
	if _, ok := pathPermissionSet[reqUrl]; !ok {
		pathPermissionSet[reqUrl] = make(map[string]ModuleAndPermission)
	}
	pathPermissionSet[reqUrl][subBuilder.Method] = subBuilder.ModuleAndPermission
}
