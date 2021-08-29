package command

import "testing"

func TestOutput(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *ExecuteTestCase
	}{
		{
			name: "output formats when interfaces provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ExecutorNode(func(o Output, d *Data) error {
					t := "there"
					o.Stdout("hello %s")
					o.Stdoutf("hello %s", t)

					k := "kenobi"
					o.Stderr("general %s")
					o.Stderrf("general %s", k)

					return nil
				})),
				WantStdout: []string{
					"hello %s",
					"hello there",
				},
				WantStderr: []string{
					"general %s",
					"general kenobi",
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			ExecuteTest(t, test.etc, nil)
		})
	}
}
