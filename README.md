![Baton Logo](./docs/images/baton-logo.png)

# `baton-calendly` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-calendly.svg)](https://pkg.go.dev/github.com/conductorone/baton-calendly) ![main ci](https://github.com/conductorone/baton-calendly/actions/workflows/main.yaml/badge.svg)

`baton-calendly` is a connector for Calendly built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the Calendly API, to sync data about Calendly organizations and users. 

Check out [Baton](https://github.com/conductorone/baton) to learn more about the project in general.

# Prerequisites

To be able to work with the connector, you need to have a Calendly account along with the Personal Access Token that can be created in the Calendly web platform. You can create one by going into `Integrations & apps` located in left side menu and then choosing the `API and webhooks` option from the list of different integrations and apps. Here you can create a new personal access token by clicking on the `Generate New Token` button.

For the connector to work, the user represented by the token must have admin permissions in the organization.

# Getting Started

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-calendly

BATON_TOKEN=token baton-calendly
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_TOKEN=token ghcr.io/conductorone/baton-calendly:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-calendly/cmd/baton-calendly@main

BATON_TOKEN=token baton-calendly
baton resources
```

# Data Model

`baton-calendly` will fetch information about the following Calendly resources:

- Organizations
- Users

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-calendly` Command Line Usage

```
baton-calendly

Usage:
  baton-calendly [flags]
  baton-calendly [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --client-id string       The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string   The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string            The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                   help for baton-calendly
      --log-format string      The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string       The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning           This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync         This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing              This must be set to enable ticketing support ($BATON_TICKETING)
      --token string           required: Personal Access Token used to authenticate with the Calendly API. ($BATON_TOKEN)
  -v, --version                version for baton-calendly

Use "baton-calendly [command] --help" for more information about a command.
```
