package south

import (
	"bytes"
	"core-api/cmd/core-api-server/app/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"core-api/pkg/util/common"

	courseV1 "open-hydra-server-api/pkg/apis/open-hydra-api/course/core/v1"

	"github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

var testCourseList = courseV1.CourseList{
	Items: []courseV1.Course{
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "test",
			},
			Spec: courseV1.CourseSpec{
				CreatedBy:   "test",
				Description: "test",
				LastUpdate:  metaV1.Now(),
				Level:       1,
				SandboxName: "test",
				Size:        1,
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "test2",
			},
			Spec: courseV1.CourseSpec{
				CreatedBy:   "test2",
				Description: "test2",
				LastUpdate:  metaV1.Now(),
				Level:       2,
				SandboxName: "test2",
				Size:        2,
			},
		},
	},
}

var testRouter = func(ws *restful.WebService) {
	coursePath := fmt.Sprintf("/%s/%s/%s/%s", OpenHydraApiPrefix, CourseGVR.Group, CourseGVR.Version, CourseGVR.Resource)
	ws.Route(ws.GET(coursePath).To(func(request *restful.Request, response *restful.Response) {
		response.WriteAsJson(testCourseList)
	}))
	ws.Route(ws.POST(coursePath).To(func(request *restful.Request, response *restful.Response) {
		// parse request body to TestBodyStruct
		// read from request.Request.Body
		body := &courseV1.Course{}
		err := json.NewDecoder(request.Request.Body).Decode(body)
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
		response.WriteAsJson(body)
	}))
}

var _ = Describe("South Api test", func() {
	var restConfig *rest.Config
	var serverConfig *config.Config
	BeforeEach(func() {
		restConfig = &rest.Config{
			Host: "https://localhost:8080",
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
				CAFile:   "/tmp/ca.crt",
			},
			BearerToken: "fakeToken",
		}
		serverConfig = &config.Config{
			KubeConfig: &config.KubeConfig{
				RestConfig: restConfig,
			},
		}
	})

	Describe("RequestKubeApiserverWithServiceAccount test", func() {
		It("should be expected", func() {
			stopChan := make(chan struct{})
			go common.StartMockHttpsServer(8080, testRouter, stopChan)
			defer close(stopChan)
			time.Sleep(1 * time.Second)
			result, err := RequestKubeApiserverWithServiceAccount(serverConfig, CourseGVR, "", nil, http.MethodGet, nil)
			Expect(err).To(BeNil())
			var resultToCompare courseV1.CourseList
			err = json.Unmarshal(result, &resultToCompare)
			Expect(err).To(BeNil())
			Expect(len(resultToCompare.Items)).To(Equal(2))
			Expect(resultToCompare.Items[0].Spec.CreatedBy).To(Equal("test"))
			Expect(resultToCompare.Items[1].Spec.CreatedBy).To(Equal("test2"))
		})
		It("should be expected with post", func() {
			stopChan := make(chan struct{})
			go common.StartMockHttpsServer(8080, testRouter, stopChan)
			defer close(stopChan)
			time.Sleep(1 * time.Second)
			dataToPost := &courseV1.Course{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test",
				},
				Spec: courseV1.CourseSpec{
					CreatedBy:   "test",
					Description: "test",
					LastUpdate:  metaV1.Now(),
					Level:       1,
					SandboxName: "test",
					Size:        1,
				},
			}
			postBody, err := json.Marshal(dataToPost)
			// convert postBody to io.ReadCloser
			postBodyReader := io.NopCloser(bytes.NewReader(postBody))
			Expect(err).To(BeNil())
			result, err := RequestKubeApiserverWithServiceAccount(serverConfig, CourseGVR, "", postBodyReader, http.MethodPost, nil)
			Expect(err).To(BeNil())
			resultToCompare := &courseV1.Course{}
			err = json.Unmarshal(result, resultToCompare)
			Expect(err).To(BeNil())
			Expect(resultToCompare.Name).To(Equal(dataToPost.Name))
		})
	})

	Describe("RequestKubeApiserverWithServiceAccountAndParseToT test", func() {
		It("should be expected", func() {
			stopChan := make(chan struct{})
			go common.StartMockHttpsServer(8080, testRouter, stopChan)
			defer close(stopChan)
			time.Sleep(1 * time.Second)
			result, err := RequestKubeApiserverWithServiceAccountAndParseToT[courseV1.CourseList](serverConfig, CourseGVR, "", nil, http.MethodGet, nil)
			Expect(err).To(BeNil())
			Expect(len(result.Items)).To(Equal(2))
			Expect(result.Items[0].Spec.CreatedBy).To(Equal("test"))
			Expect(result.Items[1].Spec.CreatedBy).To(Equal("test2"))
		})
	})
})
