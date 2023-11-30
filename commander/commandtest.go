package commander

/*

// UsageTestCase is a test case object for testing command usage.
/*type UsageTestCase struct {
	// commondels.Node is the root `commondels.Node` of the command to test.
	commondels.Node commondels.Node
	// Args is the list of arguments provided to the command.
	Args []string
	// WantString is the expected usage output.
	WantString []string
	// WantErr is the error that should be returned.
	WantErr error
}*/

// UsageTest runs a test on command usage.
/*func UsageTest(t *testing.T, utc *UsageTestCase) {
	// TODO: Remove UsageTest in favor of `ExecuteTest` with `--help` flag set.
	t.Helper()

	if utc == nil {
		utc = &UsageTestCase{}
	}

	got, err := Use(utc.Node, ParseExecuteArgs(utc.Args))
	CmpError(t, fmt.Sprintf("Use(%v)", utc.Args), utc.WantErr, err)

	if err == nil {
		if diff := cmp.Diff(strings.Join(utc.WantString, "\n"), got.String()); diff != "" {
			t.Errorf("Use(%v) returned incorrect response (-want, +got):\n%s", utc.Args, diff)
		}
	}
}*/

// PrependSetupArg prepends the SetupArg node to the given node.
/*func PreprendSetupArg(n commondels.Node) commondels.Node {
	return SerialNodes(SetupArg, n)
}
*/
