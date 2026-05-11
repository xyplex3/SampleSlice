package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables populated via -ldflags at link time.
var (
	Version   = "0.0.0" // Semantic version string (e.g. "1.2.3")
	Commit    = "unknown" // Git commit SHA at build time
	BuildDate = "unknown" // RFC 3339 timestamp of the build
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:    "version",
	Short: "Print version information",
	Long:   `Print version information for SampleSlice.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("SampleSlice v%s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", BuildDate)
		fmt.Println("A command-line tool for slicing audio into MPC programs")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}