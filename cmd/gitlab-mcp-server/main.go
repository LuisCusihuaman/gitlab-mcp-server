package main

import "github.com/vektra/mockery/v2/cmd"

// Build time variables - These are meant to be set via ldflags during the build process
// var (
// 	version = "dev"
// 	commit  = "none"
// 	date    = "unknown"
// )

func main() {
	// You can optionally pass the version info to the commands if needed
	// For example: cmd.Version = version
	cmd.Execute()
}
