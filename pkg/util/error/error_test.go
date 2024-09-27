package error

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("error test", func() {
	Describe("IsNotFound test", func() {
		It("should return true if the error is not found", func() {
			err := NewNotFound(404, "not found")
			result := IsNotFound(err)
			Expect(result).To(BeTrue())
		})

		It("should return false if the error is not not found", func() {
			err := fmt.Errorf("not found")
			result := IsNotFound(err)
			Expect(result).To(BeFalse())
		})
	})
})
