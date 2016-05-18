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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/util"
	"strings"
	"time"
	"github.com/op/go-logging"
	"github.com/yeasy/cmonit/monit"
)

var hosts []util.Host

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the monit daemon",
	Long: `Start the cmonit daemon and run the tasks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Work your own magic here
		logger.Debug("start cmd is called")
		return serve(args)
	},
}

func init() {
	RootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")
	pFlags := startCmd.PersistentFlags()
	pFlags.String("db-url", "127.0.0.1:27017", "URL of the db API")
	pFlags.String("db-name", "dev", "db name to use")
	pFlags.String("db-col_host", "host", "name of the host info collection")
	pFlags.String("db-col_monitor", "monitor", "name of the monitor collection")
	pFlags.Int("monitor-interval", 30, "Seconds of interval to collect the monitor data.")
	pFlags.Int("monitor-expire", 7, "Days wait to expire the monitor data, -1 means never expire.")
	pFlags.Int("sync-interval", 60, "Interval to sync the host info.")

	// Use viper to track those flags
	viper.BindPFlag("db.url", pFlags.Lookup("db-url"))
	viper.BindPFlag("db.name", pFlags.Lookup("db-name"))
	viper.BindPFlag("db.col_host", pFlags.Lookup("db-col_host"))
	viper.BindPFlag("db.col_monitor", pFlags.Lookup("db-col_monitor"))
	viper.BindPFlag("monitor.interval", pFlags.Lookup("monitor-interval"))
	viper.BindPFlag("monitor.expire", pFlags.Lookup("monitor-expire"))
	viper.BindPFlag("sync.interval", pFlags.Lookup("sync-interval"))
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func serve(args [] string) error {
	logging_level := strings.ToUpper(viper.GetString("logging.level"))
	if logLevel, err := logging.LogLevel(logging_level); err != nil{
		panic(fmt.Errorf("Failed to load logging level: %s", err))
	} else {
		logging.SetLevel(logLevel, "cmd")
		logger.Debugf("Setting logging level=%s\n", logging_level)
	}

	logger.Warning("Call serve() function")
	logger.Debugf("logging.level=%s\n", viper.GetString("logging.level"))
	logger.Debugf("db.url=%s\n", viper.GetString("db.url"))
	logger.Debugf("db.name=%s\n", viper.GetString("db.name"))
	logger.Debugf("db.col_host=%s\n", viper.GetString("db.col_host"))
	logger.Debugf("db.col_monitor=%s\n", viper.GetString("db.col_monitor"))
	logger.Debugf("monitor.interval=%d\n", viper.GetInt("monitor.interval"))
	logger.Debugf("monitor.expire=%d\n", viper.GetInt("monitor.expire"))
	logger.Debugf("sync.interval=%d\n", viper.GetInt("sync.interval"))


	db_url := viper.GetString("db.url")
	db_name := viper.GetString("db.name")

	db := new(util.DB)
	if _, err := db.Init(db_url, db_name); err != nil{
		logger.Errorf("Cannot init db with %s\n", db_url)
		return err
	}
	logger.Debugf("Opened DB session: %s %s",db_url, db_name)

	defer db.Close()

	// period sync data for hosts
	go syncInfo(db)
	go monitorTask(db)

	// period monitor container stats and write into db

	messages := make(chan string)
	<- messages

	return nil
}

func syncInfo(db *util.DB) {
	var err error
	for ;;{
		interval := time.Duration(viper.GetInt("sync.interval"))
		logger.Infof(">>>Run sync task, interval=%d seconds\n", interval)

		if hosts, err = db.GetHosts(); err != nil {
			logger.Warning("<<<Failed to sync host info")
			logger.Error(err)
		} else {
			logger.Debugf("<<<Synced host info: %d found: %+v\n", len(hosts), hosts)
		}

		time.Sleep(interval * time.Second)
	}
}
func monitorTask(db *util.DB) {
	cm := new(monit.ContainersMonitor)
	for ;;{
		interval := time.Duration(viper.GetInt("monitor.interval"))
		logger.Infof(">>>Run monitor task, interval=%d seconds\n", interval)

		for _, h:= range hosts {
			logger.Debugf(">>>Collect data for host=%s", h.Name)
			if err := cm.Init(h.Daemon_URL); err != nil {
				logger.Warningf("<<<Fail to init connection to %s", h.Name)
				continue
			}
			if err := cm.CollectData(h.ID, db); err != nil {
				logger.Warningf("<<<Fail to collect data from %s", h.Name)
			} else {
				logger.Debugf("<<<Collected and Saved data for host=%s", h.Name)
			}
		}

		time.Sleep(interval * time.Second)
	}
}
