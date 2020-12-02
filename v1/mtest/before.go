package mtest

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"
)

// RunBeforeSuite is for Ginkgo BeforeSuite
func RunBeforeSuite() {
	fmt.Println("Preparing...")
	SetDefaultEventuallyPollingInterval(5 * time.Second)
	SetDefaultEventuallyTimeout(60 * time.Second)
	fmt.Println("Begin tests...")
}
