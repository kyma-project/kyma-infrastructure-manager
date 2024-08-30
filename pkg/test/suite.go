package test

import (
	"os"
	"sort"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func NewEnvConf(kubeConfigFile string) *envconf.Config {
	var cfg *envconf.Config
	if kubeConfigFile == "" {
		cfg, _ = envconf.NewFromFlags()
	} else {
		cfg = envconf.NewWithKubeConfig(kubeConfigFile)
	}
	return cfg
}

func NewTestSuite(m *testing.M, cfg *envconf.Config, given ...Given) *TestSuite {
	e2eTest := &TestSuite{
		testEnv: env.NewWithConfig(cfg),
		m:       m,
	}

	//call Given functions to retreive TestSuiteRequirements
	clusterName := envconf.RandomName("kime2e", 16)
	testSuiteConds := []*TestSuiteRequirement{}
	for _, givenFct := range given {
		testSuiteConds = append(testSuiteConds, givenFct(clusterName))
	}

	//sort the TestSuiteRequirements by their execution order
	sort.Slice(testSuiteConds, func(i, j int) bool {
		return testSuiteConds[i].order < testSuiteConds[j].order
	})

	//add the TestSuiteRequirements to setup and finish hook of the test suite
	setupFcts := []env.Func{}
	finishFcts := []env.Func{}
	for _, testSuiteCond := range testSuiteConds {
		if testSuiteCond.setup != nil {
			setupFcts = append(setupFcts, testSuiteCond.setup)
		}
		if testSuiteCond.finish != nil {
			finishFcts = append(finishFcts, testSuiteCond.finish)
		}
	}
	e2eTest.testEnv.Setup(setupFcts...)
	e2eTest.testEnv.Finish(finishFcts...)

	return e2eTest
}

type TestSuite struct {
	testEnv          env.Environment
	m                *testing.M
	clusterName      string
	clusterNamespace string
}

func (ts *TestSuite) NewTestCase(t *testing.T, name string) *TestCase {
	tc := &TestCase{
		feature:          features.New(name),
		testEnv:          ts.testEnv,
		t:                t,
		clusterName:      ts.clusterName,
		clusterNamespace: ts.clusterNamespace,
	}
	return tc
}

func (ts *TestSuite) Run() {
	os.Exit(ts.testEnv.Run(ts.m))
}
