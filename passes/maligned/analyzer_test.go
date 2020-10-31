package maligned_test

import (
	"testing"

	"github.com/mdempsky/maligned/passes/maligned"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestTest(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, maligned.Analyzer, "a")
}
