package options

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
)

const (
	DefaultBindAddress = "127.0.0.1"
	DefaultBindPort    = 8081
)

type ServerOptions struct {
	BindAddress string `json:"bindAddress" mapstructure:"bindAddress"`
	BindPort    int    `json:"bindPort"    mapstructure:"bindPort"`
}

func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		BindAddress: DefaultBindAddress,
		BindPort:    DefaultBindPort,
	}
}

func (i *ServerOptions) Validate() error {
	var errs []error
	if i.BindAddress == "" {
		errs = append(errs, fmt.Errorf("bindAddress is empty"))
	}
	if i.BindPort <= 0 || i.BindPort > 65535 {
		errs = append(errs, fmt.Errorf("bindPort is out of range"))
	}
	return errors.Join(errs...)
}

func (i *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&i.BindAddress, "bindAddress", "b", i.BindAddress, "IP address on which to serve the --port, set to 0.0.0.0 for all interfaces")
	fs.IntVarP(&i.BindPort, "bindPort", "p", i.BindPort, "port to listen to for incoming requests")
}
