package osperrors_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOSPErrors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OSP Errors Suite")
}
