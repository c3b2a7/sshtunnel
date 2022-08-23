package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/c3b2a7/sshtunnel/constant"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

var app struct {
	Verbose bool
	Config  Config
	Tunnels []Tunnel
}

var flags struct {
	config  string
	version bool
}

func init() {
	flag.BoolVar(&app.Verbose, "verbose", false, "verbose mode")
	flag.StringVar(&flags.config, "config", "", "config file")
	flag.BoolVar(&flags.version, "v", false, "show version information")
	flag.Parse()

	if flags.version {
		fmt.Printf("%s, %s\n", constant.Version, constant.BuildTime)
		os.Exit(0)
	}

	if flags.config == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	if err := initConfig(flags.config); err != nil {
		logger.Fatalln(err)
	}

	client, err := Connect(app.Config)
	if err != nil {
		logger.Fatalln(err)
	}
	for _, tunnel := range app.Tunnels {
		t := tunnel
		go func() {
			err := t.Bridge(client)
			if err != nil {
				logger.Printf("failed to start tunnel: %v", err)
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func initConfig(config string) error {
	data, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}

	var c struct {
		Tunnels []struct {
			Remote string `json:"remote"`
			Local  string `json:"local"`
			Mode   string `json:"mode"`
		} `json:"tunnels,omitempty"`
		Target     string `json:"target"`
		Username   string `json:"username"`
		PrivateKey string `json:"private-key"`
		Passphrase string `json:"passphrase"`
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return err
	}

	app.Config.Target, err = NewEndpoint(c.Target)
	if err != nil {
		return err
	}
	app.Config.Username = c.Username
	app.Config.Passphrase = c.Passphrase
	if c.PrivateKey != "" {
		app.Config.PrivateKey, err = ioutil.ReadFile(c.PrivateKey)
		if err != nil {
			return err
		}
	}
	var tunnels []Tunnel
	for _, tunnel := range c.Tunnels {
		var local, remote Endpoint
		local, err = NewEndpoint(tunnel.Local)
		if err != nil {
			return err
		}
		if tunnel.Remote != "" {
			remote, err = NewEndpoint(tunnel.Remote)
			if err != nil {
				return err
			}
		}
		var mode Mode
		mode, err = PickMode(tunnel.Mode)
		if err != nil {
			return err
		}
		tunnels = append(tunnels, Tunnel{
			Local:  local,
			Remote: remote,
			Mode:   mode,
		})
	}
	app.Tunnels = tunnels

	return nil
}
