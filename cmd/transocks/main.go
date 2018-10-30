package main

import (
	"flag"
	"fmt"
	"net"
	"net/url"

	"github.com/BurntSushi/toml"
	cmd "github.com/m4i/well"
	"github.com/cybozu-go/log"
	"github.com/m4i/transocks"
)

type tomlConfig struct {
	Listen   string        `toml:"listen"`
	ProxyURL string        `toml:"proxy_url"`
	Log      cmd.LogConfig `toml:"log"`
}

const (
	defaultAddr = "localhost:1081"
)

var (
	configFile = flag.String("f", "/etc/transocks.toml",
		"TOML configuration file path")
)

func loadConfig() (*transocks.Config, error) {
	tc := &tomlConfig{
		Listen: defaultAddr,
	}
	md, err := toml.DecodeFile(*configFile, tc)
	if err != nil {
		return nil, err
	}
	if len(md.Undecoded()) > 0 {
		return nil, fmt.Errorf("undecoded key in TOML: %v", md.Undecoded())
	}

	c := transocks.NewConfig()
	c.Addr = tc.Listen

	u, err := url.Parse(tc.ProxyURL)
	if err != nil {
		return nil, err
	}
	c.ProxyURL = u

	err = tc.Log.Apply()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func serve(lns []net.Listener, c *transocks.Config) {
	s, err := transocks.NewServer(c)
	if err != nil {
		log.ErrorExit(err)
	}

	for _, ln := range lns {
		s.Serve(ln)
	}
	err = cmd.Wait()
	if err != nil && !cmd.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

func main() {
	flag.Parse()

	c, err := loadConfig()
	if err != nil {
		log.ErrorExit(err)
	}

	g := &cmd.Graceful{
		Listen: func() ([]net.Listener, error) {
			return transocks.Listeners(c)
		},
		Serve: func(lns []net.Listener) {
			serve(lns, c)
		},
	}
	g.Run()

	err = cmd.Wait()
	if err != nil && !cmd.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
