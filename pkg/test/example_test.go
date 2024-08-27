package test

import (
	"testing"

	"sigs.k8s.io/e2e-framework/klient"
)

var ts *TestSuite

func TestMain(m *testing.M) {
	ts = NewTestSuite(m, "kimtests", "/Users/I539990/tmp/kind-kc.yaml")
	ts.Run()
}

func TestKim(t *testing.T) {
	tc := ts.NewTestCase(t, "my fance test case1")
	tc.Assert("Deploy KIM successfully", func(t *testing.T, k8sClient klient.Client) {
		t.Log("Hello - I'm now running1")
	})
	tc.Run()
}
