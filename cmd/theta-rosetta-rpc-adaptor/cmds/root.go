package cmds

import (
	"fmt"
	"os"
	"path"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"
)

var cfgPath string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "theta-rosetta-rpc-adaptor",
	Short: "Theta Rosetta RPC Adaptor",
	Long:  `Theta Rosetta RPC Adaptor`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgPath, "config", getDefaultConfigPath(), fmt.Sprintf("config path (default is %s)", getDefaultConfigPath()))
	viper.BindPFlag(cmn.CfgRosettaMode, RootCmd.PersistentFlags().Lookup("mode"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AddConfigPath(cfgPath)

	// Search config (without extension).
	viper.SetConfigName("config")

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func getDefaultConfigPath() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return path.Join(home, ".theta-rosetta-rpc-adaptor")
}
