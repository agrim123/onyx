package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "onyx",
	Short: "A small command line utility to easily perform otherwise long tasks on AWS console.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(ecsCommand, ec2Command, whoamiCmd, cloudwatchCommand, sandstormCommand)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
