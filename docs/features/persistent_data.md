# Persistent Data

Storing persistent data between CLI executions is also a critical feature of useful CLIs that this package supports. Basically, whenever a CLI is run, the `sourcerer.CLI.Changed()` function is used to determine if the CLI's underlying data changed. If it has, then the `CLI` is json serialized and stored in the directory pointed to by the `COMMAND_CLI_CACHE` environment variable.

## Setup

Simply create a directory in your file system and point the `COMMAND_CLI_CACHE` environment variable to it:

```bash
export COMMAND_CLI_CACHE=/path/to/your/directory
```
