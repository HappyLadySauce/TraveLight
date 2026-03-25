package options

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/HappyLadySauce/TraveLight/pkg/utils/homedir"
)

const (
	configFlagName = "config"
)

var cfgFile string

func init() {
	pflag.StringVarP(&cfgFile, "config", "f", cfgFile, "Read configuration from specified `FILE`, "+
		"support JSON, TOML, YAML, HCL, or Java properties formats.")
}

// addConfigFlag adds flags for a specific server to the specified FlagSet
// object.
func AddConfigFlag(fs *pflag.FlagSet, basename string) {
	fs.AddFlag(pflag.Lookup(configFlagName))

	viper.AutomaticEnv()
	viper.SetEnvPrefix(strings.Replace(strings.ToUpper(basename), "-", "_", -1))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	cobra.OnInitialize(func() {
		if cfgFile != "" {
			// Support ${ENV_VAR} expansion inside config files.
			// This enables passing config values via environment variables (e.g. from make).
			b, err := os.ReadFile(cfgFile)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: failed to read configuration file(%s): %v\n", cfgFile, err)
				os.Exit(1)
			}

			expanded := os.ExpandEnv(string(b))
			ext := strings.TrimPrefix(filepath.Ext(cfgFile), ".")
			if ext != "" {
				viper.SetConfigType(ext)
			}
			if err := viper.ReadConfig(strings.NewReader(expanded)); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: failed to read configuration file(%s): %v\n", cfgFile, err)
				os.Exit(1)
			}
			return
		} else {
			viper.AddConfigPath(".")

			if names := strings.Split(basename, "-"); len(names) > 1 {
				viper.AddConfigPath(filepath.Join(homedir.HomeDir(), "."+names[0]))
				viper.AddConfigPath(filepath.Join("/etc", names[0]))
			}

			viper.SetConfigName(basename)
		}

		if err := viper.ReadInConfig(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: failed to read configuration file(%s): %v\n", cfgFile, err)
			os.Exit(1)
		}
	})
}
