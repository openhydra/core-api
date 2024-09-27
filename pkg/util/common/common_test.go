package common

import (
	"os"
	"time"

	"github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testRouter = func(ws *restful.WebService) {
	ws.Route(ws.GET("/").To(func(request *restful.Request, response *restful.Response) {
		response.Write([]byte("welcome to core-api here you can build you own api"))
	}))
}

var _ = Describe("common help func test", func() {
	Describe("GetStringValueOrDefault test", func() {
		It("should return targetValue if targetValue is not empty", func() {
			result := GetStringValueOrDefault("targetValue", "defaultValue")
			Expect(result).To(Equal("targetValue"))
		})

		It("should return defaultValue if targetValue is empty", func() {
			result := GetStringValueOrDefault("", "defaultValue")
			Expect(result).To(Equal("defaultValue"))
		})
	})
	Describe("CommonRequest test", func() {
		It("should return an error if http.NewRequest failed", func() {
			stopChan := make(chan struct{})
			go StartMockServer(8090, testRouter, stopChan)
			defer close(stopChan)
			time.Sleep(1 * time.Second)
			_, _, _, err := CommonRequest("http://localhost:8090", "GET", "", nil, nil, false, false, 0)
			Expect(err).To(BeNil())
		})
	})

	Describe("CreateDirIfNotExists test", func() {
		It("should return nil if the directory is created successfully", func() {
			err := CreateDirIfNotExists("/tmp/test")
			Expect(err).To(BeNil())
			// check directory exists
			_, err = os.Stat("/tmp/test")
			Expect(err).To(BeNil())
			// remove directory
			err = os.Remove("/tmp/test")
			Expect(err).To(BeNil())
		})
	})

	Describe("BuildPath test", func() {
		It("should return the correct path", func() {
			result := BuildPath("http://localhost", "test")
			Expect(result).To(Equal("http://localhost/test"))
		})
	})

	Describe("Base64Decode test", func() {
		It("should return the correct decoded string", func() {
			result, err := Base64Decode("dGVzdA==")
			Expect(err).To(BeNil())
			Expect(string(result)).To(Equal("test"))
		})
	})
})
