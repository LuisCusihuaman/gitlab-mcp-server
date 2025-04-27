package main

import "fmt"

// Placeholders for version info injected by goreleaser/build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Println("GitLab MCP Server (minimal placeholder)")
	// We will replace this with actual cobra/viper setup later
}
