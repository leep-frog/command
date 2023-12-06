package commandtest

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
)

// Changeable is an interface for commands that can be changed.
// Note: this is really just using a function from the `sourcerer.CLI` interface.
type Changeable interface {
	// Changed returns whether or not the undelrying command object has changed.
	Changed() bool
}

// ExecuteTestCase is a test case object for testing command execution.
type ExecuteTestCase struct {
	// Node is the root `Node` of the command to test.
	Node command.Node
	// Args is the list of arguments provided to the command.
	Args []string
	// Env is the map of os environment variables to stub. If nil, this is not stubbed.
	Env map[string]string
	// OS is the OS to use for the test.
	OS command.OS

	// WantData is the `Data` object that should be constructed.
	WantData *command.Data
	// SkipDataCheck skips the check on `WantData`.
	SkipDataCheck bool
	// DataCmpOpts is the set of cmp.Options that should be used
	// when comparing data.
	DataCmpOpts cmp.Options
	// WantExecuteData is the `ExecuteData` object that should be constructed.
	WantExecuteData *command.ExecuteData
	// WantStdout is the data that should be sent to stdout.
	WantStdout string
	// WantStderr is the data that should be sent to stderr.
	WantStderr string
	// WantErr is the error that should be returned.
	WantErr error
	// WantPanic is the object that should be passed to panic (or nil if no panic expected).
	WantPanic interface{}

	// WantRunContents are the set of shell commands that should have been run.
	WantRunContents []*RunContents

	// RequiresSetup indicates whether or not the command requires setup
	RequiresSetup bool
	// SetupContents is the contents of the setup file provided to the command.
	SetupContents []string

	// RunResponses are the stubbed responses to return from exec.Cmd.Run.
	RunResponses []*FakeRun
}

func (etc *ExecuteTestCase) GetEnv() map[string]string {
	if etc.Env == nil {
		etc.Env = map[string]string{}
	}
	return etc.Env
}

// FakeRun is a fake shell command run.
type FakeRun struct {
	Stdout []string
	Stderr []string
	Err    error
	F      func(t *testing.T)
}

// CompleteTestCase is a test case object for testing command autocompletion.
type CompleteTestCase struct {
	// Node is the root `Node` of the command to test.
	Node command.Node
	// Args is the list of arguments provided to the command.
	// Remember that args requires a dummy command argument (e.g. "cmd ")
	// since `COMP_LINE` includes that.
	Args string
	// PassthroughArgs are the passthrough args provided to the command autocompletion.
	PassthroughArgs []string
	// Env is the map of os environment variables to stub. If nil, this is not stubbed.
	Env map[string]string
	// OS is the OS to use for the test.
	OS command.OS

	// Want is the expected `Autocompletion` object produced by the test.
	Want *command.Autocompletion
	// WantErr is the error that should be returned.
	WantErr error
	// WantData is the `Data` object that should be constructed.
	WantData *command.Data
	// SkipDataCheck skips the check on `WantData`.
	SkipDataCheck bool
	// DataCmpOpts is the set of cmp.Options that should be used
	// when comparing data.
	DataCmpOpts cmp.Options

	// RunResponses are the stubbed responses to return from exec.Cmd.Run.
	RunResponses []*FakeRun

	// WantRunContents are the set of shell commands that should have been run.
	WantRunContents []*RunContents
}

func (ctc *CompleteTestCase) GetEnv() map[string]string {
	if ctc.Env == nil {
		ctc.Env = map[string]string{}
	}
	return ctc.Env
}

type RunContents struct {
	Name          string
	Args          []string
	Dir           string
	StdinContents string
}

// FakeOS is a fake `command.OS` interface implementer that can be used for testing purposes.
type FakeOS struct{}

func (*FakeOS) SetEnvVar(variable, value string) string {
	return fmt.Sprintf("FAKE_SET[(variable=%s), (value=%s)]", variable, value)
}

func (*FakeOS) UnsetEnvVar(variable string) string {
	return fmt.Sprintf("FAKE_UNSET[(variable=%s)]", variable)
}
