package tests

import (
"testing"
. "github.com/puppetlabs/go-pspec/pspec"
)

func TestPSpecs(t *testing.T) {
	RunPspecTests(t, `testdata`)
}

