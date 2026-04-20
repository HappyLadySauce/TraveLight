package options

import (
	"encoding/json"

	"github.com/spf13/pflag"
	"k8s.io/component-base/cli/flag"

	"github.com/HappyLadySauce/TraveLight/pkg/options"
)

// Options holds the options for the application
// 应用选项
type Options struct {
	*options.ServerOptions   `mapstructure:"server"`
	*options.DatabaseOptions `mapstructure:"db"`
	*options.RedisOptions    `mapstructure:"redis"`
	*options.AuthOptions     `mapstructure:"auth"`
}

// NewOptions creates a new Options struct
// 创建一个新的选项结构体
func NewOptions() *Options {
	return &Options{
		ServerOptions:   options.NewServerOptions(),
		DatabaseOptions: options.NewDatabaseOptions(),
		RedisOptions:    options.NewRedisOptions(),
		AuthOptions:     options.NewAuthOptions(),
	}
}

// AddFlags adds the flags to the specified FlagSet and returns the grouped flag sets.
// 添加标志到指定的 FlagSet 并返回分组的标志集
func (o *Options) AddFlags(fs *pflag.FlagSet, basename string) *flag.NamedFlagSets {
	nfs := &flag.NamedFlagSets{}

	// add the flags to the NamedFlagSets
	// 添加标志到 NamedFlagSets 中
	configFS := nfs.FlagSet("Config")
	options.AddConfigFlag(configFS, basename)

	serverFS := nfs.FlagSet("Server")
	o.ServerOptions.AddFlags(serverFS)

	dbFS := nfs.FlagSet("PostgreSQL")
	o.DatabaseOptions.AddFlags(dbFS)

	redisFS := nfs.FlagSet("Redis")
	o.RedisOptions.AddFlags(redisFS)

	authFS := nfs.FlagSet("Auth")
	o.AuthOptions.AddFlags(authFS)

	// add the flags to the main Command
	// 添加标志到主命令
	for _, name := range nfs.Order {
		fs.AddFlagSet(nfs.FlagSets[name])
	}
	return nfs
}

// String returns the string representation of the Options struct
// 返回选项结构体的字符串表示
func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}
