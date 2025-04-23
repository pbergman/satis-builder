package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/pbergman/logger"
	"github.com/rs/xid"
)

func getLogger() *logger.Logger {
	return logger.NewLogger("app", logger.NewWriterHandler(os.Stdout, logger.LogLevelDebug(), false))
}

func main() {

	var location string
	var logger = getLogger()
	var queue = make(chan os.Signal, 1)

	signal.Notify(queue, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM)

	var ctx, cancel = context.WithCancel(context.Background())

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

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Debug("received signal, exiting")
				cancel()
				signal.Stop(queue)
				return
			case s := <-queue:
				switch s {
				case syscall.SIGTERM:
					logger.Debug("received signal SIGTERM")
					cancel()
					signal.Stop(queue)
					return
				case syscall.SIGUSR1:
					logger.Debug("received signal SIGUSR1")

					if err := BuildSatis(ctx, cli, usr, cnf, logger); err != nil {
						logger.Error(fmt.Sprintf("[%s] failed to build image: %s", xid.New(), err.Error()))
					} else {
						logger.Debug("successfully built satis")
					}

				case syscall.SIGUSR2:
					logger.Debug("received signal SIGUSR2")

					if err := PullImage(ctx, cli, cnf, logger); err != nil {
						logger.Error(fmt.Sprintf("[%s] failed to pull image: %s", xid.New(), err.Error()))
					} else {
						logger.Debug("successfully pulled satis")
					}
				}
			}
		}
	}()

	var handler = &Handler{
		cnf:    cnf,
		logger: logger,

		ctx: ctx,
		cli: cli,
		usr: usr,
	}

	logger.Debug("starting listening on " + cnf.Listen)

	srv := &http.Server{
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		Addr:    cnf.Listen,
		Handler: handler,
	}

	if err := srv.ListenAndServe(); err != nil {
		logger.Error(err)
		os.Exit(2)
	}
}
