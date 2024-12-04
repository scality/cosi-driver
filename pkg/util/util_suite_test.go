package util_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGRPCFactorySuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utilities Test Suite")
}
