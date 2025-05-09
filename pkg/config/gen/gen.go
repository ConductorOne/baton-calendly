package main

import (
	cfg "github.com/conductorone/baton-calendly/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("calendly", cfg.Config)
}
