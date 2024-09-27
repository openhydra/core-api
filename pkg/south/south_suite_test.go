package south_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSouth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "South Suite")
}
