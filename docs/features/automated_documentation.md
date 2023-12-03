# Automatic Usage Documentation

Writing usage docs for your CLIs can be tedious to write and maintain. This package automatically creates your usage doc for you based on your CLI's node structure.

The below types and functions are configured to make CLI usage docs as seamless as possible:

- `command.Arg` and `commander.Flag` functions require a `description` field which is used in the auto-generated usage doc.

- `command.BranchNode` automatically updates the usage doc to enumerate all possible options, and it's default node.
