package evaluator

import (
	"testing"
	"github.com/puppetlabs/go-pspec/pspec"
)

func TestPSpecs(t *testing.T) {
	pspec.RunPspecTests(t, `testdata`)
}
