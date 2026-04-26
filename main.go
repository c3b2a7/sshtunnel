package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/c3b2a7/sshtunnel/internal/log"
)

var (
	configPath  string
	showVersion bool
	verbose     bool
)

func main() {
	flag.StringVar(&configPath, "config", "", "config file")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.BoolVar(&verbose, "verbose", false, "verbose mode")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s, %s\n", Version, BuildTime)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := log.NewLogger(os.Stderr, logLevelFromVerbosity(verbose))

	if err := run(log.WithLogger(ctx, logger)); err != nil {
		log.Errorf(ctx, "%s", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if configPath == "" {
		flag.Usage()
		return fmt.Errorf("config file is required")
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	client, err := DialSSH(config.SSH)
	if err != nil {
		return err
	}
	defer client.Close()

	for i, tunnel := range config.Tunnels {
		tunnelID := fmt.Sprintf("tunnel-%d", i+1)
		tunnelCtx := log.WithTunnelLogger(ctx, tunnelID)

		log.Infof(tunnelCtx, "starting %s: mode=%s localAddr=%s remoteAddr=%s", tunnelID, tunnel.Mode, tunnel.Local, tunnel.Remote)
		if err = ServeTunnel(tunnelCtx, client, tunnel); err != nil {
			return fmt.Errorf("failed to start %s: %v", tunnelID, err)
		}
		log.Infof(tunnelCtx, "started %s", tunnelID)
	}

	<-ctx.Done()
	log.Infof(ctx, "exiting")

	return nil
}

func logLevelFromVerbosity(verbose bool) slog.Level {
	if verbose {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
