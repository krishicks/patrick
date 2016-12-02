package patrick_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPatrick(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Patrick Suite")
}
