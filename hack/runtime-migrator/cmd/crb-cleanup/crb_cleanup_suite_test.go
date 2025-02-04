package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCrbCleanup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CrbCleanup Suite")
}
