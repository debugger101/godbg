package main

import (
	 . "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestAbsoulteFilenameError(t *testing.T) {
	g := NewGomegaWithT(t)

	oriArgs := os.Args
	defer func() {
		os.Args = oriArgs
	}()
	os.Args = []string{"godbg","be", "hello.go"}

	_, err := absoulteFilename()
	g.Expect(err).ShouldNot(Equal(nil))
}

func TestAbsoulteFilename(t *testing.T) {
	g := NewGomegaWithT(t)
	oriArgs := os.Args
	defer func() {
		os.Args = oriArgs
	}()

	os.Args = []string{"godbg","debug", "main_test.go"}

	_, err := absoulteFilename()
	g.Expect(err).Should(BeNil())
}