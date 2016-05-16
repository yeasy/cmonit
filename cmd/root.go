// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/util"
	"path/filepath"
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("cmd")

var (
	cfgFile string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cmonit",
	Short: "A container monitor",
	Long: `Monitor the container host health, container stats...`,
// Uncomment the following line if your bare application
// has an action associated with it:
//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	if logLevel, err := logging.LogLevel(viper.GetString("logging.level")); err != nil{
		panic(fmt.Errorf("Failed to load logging level: %s", err))
	} else {
		logging.SetLevel(logLevel, "cmd")
	}

	logger.Debugf("logging.level=%s", viper.GetString("logging.level"))
	logger.Debugf("monitor.interval=%d", viper.GetInt("monitor.interval"))
	logger.Debugf("sync.interval=%d", viper.GetInt("sync.interval"))

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	pFlags := RootCmd.PersistentFlags()

	pFlags.StringVar(&cfgFile, "config", "",
		"config file (default name is cmonit.yaml, will search paths of $HOME, /etc/, ./ or GOPATH/pkg)")
	pFlags.String("logging-level", "debug", "logging level")
	pFlags.Int("monitor-interval", 10, "Interval to collect the monitor data.")
	pFlags.Int("sync-interval", 10, "Interval to sync the host info.")

	// Use viper to track those flags
	viper.BindPFlag("logging.level", pFlags.Lookup("logging-level"))
	viper.BindPFlag("monitor.interval", pFlags.Lookup("monitor-interval"))
	viper.BindPFlag("sync.interval", pFlags.Lookup("sync-interval"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(util.RootName) // Name of config file (without extension)
	viper.AddConfigPath("$HOME")  // adding home directory as first search path
	viper.AddConfigPath("..")
	viper.AddConfigPath("/etc/"+util.RootName)
	// Path to look for the config file in based on GOPATH
	gopath := os.Getenv("GOPATH")
	for _, p := range filepath.SplitList(gopath) {
		peerpath := filepath.Join(p, "src/github.com/yeasy/cmonit")
		viper.AddConfigPath(peerpath)
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Found config file: %s\n", viper.ConfigFileUsed())
	} else{
		panic(fmt.Errorf("Fatal error when reading config %s: %s\n", util.RootName, err))
	}

	viper.SetEnvPrefix(util.RootName)
	viper.AutomaticEnv()          // read in environment variables that match
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}
