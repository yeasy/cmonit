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
	"strings"
	"time"

	"github.com/op/go-logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/monit"
	"github.com/yeasy/cmonit/util"
)

var hosts *[]util.Host

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the monit daemon",
	Long:  `Start the cmonit daemon and run the tasks.`,
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
	pFlags.String("input-url", "127.0.0.1:27017", "URL of the db API")
	pFlags.String("input-db_name", "dev", "db name to use")
	pFlags.String("input-col_host", "host", "name of the host info collection")
	pFlags.String("input-col_cluster", "cluster_active", "name of the running cluster collection")

	pFlags.String("output-mongo-url", "127.0.0.1:27017", "URL of the db API")
	pFlags.String("output-mongo-db_name", "dev", "db name to use")
	pFlags.String("output-mongo-col_host", "host", "name of the host info collection")
	pFlags.String("output-mongo-col_cluster", "cluster", "name of the running cluster collection")
	pFlags.String("output-es-url", "127.0.0.1:9200", "URL of the es API")

	pFlags.Int("sync-interval", 30, "Interval to sync the info from db.")

	pFlags.Int("monitor-expire", 7, "Days wait to expire the monitor data, -1 means never expire.")
	pFlags.Int("monitor-system-interval", 30, "Seconds of interval to collect the system data.")
	pFlags.Int("monitor-network-interval", 10, "Seconds of interval to collect the network data.")

	// Use viper to track those flags
	viper.BindPFlag("input.url", pFlags.Lookup("input-url"))
	viper.BindPFlag("input.db_name", pFlags.Lookup("input-db_name"))
	viper.BindPFlag("input.col_host", pFlags.Lookup("input-col_host"))
	viper.BindPFlag("input.col_cluster", pFlags.Lookup("input-col_cluster"))

	viper.BindPFlag("output.mongo.url", pFlags.Lookup("output-mongo-url"))
	viper.BindPFlag("output.mongo.db_name", pFlags.Lookup("output-mongo-db_name"))
	viper.BindPFlag("output.mongo.col_host", pFlags.Lookup("output-mongo-col_host"))
	viper.BindPFlag("output.mongo.col_cluster", pFlags.Lookup("output-mongo-col_cluster"))
	viper.BindPFlag("output.es.url", pFlags.Lookup("output-es-url"))

	viper.BindPFlag("sync.interval", pFlags.Lookup("sync-interval"))

	viper.BindPFlag("monitor.expire", pFlags.Lookup("monitor-expire"))
	viper.BindPFlag("monitor.system.interval", pFlags.Lookup("monitor-system-interval"))
	viper.BindPFlag("monitor.network.interval", pFlags.Lookup("monitor-network-interval"))
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func serve(args []string) error {
	loggingLevel := strings.ToUpper(viper.GetString("logging.level"))
	if logLevel, err := logging.LogLevel(loggingLevel); err != nil {
		panic(fmt.Errorf("Failed to load logging level: %s", err))
	} else {
		logging.SetLevel(logLevel, "cmd")
		logger.Debugf("Setting logging level=%s\n", loggingLevel)
	}

	for _, k := range viper.AllKeys() {
		logger.Debugf("%s = %v\n", k, viper.Get(k))
	}

	//open input db
	inputURL, inputDB := viper.GetString("input.url"), viper.GetString("input.db_name")
	input := new(util.DB)
	if err := input.Init(inputURL, inputDB); err != nil {
		logger.Errorf("Cannot init db with %s\n", inputURL)
		return err
	}
	defer input.Close()
	logger.Debugf("Opened input DB session: %s %s", inputURL, inputDB)
	input.SetCol("host", viper.GetString("input.col_host"))
	input.SetCol("cluster", viper.GetString("input.col_cluster"))

	//open output db
	outputURL, inputDB := viper.GetString("output.mongo.url"), viper.GetString("output.mongo.db_name")
	output := new(util.DB)
	if err := output.Init(outputURL, inputDB); err != nil {
		logger.Errorf("Cannot init db with %s\n", outputURL)
		return err
	}
	defer output.Close()
	logger.Debugf("Opened output DB session: %s %s", outputURL, inputDB)
	input.SetCol("host", viper.GetString("output.mongo.col_host"))
	input.SetCol("cluster", viper.GetString("output.mongo.col_cluster"))
	input.SetCol("container", viper.GetString("output.mongo.col_container"))
	input.SetIndex("host", "host_id", viper.GetInt("monitor.expire"))
	input.SetIndex("cluster", "cluster_id", viper.GetInt("monitor.expire"))
	input.SetIndex("container", "container_id", viper.GetInt("monitor.expire"))

	// period sync data for hosts
	go syncInfo(input)

	// period monitor container stats and write into db
	//go monitorTask(output)

	messages := make(chan string)
	<-messages

	return nil
}

func syncInfo(db *util.DB) {
	for {
		interval := time.Duration(viper.GetInt("sync.interval"))
		logger.Infof(">>>Run sync task, interval=%d seconds\n", interval)

		if hostsTemp, err := db.GetHosts(); err != nil {
			logger.Warning("<<<Failed to sync host info")
			logger.Error(err)
		} else {
			logger.Debugf("<<<Synced host info: %d found: %+v\n", len(*hostsTemp), *hostsTemp)
			hosts = hostsTemp
		}

		time.Sleep(interval * time.Second)
	}
}

func monitorTask(output *util.DB) {
	cm := new(monit.ContainersMonitor)
	for {
		interval := time.Duration(viper.GetInt("monitor.interval"))
		logger.Infof(">>>Run monitor task, interval=%d seconds\n", interval)

		for _, h := range *hosts {
			logger.Debugf(">>>Collect data for host=%s", h.Name)
			if err := cm.Init(h.Daemon_URL); err != nil {
				logger.Warningf("<<<Fail to init connection to %s", h.Name)
				continue
			}
			if err := cm.CollectData(h.ID, output); err != nil {
				logger.Warningf("<<<Fail to collect data from %s", h.Name)
			} else {
				logger.Debugf("<<<Collected and Saved data for host=%s", h.Name)
			}
		}

		time.Sleep(interval * time.Second)
	}
}
