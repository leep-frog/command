# Testing

It is tremendously difficult to maintain complex code without testing. This package supports testing all CLI use cases:

- `ExecuteTest(*testing.T, *UsageTestCase)` tests command execution logic.

- `AutocompleteTest(*testing.T, *UsageTestCase)` tests command autocompletion logic.

- `UsageTest(*testing.T, *UsageTestCase)` tests CLI usage docs.

The following testing format is recommended for all test types:

```go

// The CLI you want to test that implement `sourcerer.CLI`.
type myCLI struct {...}

func TestExecution(t *testing.T) {
	for _, test := range []struct {
		name string
		cli    *myCLI
		etc  *commandtest.ExecuteTestCase
		wantCLI *myCLI
	}{
    {
      name: "simple execution works",
      etc: &commandtest.ExecuteTestCase{
        Args: []string{"arg1", "arg2", /* ... */}
      },
    },
    /* Include more test cases */
  } {
    t.Run(test.name, func(t *testing.T) {
      // Test the command's execution.
			test.etc.Node = test.cli.Node()
			command.ExecuteTest(t, test.etc)

      // Test if the command changed
			command.ChangeTest(t, test.wantCLI, test.cli, cmp.AllowUnexported(myCLI{}))
		})
  }
}
```
