package mtest

import . "github.com/onsi/ginkgo"

// FunctionsSuite is a test suite that tests small test cases
var FunctionsSuite = func() {
	Context("cleanup", TestCleanup)
	Context("example", TestExample)
	Context("pod", TestPod)
	Context("BMC", TestBMC)
	Context("volume", TestVolume)
}
