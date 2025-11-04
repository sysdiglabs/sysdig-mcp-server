package sysdig_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSysdig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sysdig Client Integration Suite")
}
