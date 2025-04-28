package main

import (
	"context" // Added for signal handling
	"fmt"
	stdlog "log" // Use standard log for initial fatal errors
	"os"
	"os/signal" // Added for signal handling
	"syscall"   // Added for signal handling

	"github.com/LuisCusihuaman/gitlab-mcp-server/pkg/gitlab" // Reference pkg/gitlab for DefaultTools
	log "github.com/sirupsen/logrus"                         // Using logrus for structured logging
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Injected by goreleaser
var version = "dev"
var commit = "none"
var date = "unknown"

var (
	rootCmd = &cobra.Command{
		Use:     "gitlab-mcp-server",
		Short:   "GitLab MCP Server",
		Long:    `A GitLab MCP server that provides tools for interacting with GitLab resources via the Model Context Protocol.`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Start server communicating via standard input/output",
		Long:  `Starts the GitLab MCP server, listening for JSON-RPC messages on stdin and sending responses to stdout.`,
		Run: func(_ *cobra.Command, _ []string) {
			// --- Subtask 6.2: Initialize Logger ---
			logLevel := viper.GetString("log.level")
			logFile := viper.GetString("log.file")
			logger, err := initLogger(logLevel, logFile)
			if err != nil {
				stdlog.Fatalf("Failed to initialize logger: %v", err) // Use stdlog before logger is ready
			}
			logger.Info("Logger initialized")

			// --- Subtask 6.2: Initialize Signal Handling ---
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop() // Ensure stop is called to release resources
			logger.Info("Signal handling initialized")

			// Placeholder for server execution logic (Subtask 6.3)
			logger.Info("Stdio server starting execution...")
			// TODO: Read config from Viper (done partially for logger)
			// TODO: Initialize GitLab client
			// TODO: Initialize Toolsets
			// TODO: Create MCP Server
			// TODO: Register Toolsets
			// TODO: Create Stdio Server
			// TODO: Start listening
			// TODO: Wait for context cancellation or error
			logger.Info("Placeholder: Server execution logic goes here.")

			<-ctx.Done() // Wait for signal
			logger.Info("Shutdown signal received, terminating server...")
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Set version template
	rootCmd.SetVersionTemplate("{{.Short}}\n{{.Version}}\n")

	// Define persistent flags for the root command (and inherited by subcommands)
	rootCmd.PersistentFlags().StringSlice("toolsets", gitlab.DefaultTools, "Comma-separated list of toolsets to enable (e.g., 'projects,issues' or 'all')")
	rootCmd.PersistentFlags().Bool("read-only", false, "Restrict the server to read-only operations")
	rootCmd.PersistentFlags().String("gitlab-host", "", "Optional: Specify the GitLab hostname for self-managed instances (e.g., gitlab.example.com)")
	rootCmd.PersistentFlags().String("gitlab-token", "", "GitLab Personal Access Token (required)")
	rootCmd.PersistentFlags().String("log-file", "", "Optional: Path to write log output to a file")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (e.g., debug, info, warn, error)")
	// TODO: Add optional flags like --enable-command-logging if needed later

	// Bind persistent flags to Viper
	// Note the mapping from flag name (kebab-case) to viper key (often snake_case or kept kebab-case) and ENV var (UPPER_SNAKE_CASE)
	_ = viper.BindPFlag("toolsets", rootCmd.PersistentFlags().Lookup("toolsets"))
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("gitlab-host"))    // Viper key "host" -> GITLAB_HOST
	_ = viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("gitlab-token"))  // Viper key "token" -> GITLAB_TOKEN
	_ = viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("log-file"))   // Viper key "log.file" -> GITLAB_LOG_FILE
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level")) // Viper key "log.level" -> GITLAB_LOG_LEVEL

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Set ENV var prefix
	viper.SetEnvPrefix("GITLAB")
	// Read in environment variables that match defined flags/keys
	viper.AutomaticEnv()

	// Optional: Configure reading from a config file
	// viper.SetConfigName("config") // name of config file (without extension)
	// viper.AddConfigPath(".")      // optionally look for config in the working directory
	// viper.AddConfigPath("$HOME/.gitlab-mcp-server") // call multiple times to add search paths
	// If a config file is found, read it in.
	// if err := viper.ReadInConfig(); err == nil {
	//  fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	// }
}

// initLogger sets up the logrus logger based on configuration.
func initLogger(level string, filePath string) (*log.Logger, error) {
	logger := log.New()

	// Set Log Level
	lvl, err := log.ParseLevel(level)
	if err != nil {
		logger.Warnf("Invalid log level '%s', defaulting to 'info': %v", level, err)
		lvl = log.InfoLevel
	}
	logger.SetLevel(lvl)

	// Set Output
	if filePath != "" {
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file '%s': %w", filePath, err)
		}
		logger.SetOutput(file)
		// Optional: Also log to stderr if logging to file?
		// logger.SetOutput(io.MultiWriter(os.Stderr, file))
	} else {
		logger.SetOutput(os.Stderr)
	}

	// Set Formatter (using TextFormatter for now)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	return logger, nil
}

func main() {
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
