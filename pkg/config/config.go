package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	Token = field.StringField(
		"token",
		field.WithDisplayName("Personal access token"),
		field.WithDescription("Personal Access Token used to authenticate with the Calendly API."),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)
	BaseURLField = field.StringField(
		"base-url",
		field.WithDescription("Override the Calendly API URL (for testing)"),
		field.WithHidden(true),
	)
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{Token, BaseURLField},
	field.WithConnectorDisplayName("Calendly"),
	field.WithHelpUrl("/docs/baton/calendly"),
	field.WithIconUrl("/static/app-icons/calendly.svg"),
)

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(cfg *Calendly) error {
	return nil
}
