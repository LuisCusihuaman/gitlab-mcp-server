package main

import (
	"context"    // Added for signal handling
	"fmt"        // Added for stdio logging
	stdlog "log" // Use standard log for initial fatal errors
	"os"
	"os/signal" // Added for signal handling
	"strings"   // Added for toolset parsing
	"syscall"   // Added for signal handling

	"github.com/LuisCusihuaman/gitlab-mcp-server/pkg/gitlab" // Reference pkg/gitlab
	// Reference pkg/toolsets
	// iolog "github.com/github/github-mcp-server/pkg/log" // TODO: Consider adding if command logging is needed
	"github.com/mark3labs/mcp-go/server" // MCP server components
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gl "gitlab.com/gitlab-org/api/client-go" // Alias for GitLab client library
	// MCP types
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

			// --- Subtask 6.3: Main Execution Flow ---
			logger.Info("Starting main execution flow...")

			// Read configuration
			token := viper.GetString("token")
			if token == "" {
				logger.Fatal("Required configuration missing: GITLAB_TOKEN (or --gitlab-token) must be set.")
			}
			host := viper.GetString("host") // Optional, defaults handled by NewClient
			readOnly := viper.GetBool("read-only")

			// Special handling for toolsets slice from env var
			var enabledToolsets []string
			toolsetsStr := viper.GetString("toolsets") // Get as string first
			if toolsetsStr != "" {
				enabledToolsets = strings.Split(toolsetsStr, ",")
			} else {
				// Fallback or default if necessary, viper should handle defaults from flags though
				enabledToolsets = gitlab.DefaultTools
				logger.Infof("No toolsets specified via config/env, using default: %v", enabledToolsets)
			}
			// Alternative using UnmarshalKey (might be cleaner if defaults work correctly)
			// err = viper.UnmarshalKey("toolsets", &enabledToolsets)
			// if err != nil {
			// 	logger.Fatalf("Failed to unmarshal toolsets: %v", err)
			// }
			logger.Infof("Enabled toolsets: %v", enabledToolsets)
			logger.Infof("Read-only mode: %t", readOnly)
			if host != "" {
				logger.Infof("Using custom GitLab host: %s", host)
			}

			// Initialize GitLab Client directly
			clientOpts := []gl.ClientOptionFunc{}
			if host != "" {
				clientOpts = append(clientOpts, gl.WithBaseURL(host))
			}
			glClient, err := gl.NewClient(token, clientOpts...)
			if err != nil {
				logger.Fatalf("Failed to initialize GitLab client: %v", err)
			}
			logger.Info("GitLab client initialized")

			// Define GetClientFn closure again
			getClient := func(_ context.Context) (*gl.Client, error) {
				return glClient, nil // Provide the initialized client via closure
			}

			// TODO: Initialize Translations (deferring for now)
			// t, dumpTranslations := translations.TranslationHelper()

			// Initialize Toolsets, passing the getClient function
			toolsetGroup, err := gitlab.InitToolsets(enabledToolsets, readOnly, getClient /*, t */)
			if err != nil {
				logger.Fatalf("Failed to initialize toolsets: %v", err)
			}
			logger.Info("Toolsets initialized")

			// Create MCP Server
			// Use app name and version
			mcpServer := gitlab.NewServer("gitlab-mcp-server", version)
			logger.Info("MCP server wrapper created")

			// Register Toolsets with the server (does not return error)
			toolsetGroup.RegisterTools(mcpServer)
			logger.Info("Toolsets registered with MCP server")

			// Create Stdio Server
			stdioServer := server.NewStdioServer(mcpServer)
			// Configure logger for the stdio transport layer
			stdioLogger := stdlog.New(logger.Writer(), "[StdioServer] ", 0) // Use logger's writer
			stdioServer.SetErrorLogger(stdioLogger)
			logger.Info("Stdio server transport created")

			// Start Listening in a goroutine
			errC := make(chan error, 1)
			go func() {
				logger.Info("Starting to listen on stdio...")
				// TODO: Add command logging wrapper if flag is enabled
				// in, out := io.Reader(os.Stdin), io.Writer(os.Stdout)
				// if viper.GetBool("enable-command-logging") {
				// 	 loggedIO := iolog.NewIOLogger(in, out, logger)
				// 	 in, out = loggedIO, loggedIO
				// }
				errC <- stdioServer.Listen(ctx, os.Stdin, os.Stdout)
			}()

			// Announce readiness on stderr
			fmt.Fprintf(os.Stderr, "GitLab MCP Server running on stdio (Version: %s, Commit: %s)\n", version, commit)
			logger.Info("Server running, waiting for requests or signals...")

			// Wait for shutdown signal or server error
			select {
			case <-ctx.Done(): // Triggered by signal
				logger.Info("Shutdown signal received, context cancelled.")
			case err := <-errC: // Triggered by server.Listen returning an error
				if err != nil && err != context.Canceled {
					logger.Errorf("Server encountered an error: %v", err)
					// We might want os.Exit(1) here depending on desired behavior
				} else {
					logger.Info("Server listener stopped gracefully.")
				}
			}

			logger.Info("Server shutting down.")
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
