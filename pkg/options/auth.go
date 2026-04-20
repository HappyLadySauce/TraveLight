package options

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

const (
	DefaultJWTSecret            = "change_me_in_production"
	DefaultAccessTokenTTLMinute = 30
	DefaultRefreshTokenTTLHour  = 168
)

// AuthOptions stores auth-related options.
// AuthOptions 存储认证相关配置。
type AuthOptions struct {
	JWTSecret             string `json:"jwtSecret" mapstructure:"jwtSecret"`
	AccessTokenTTLMinute  int    `json:"accessTokenTTLMinute" mapstructure:"accessTokenTTLMinute"`
	RefreshTokenTTLHour   int    `json:"refreshTokenTTLHour" mapstructure:"refreshTokenTTLHour"`
}

func NewAuthOptions() *AuthOptions {
	return &AuthOptions{
		JWTSecret:            DefaultJWTSecret,
		AccessTokenTTLMinute: DefaultAccessTokenTTLMinute,
		RefreshTokenTTLHour:  DefaultRefreshTokenTTLHour,
	}
}

func (o *AuthOptions) Validate() error {
	var errs []error
	if o.JWTSecret == "" {
		errs = append(errs, fmt.Errorf("jwtSecret is empty"))
	}
	if o.AccessTokenTTLMinute <= 0 {
		errs = append(errs, fmt.Errorf("accessTokenTTLMinute must be positive"))
	}
	if o.RefreshTokenTTLHour <= 0 {
		errs = append(errs, fmt.Errorf("refreshTokenTTLHour must be positive"))
	}
	return errors.Join(errs...)
}

func (o *AuthOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.JWTSecret, "jwtSecret", o.JWTSecret, "jwt secret")
	fs.IntVar(&o.AccessTokenTTLMinute, "accessTokenTTLMinute", o.AccessTokenTTLMinute, "access token ttl in minutes")
	fs.IntVar(&o.RefreshTokenTTLHour, "refreshTokenTTLHour", o.RefreshTokenTTLHour, "refresh token ttl in hours")
}

func (o *AuthOptions) AccessTokenTTL() time.Duration {
	return time.Duration(o.AccessTokenTTLMinute) * time.Minute
}

func (o *AuthOptions) RefreshTokenTTL() time.Duration {
	return time.Duration(o.RefreshTokenTTLHour) * time.Hour
}
