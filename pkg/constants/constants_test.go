package constants_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"                  // Ginkgo for test descriptions
	. "github.com/onsi/gomega"                     // Gomega for assertions
	"github.com/scality/cosi-driver/pkg/constants" // Import the constants package
)

func TestConstants(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Constants Suite")
}

var _ = Describe("Constants", func() {
	Context("Log level constants", func() {
		It("should have the correct values for log levels", func() {
			Expect(constants.LvlDefault).To(Equal(1), "LvlDefault should start at 1")
			Expect(constants.LvlInfo).To(Equal(2), "LvlInfo should have the value 2")
			Expect(constants.LvlEvent).To(Equal(3), "LvlEvent should have the value 3")
			Expect(constants.LvlDebug).To(Equal(4), "LvlDebug should have the value 4")
			Expect(constants.LvlTrace).To(Equal(5), "LvlTrace should have the value 5")
		})
	})
})
