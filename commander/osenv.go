package commander

import (
	"fmt"

	"github.com/leep-frog/command/command"
)

// EnvArg is a `command.Processor` that loads an environment variable's value into `command.Data`.
// The provided `name` is also used as the `command.Data` key. It results in an error
// if the environment variable is not set or if any of the validators fails.
type EnvArg struct {
	// Name is the name of the environment variable
	Name string
	// Optional indiciates whether a value is required to be set. An error will be
	// returned if this is false and no environment variable value exists.
	Optional bool
	// Validators are the validators to run on the on the environment variable's value. These are not
	// executed if `Optional` is true and the environment variable does not exist.
	// Note that these run after the `Transformers`
	Validators []*ValidatorOption[string]
	// Transformers are the transformations to run on the environment variable's value. These are not
	// executed if `Optional` is true and the environment variable does not exist.
	// Note that these run prior to the `Validators`
	Transformers []*Transformer[string]
	// DontRunOnComplete indicates whether or not this `command.Processor` will execute when running argument completion
	DontRunOnComplete bool
}

func (ea *EnvArg) Complete(i *command.Input, d *command.Data) (*command.Completion, error) {
	if ea.DontRunOnComplete {
		return nil, nil
	}
	return nil, ea.run(d)
}

func (ea *EnvArg) Usage(*command.Input, *command.Data, *command.Usage) error { return nil }

func (ea *EnvArg) Get(d *command.Data) string { return d.String(ea.Name) }

func (ea *EnvArg) Provided(d *command.Data) bool { return d.Has(ea.Name) }

func (ea *EnvArg) Execute(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
	return o.Err(ea.run(d))
}

func (ea *EnvArg) run(d *command.Data) error {
	fmt.Println("YUP")
	s, ok := command.OSLookupEnv(ea.Name)
	if !ok {
		if ea.Optional {
			return nil
		}
		return fmt.Errorf("Environment variable %s is not set", ea.Name)
	}

	for _, t := range ea.Transformers {
		newS, err := t.F(s, d)
		if err != nil {
			return fmt.Errorf("Environment variable transformation failed: %s", err)
		}
		s = newS
	}

	for _, v := range ea.Validators {
		if err := v.Validate(s, d); err != nil {
			return fmt.Errorf("Invalid value for environment variable %s: %s", ea.Name, err)
		}
	}
	d.Set(ea.Name, s)
	return nil
}

// SetEnvVarProcessor returns a `command.Processor` that sets the environment variable to the provided value.
func SetEnvVarProcessor(envVar, value string) command.Processor {
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.SetEnvVar(envVar, value))
		return nil
	}, nil)
}

// UnsetEnvVarProcessor returns a `command.Processor` that unsets the environment variable.
func UnsetEnvVarProcessor(envVar string) command.Processor {
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.UnsetEnvVar(envVar))
		return nil
	}, nil)
}
