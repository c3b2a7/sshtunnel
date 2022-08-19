package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/c3b2a7/sshtunnel/constant"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
)

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
		log.Fatalln(err)
	}

	client, err := Connect(app.Config)
	if err != nil {
		log.Fatalln(err)
	}
	for _, tunnel := range app.Config.Tunnels {
		t := tunnel
		go func() {
			err := t.Bridge(client)
			if err != nil {
				logf("failed to start tunnel: %v", err)
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
		} `json:"tunnels,omitempty"`
		Target     string `json:"target"`
		Username   string `json:"username"`
		PrivateKey string `json:"ssh-key"`
		Passphrase string `json:"passphrase"`
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return err
	}

	app.Config.Target, _ = NewEndpoint(c.Target)
	app.Config.Username = c.Username
	app.Config.Passphrase = c.Passphrase
	if c.PrivateKey != "" {
		app.Config.PrivateKey = readPrivateKey(c.PrivateKey)
	}
	var tunnels []Tunnel
	for _, tunnel := range c.Tunnels {
		l, err := NewEndpoint(tunnel.Local)
		r, err := NewEndpoint(tunnel.Remote)
		if err != nil {
			return err
		}
		tunnels = append(tunnels, Tunnel{
			Local:  l,
			Remote: r,
		})
	}
	app.Config.Tunnels = tunnels

	return nil
}

func readPrivateKey(file string) (bytes []byte) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(err)
		return
	}
	return
}
