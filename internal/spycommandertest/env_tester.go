package spycommandertest

import (
	"testing"

	"github.com/leep-frog/command/internal/stubs"
)

type envTester struct{}

func (et *envTester) setup(t *testing.T, tc *testContext) {
	stubs.StubEnv(t, tc.testCase.getEnv())
}

func (et *envTester) check(t *testing.T, tc *testContext) {}
