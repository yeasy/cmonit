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
	"github.com/yeasy/cmonit/agent"
	"github.com/yeasy/cmonit/database"
)

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

	//pFlags.Int("sync-interval", 30, "Interval to sync the info from db.")

	pFlags.Int("monitor-expire", 7, "Days wait to expire the monitor data, -1 means never expire.")
	pFlags.Int("monitor-interval", 30, "Seconds of interval to monitor.")

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

	viper.BindPFlag("monitor.expire", pFlags.Lookup("monitor-expire"))
	viper.BindPFlag("monitor.interval", pFlags.Lookup("monitor-interval"))
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

	//open and init input db
	inputURL, inputDB := viper.GetString("input.url"), viper.GetString("input.db_name")
	input := new(database.DB)
	if err := input.Init(inputURL, inputDB); err != nil {
		logger.Errorf("Cannot init db with %s\n", inputURL)
		return err
	}
	defer input.Close()
	input.SetCol("host", viper.GetString("input.col_host"))
	input.SetCol("cluster", viper.GetString("input.col_cluster"))
	logger.Debugf("Inited input DB session: %s %s", inputURL, inputDB)

	//open and init output db
	outputURL, outputDB := viper.GetString("output.mongo.url"), viper.GetString("output.mongo.db_name")
	output := new(database.DB)
	if err := output.Init(outputURL, outputDB); err != nil {
		logger.Errorf("Cannot init db with %s\n", outputURL)
		return err
	}
	defer output.Close()
	logger.Debugf("Opened output DB session: %s %s", outputURL, outputDB)
	output.SetCol("host", viper.GetString("output.mongo.col_host"))
	output.SetCol("cluster", viper.GetString("output.mongo.col_cluster"))
	output.SetCol("container", viper.GetString("output.mongo.col_container"))
	output.SetIndex("host", "host_id", viper.GetInt("monitor.expire"))
	output.SetIndex("cluster", "cluster_id", viper.GetInt("monitor.expire"))
	output.SetIndex("container", "container_id", viper.GetInt("monitor.expire"))
	logger.Debugf("Inited output DB session: %s %s", outputURL, outputDB)

	// period monitor container stats and write into db
	go monitTask(input, output)

	messages := make(chan string)
	<-messages

	return nil
}

func monitTask(input, output *database.DB) {
	interval := time.Duration(viper.GetInt("monitor.interval"))
	var hosts *[]database.Host
	var err error
	for {
		logger.Infof(">>>Run monitor task, interval=%d seconds\n", interval)

		//first sync info
		start := time.Now()
		if hosts, err = input.GetHosts(); err != nil {
			logger.Warning("<<<Failed to sync host info")
			logger.Error(err)
			time.Sleep(interval * time.Second)
			continue
		}
		logger.Debugf("<<<Synced host info: %d found: %+v\n", len(*hosts), *hosts)
		end := time.Now()
		delta := end.Sub(start)
		fmt.Printf("sync task used time: %s\n", delta)

		//now collect data
		for _, h := range *hosts {
			logger.Debugf(">>>Starting monit host=%s\n", h.Name)
			hm := new(agent.HostMonitor)
			go hm.Monit(&h, input, output)
		}

		time.Sleep(interval * time.Second)
	}
}
