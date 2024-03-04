package main

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/spf13/cobra"
)

// config defines the external configuration required for the connector to run.
type config struct {
	cli.BaseConfig `mapstructure:",squash"` // Puts the base config options in the same place as the connector options

	Token string `mapstructure:"token"`
}

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *config) error {
	if cfg.Token == "" {
		return fmt.Errorf("token is required, use --help for more information")
	}

	return nil
}

func cmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("token", "", "Personal Access Token used to authenticate with the Calendly API. ($BATON_TOKEN)")
}
