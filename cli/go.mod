module github.com/intruder0007/Cli/cli

go 1.22

require (
	github.com/intruder0007/Cli/core v0.1.0
	github.com/intruder0007/Cli/sdk/go v0.1.0
	golang.org/x/term v0.29.0
)

replace (
	github.com/intruder0007/Cli/core => ../core
	github.com/intruder0007/Cli/sdk/go => ../sdk/go
)
