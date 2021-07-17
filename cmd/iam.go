package cmd

import (
	"bitbucket.org/agrim123/onyx/pkg/core/iam"
	"bitbucket.org/agrim123/onyx/pkg/logger"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Returns the user making requests",
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
