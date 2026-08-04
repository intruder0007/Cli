module github.com/intruder0007/Lumo/cli

go 1.22

require (
	github.com/intruder0007/Lumo/core v0.1.0
	github.com/intruder0007/Lumo/sdk/go v0.1.0
	golang.org/x/term v0.29.0
)

require golang.org/x/sys v0.30.0 // indirect

replace (
	github.com/intruder0007/Lumo/core => ../core
	github.com/intruder0007/Lumo/sdk/go => ../sdk/go
)
