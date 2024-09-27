package route

import (
	"core-api/cmd/core-api-server/app/config"
	coreApiLog "core-api/pkg/logger"
	"core-api/pkg/util/common"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
)

// note for route unit test all we test here is only the route reachability
// so we do not detail return value of the handler
func CreateOpenhydraRouteReachabilityTest(config *config.Config) *ChiRouteBuilder {
	OpenhydraRoute := GetOpenhydraRoute(config)
	for index, methodHandler := range OpenhydraRoute.MethodHandlers {
		OpenhydraRoute.MethodHandlers[index].Handler = func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(fmt.Sprintf("%s/%s/%s%s with method %s says hi", openhydraPrefix, openhydraGroup, openhydraAPIVersion, methodHandler.Pattern, methodHandler.Method)))
		}
	}
	return OpenhydraRoute
}

var _ = Describe("Route", func() {
	var serverConfig *config.Config
	var r *ChiRouteBuilder
	openhydraRoutePrefix := fmt.Sprintf("%s/%s/%s", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
	var builder IRouteProvider
	// var commonHeaders = map[string]string{
	// 	"Content-Type": "application/json",
	// }
	BeforeEach(func() {
		serverConfig = config.DefaultConfig()
		serverConfig.KubeConfig = &config.KubeConfig{
			RestConfig: &rest.Config{
				Host: "https://localhost:8080",
			},
		}
		r = CreateOpenhydraRouteReachabilityTest(serverConfig)
		builder = NewDefaultRootRoute()
	})

	Describe("RouteRegister test", func() {
		BeforeEach(func() {
			coreApiLog.InitLogger("DEBUG")
		})
		It("should return a expected result with openhydra root", func() {
			provider := NewDefaultRouteRegister(serverConfig, nil)
			provider.RegisterRoute(builder)
			root := builder.GetRoot()
			Expect(len(root.Routes())).To(Equal(6))
		})
	})

	Describe("OpenhydraRoute", func() {
		It("should return a expected result with openhydra root", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, r)
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodGet, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/ with method GET says hi"))
		})
		It("should return a expected result with openhydra devices", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/devices", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodGet, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/devices with method GET says hi"))
		})
		It("should return a expected result with openhydra devices post", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/devices", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodPost, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/devices with method POST says hi"))
		})
		It("should return a expected result with openhydra devices/{userId}", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/devices/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodGet, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/devices/{userId} with method GET says hi"))
		})
		It("should return a expected result with openhydra devices/{id} put", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/devices/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			_, _, code, err := common.CommonRequest(reqUrl, http.MethodPut, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusMethodNotAllowed))
			//Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/devices/{userId} with method PUT says hi"))
		})
		It("should return a expected result with openhydra courses", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/courses", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodGet, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/courses with method GET says hi"))
		})
		It("should return a expected result with openhydra courses post", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/courses", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodPost, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/courses with method POST says hi"))
		})
		It("should return a expected result with openhydra courses/{id}", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/courses/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodGet, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/courses/{courseId} with method GET says hi"))
		})
		It("should return a expected result with openhydra courses/{id} put", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/courses/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			_, _, code, err := common.CommonRequest(reqUrl, http.MethodPut, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusMethodNotAllowed))
			//Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/courses/{courseId} with method PUT says hi"))
		})
		It("should return a expected result with openhydra courses/{id} delete", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/courses/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodDelete, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/courses/{courseId} with method DELETE says hi"))
		})
		It("should return a expected result with openhydra datasets", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/datasets", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodGet, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/datasets with method GET says hi"))
		})
		It("should return a expected result with openhydra datasets post", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/datasets", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodPost, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/datasets with method POST says hi"))
		})
		It("should return a expected result with openhydra datasets/{id}", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/datasets/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodGet, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/datasets/{datasetId} with method GET says hi"))
		})
		It("should return a expected result with openhydra datasets/{id} put", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/datasets/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			_, _, code, err := common.CommonRequest(reqUrl, http.MethodPut, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusMethodNotAllowed))
			//Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/datasets/{id} with method PUT says hi"))
		})
		It("should return a expected result with openhydra datasets/{id} delete", func() {
			provider := NewDefaultRootRoute()
			url := fmt.Sprintf("%s/%s/%s/datasets/123", openhydraPrefix, openhydraGroup, openhydraAPIVersion)
			provider.RegisterRoute(openhydraRoutePrefix, CreateOpenhydraRouteReachabilityTest(serverConfig))
			root := provider.GetRoot()
			Expect(root).NotTo(BeNil())
			go http.ListenAndServe(":3000", root)
			time.Sleep(1 * time.Second)
			reqUrl := fmt.Sprintf("http://localhost:3000%s", url)
			body, _, code, err := common.CommonRequest(reqUrl, http.MethodDelete, "", nil, nil, false, false, 3*time.Second)
			Expect(err).To(BeNil())
			Expect(code).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("/apis/open-hydra-server.openhydra.io/v1/datasets/{datasetId} with method DELETE says hi"))
		})
	})
})
