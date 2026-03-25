package options

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
)

const (
	DefaultRedisHost = "localhost:6379"
	DefaultRedisPass = ""
	DefaultRedisDB   = 0
)

type RedisOptions struct {
	RedisHost string `json:"redisHost" mapstructure:"redisHost"`
	RedisPass string `json:"redisPass" mapstructure:"redisPass"`
	RedisDB   int    `json:"redisDB" mapstructure:"redisDB"`
}

func NewRedisOptions() *RedisOptions {
	return &RedisOptions{
		RedisHost: DefaultRedisHost,
		RedisPass: DefaultRedisPass,
		RedisDB:   DefaultRedisDB,
	}
}

func (o *RedisOptions) Validate() error {
	var errs []error
	if o.RedisHost == "" {
		errs = append(errs, fmt.Errorf("redis host is empty"))
	}
	if o.RedisDB < 0 {
		errs = append(errs, fmt.Errorf("redis db is negative"))
	}
	return errors.Join(errs...)
}

func (o *RedisOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.RedisHost, "redisHost", o.RedisHost, "redis host")
	fs.StringVar(&o.RedisPass, "redisPass", o.RedisPass, "redis password")
	fs.IntVar(&o.RedisDB, "redisDB", o.RedisDB, "redis db")
}
