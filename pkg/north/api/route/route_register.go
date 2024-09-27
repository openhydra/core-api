package route

import (
	"core-api/cmd/core-api-server/app/config"
)

type DefaultRouteRegister struct {
	config   *config.Config
	stopChan <-chan struct{}
}

func NewDefaultRouteRegister(config *config.Config, stopChan <-chan struct{}) *DefaultRouteRegister {
	return &DefaultRouteRegister{
		config:   config,
		stopChan: stopChan,
	}
}

// here where we register the route
func (r *DefaultRouteRegister) RegisterRoute(provider IRouteProvider) {
	provider.RegisterRoute(openhydraFullPathPrefix, GetOpenhydraRoute(r.config))
	provider.RegisterRoute(coreApiFullPathPrefix, GetCoreApiRoute(r.config))
	provider.RegisterRoute(rayLLMInferenceFullPathPrefix, GetRayLLMInferenceRoute(r.config))
	provider.RegisterRoute(ragFullPath, GetRagRoute(r.config, r.stopChan))
	provider.RegisterRoute(openhydraExtendFullPathPrefix, GetOpenhydraExtendRoute(r.config))
	provider.RegisterRoute(xInferenceFullPath, GetXInferenceRoute(r.config))
}
