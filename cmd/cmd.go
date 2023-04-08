package cmd

import (
	"os"

	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/sagan/ptool/config"
	"github.com/spf13/cobra"
)

// Root represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ptool",
	Short: "ptool command [flags]",
	Long:  `ptool.`,
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.OnInitialize(func() {
		// level: panic(0), fatal(1), error(2), warn(3), info(4), debug(5), trace(6). Default level = warning(3)
		config.ConfigDir = filepath.Dir(config.ConfigFile)
		logLevel := 3 + config.VerboseLevel
		log.SetLevel(log.Level(logLevel))
		log.Debugf("ptool start: %s", os.Args)
		log.Infof("config file: %s", config.ConfigFile)
	})
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	configFile, _ := os.UserHomeDir()
	configFile += "/.config/ptool/ptool.yaml"
	_, err := os.Stat(configFile)
	if err != nil {
		if _, err = os.Stat("ptool.yaml"); err == nil {
			configFile = "ptool.yaml"
		}
	}

	// global flags
	RootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", configFile, "config file ([ptool.yaml])")
	RootCmd.PersistentFlags().CountVarP(&config.VerboseLevel, "", "v", "verbose (-v, -vv, -vvv)")
	// local flags
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
