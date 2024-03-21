package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"

	"github.com/conductorone/baton-calendly/pkg/connector"
)

var version = "dev"

func main() {
	ctx := context.Background()

	cfg := &config{}
	cmd, err := cli.NewCmd(ctx, "baton-calendly", cfg, validateConfig, getConnector)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version
	cmdFlags(cmd)

	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func prepareAuth(cfg *config) any {
	if cfg.Token != "" {
		return uhttp.NewBearerAuth(cfg.Token)
	}

	return &uhttp.NoAuth{}
}

func getClient(ctx context.Context, cfg *config, auth any) (*http.Client, error) {
	var (
		httpClient *http.Client
		err        error
	)
	if cfg.Token != "" {
		authBearer := auth.(*uhttp.BearerAuth)
		httpClient, err = authBearer.GetClient(ctx)
		if err != nil {
			return nil, err
		}

		return httpClient, nil
	}

	authCred := auth.(uhttp.AuthCredentials)
	httpClient, err = authCred.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	return httpClient, nil
}

func getConnector(ctx context.Context, cfg *config) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)
	auth := prepareAuth(cfg)
	httpClient, err := getClient(ctx, cfg, auth)
	if err != nil {
		return nil, err
	}

	cb, err := connector.New(ctx, httpClient)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	c, err := connectorbuilder.NewConnector(ctx, cb)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
