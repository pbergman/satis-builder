package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/docker/docker/client"
	"github.com/pbergman/logger"
)

func getLogger() *logger.Logger {
	return logger.NewLogger("app", logger.NewWriterHandler(os.Stdout, logger.Debug, false))
}

func main() {

	var location string
	var logger = getLogger()
	var ctx = context.Background()

	flag.StringVar(&location, "config", "./config.yaml", "Config file location.")
	flag.Parse()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	cnf, usr, err := GetConfig(location)

	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	if err := BuildSatis(ctx, cli, usr, cnf, logger); err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	var handler = &Handler{
		cnf:    cnf,
		logger: logger,

		ctx: ctx,
		cli: cli,
		usr: usr,
	}

	logger.Debug("starting listening on " + cnf.Listen)

	if err := http.ListenAndServe(cnf.Listen, handler); err != nil {
		logger.Error(err)
		os.Exit(2)
	}
}
