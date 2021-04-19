package cmd

import (
	"fmt"
	"os"

	"github.com/agrim123/onyx/pkg/iam"
	"github.com/agrim123/onyx/pkg/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "onyx",
	Short: "Onyx is a lightweight wrapper over aws sdk tweaked to our needs",
	Long:  ``,
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Returns the user making requests",
	Long:  ``,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := iam.Whoami()
		if err != nil {
			return err
		}

		logger.Info(name)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(ecsCommand, ec2Command, whoamiCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// logger.Info("All done.")
}
