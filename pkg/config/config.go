package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var Token = field.StringField(
	"token",
	field.WithDescription("Personal Access Token used to authenticate with the Calendly API."),
	field.WithRequired(true),
)

//go:generate go run ./gen
var Config = field.NewConfiguration([]field.SchemaField{
	Token,
})

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(cfg *Calendly) error {
	return nil
}
