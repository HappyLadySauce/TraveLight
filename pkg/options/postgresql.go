package options

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
)

const (
	DefaultDatabaseHost     = "localhost"
	DefaultDatabasePort     = 5432
	DefaultDatabaseUsername = "TraveLight"
	DefaultDatabasePassword = "TraveLight"
	DefaultDatabaseDatabase = "TraveLight"
	DefaultDatabaseTimeZone = "Asia/Shanghai"
)

type DatabaseOptions struct {
	Host     string `json:"host" mapstructure:"host"`
	Port     int    `json:"port" mapstructure:"port"`
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
	Database string `json:"database" mapstructure:"database"`
	TimeZone string `json:"timeZone" mapstructure:"timeZone"`
}

func NewDatabaseOptions() *DatabaseOptions {
	return &DatabaseOptions{
		Host:     DefaultDatabaseHost,
		Port:     DefaultDatabasePort,
		Username: DefaultDatabaseUsername,
		Password: DefaultDatabasePassword,
		Database: DefaultDatabaseDatabase,
		TimeZone: DefaultDatabaseTimeZone,
	}
}

func (o *DatabaseOptions) Validate() error {
	var errs []error
	if o.Host == "" {
		errs = append(errs, fmt.Errorf("host is empty"))
	}
	if o.Port <= 0 {
		errs = append(errs, fmt.Errorf("port is negative"))
	}
	if o.Username == "" {
		errs = append(errs, fmt.Errorf("username is empty"))
	}
	if o.Password == "" {
		errs = append(errs, fmt.Errorf("password is empty"))
	}
	if o.Database == "" {
		errs = append(errs, fmt.Errorf("database is empty"))
	}
	if o.TimeZone == "" {
		errs = append(errs, fmt.Errorf("timeZone is empty"))
	}
	return errors.Join(errs...)
}

func (o *DatabaseOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, "host", o.Host, "database host")
	fs.IntVar(&o.Port, "port", o.Port, "database port")
	fs.StringVar(&o.Username, "username", o.Username, "database username")
	fs.StringVar(&o.Password, "password", o.Password, "database password")
	fs.StringVar(&o.Database, "database", o.Database, "database name")
	fs.StringVar(&o.TimeZone, "timeZone", o.TimeZone, "database timeZone")
}
