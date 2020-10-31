package main_test

import (
	"testing"

	maligned "github.com/mdempsky/maligned"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestTest(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, maligned.Analyzer, "a")
}
