package driver_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDriverSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driver Test Suite")
}
