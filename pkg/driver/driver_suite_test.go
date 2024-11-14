package driver_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCosiDev(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driver Test Suite")
}
