package grpcfactory_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGRPCFactorySuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gRPC Factory Test Suite")
}
