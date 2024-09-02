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

func NewSuite(m *testing.M, cfg *envconf.Config, given ...Given) *Suite {
	e2eTest := &Suite{
		testEnv: env.NewWithConfig(cfg),
		m:       m,
	}

	//call Given functions to retreive TestSuiteRequirements
	clusterName := envconf.RandomName("kime2e", 16)
	testSuiteReqs := []*Requirement{}
	for _, givenFct := range given {
		testSuiteReqs = append(testSuiteReqs, givenFct(clusterName))
	}

	//sort the TestSuiteRequirements by their execution order
	sort.Slice(testSuiteReqs, func(i, j int) bool {
		return testSuiteReqs[i].order < testSuiteReqs[j].order
	})

	//add the TestSuiteRequirements to setup and finish hook of the test suite
	setupFcts := []env.Func{}
	finishFcts := []env.Func{}
	for _, testSuiteCond := range testSuiteReqs {
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

type Suite struct {
	testEnv          env.Environment
	m                *testing.M
	clusterName      string
	clusterNamespace string
}

func (s *Suite) NewFeature(t *testing.T, name string) *Feature {
	f := &Feature{
		feature:          features.New(name),
		testEnv:          s.testEnv,
		t:                t,
		clusterName:      s.clusterName,
		clusterNamespace: s.clusterNamespace,
	}
	return f
}

func (s *Suite) Run() {
	os.Exit(s.testEnv.Run(s.m))
}
