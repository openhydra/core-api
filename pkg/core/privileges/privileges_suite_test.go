package privileges_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPrivileges(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Privileges Suite")
}
