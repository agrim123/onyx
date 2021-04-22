package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "onyx",
	Short: "A small command line utility to easily perform otherwise long tasks on AWS console.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(ecsCommand, ec2Command, whoamiCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
