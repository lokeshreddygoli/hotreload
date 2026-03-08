package cmd

import (
	"fmt"
	"os"

	"github.com/lokeshreddygoli/hotreload/internal/engine"
	"github.com/spf13/cobra"
)

var (
	rootDir  string
	buildCmd string
	execCmd  string
)

var rootCmd = &cobra.Command{
	Use:   "hotreload",
	Short: "Hot-reload CLI for Go projects",
	Long: `hotreload watches a project folder for code changes.
Whenever something changes it automatically rebuilds and restarts the server.

Example:
  hotreload --root ./myproject --build "go build -o ./bin/server ./cmd/server" --exec "./bin/server"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if rootDir == "" {
			return fmt.Errorf("--root is required")
		}
		if buildCmd == "" {
			return fmt.Errorf("--build is required")
		}
		if execCmd == "" {
			return fmt.Errorf("--exec is required")
		}

		e, err := engine.New(engine.Config{
			Root:     rootDir,
			BuildCmd: buildCmd,
			ExecCmd:  execCmd,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		return e.Run()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&rootDir, "root", "", "Directory to watch for file changes (required)")
	rootCmd.Flags().StringVar(&buildCmd, "build", "", "Command to build the project (required)")
	rootCmd.Flags().StringVar(&execCmd, "exec", "", "Command to run the built server (required)")
}
