package apiserver

import (
	"core-api/cmd/core-api-server/app/config"
	customMiddleware "core-api/pkg/core/apiserver/custom_middleware"
	"core-api/pkg/core/auth"
	"core-api/pkg/core/privileges"
	"core-api/pkg/k8s"
	coreApiLog "core-api/pkg/logger"
	northApiRoute "core-api/pkg/north/api/route"
	"fmt"
	"net/http"
	"strings"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"
	"k8s.io/client-go/kubernetes"

	_ "core-api/docs"

	"github.com/common-nighthawk/go-figure"
	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func RunServer(serverConfig *config.Config) error {

	c := make(chan struct{}, 1)

	// Run the server
	var rootRoute *chi.Mux
	var rootRouteProvider northApiRoute.IRouteProvider
	rootRouteProvider = GetRootRouteProvider(serverConfig, c)

	// init k8s clientSet
	if serverConfig.KubeConfig.RestConfig != nil {
		k8sClientSet, err := kubernetes.NewForConfig(serverConfig.KubeConfig.RestConfig)
		if err != nil {
			coreApiLog.Logger.Error("Failed to create k8s clientSet", "error", err)
			return err
		}
		// init k8s helper
		err = k8s.InitK8sHelper(k8s.DefaultK8sHelperType, k8sClientSet, c)
		if err != nil {
			coreApiLog.Logger.Error("Failed to create k8s helper", "error", err)
			return err
		}

	} else {
		coreApiLog.Logger.Warn("KubeConfig is nil so k8s clientSet is not created")
	}

	if !serverConfig.CoreApiConfig.DisableAuth {
		stopCh := signals.SetupSignalHandler().Done()

		go func() {
			<-stopCh
			time.Sleep(2 * time.Second)
			close(c)
		}()
		// init user provider
		userProver, err := auth.CreateUserProvider(serverConfig, auth.KeystoneAuthProvider)
		if err != nil {
			coreApiLog.Logger.Error("Failed to create user provider", "error", err)
			return err
		}

		// init basic auth middleware
		basicAuthMiddleware, err := customMiddleware.NewCoreBaseAuth(&privileges.DefaultPrivilegeProvider{}, userProver, customMiddleware.DefaultCoreBaseAuth, rootRouteProvider, c)
		if err != nil {
			coreApiLog.Logger.Error("Failed to create basic auth middleware", "error", err)
			return err
		}

		basicAuthMiddleware.SetWhiteListedRoutes([]string{
			"/doc/*",
			"/apis/core-api.openhydra.io/v1/users/login",
			"/apis/core-api.openhydra.io/v1/licenses/{licenseId}",
			"/apis/core-api.openhydra.io/v1/licenses",
			"/apis/core-api.openhydra.io/v1/versions/{versionId}",
			"/apis/core-api.openhydra.io/v1/versions",
		})
		go basicAuthMiddleware.RunBackgroundCache()
		rootRouteProvider = GetRootRouteProvider(serverConfig, c, basicAuthMiddleware.BasicAuth)
	} else {
		rootRouteProvider = GetRootRouteProvider(serverConfig, c)
	}

	rootRoute = rootRouteProvider.GetRoot()

	go func() {
		fmt.Println(figure.NewColorFigure(strings.ToUpper("core-api"), "isometric1", "green", true).String())
		coreApiLog.Logger.Info("Attempting to mount doc route")
		rootRoute.Mount("/doc", httpSwagger.WrapHandler)
		coreApiLog.Logger.Info("Starting server", "port", serverConfig.CoreApiConfig.Port, "disable_auth", serverConfig.CoreApiConfig.DisableAuth)
		err := http.ListenAndServe(fmt.Sprintf(":%s", serverConfig.CoreApiConfig.Port), rootRoute)
		if err != nil {
			coreApiLog.Logger.Error("Failed to start server", "error", err)
			close(c)
		}
	}()

	<-c
	return nil
}
