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
	"runtime"
	"strings"
	"time"

	"github.com/op/go-logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/agent"
	"github.com/yeasy/cmonit/data"
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
	pFlags.String("input-mongo-url", "mongo:27017", "URL of the db API")
	pFlags.String("input-mongo-db_name", "dev", "db name to use")
	pFlags.String("input-mongo-col_host", "host", "name of the host info collection")
	pFlags.String("input-mongo-col_cluster", "cluster_active", "name of the running cluster collection")

	pFlags.String("output-mongo-url", "", "URL of the db API")
	pFlags.String("output-mongo-db_name", "monitor", "db name to use")
	pFlags.String("output-mongo-col_host", "host", "name of the host info collection")
	pFlags.String("output-mongo-col_cluster", "cluster", "name of the running cluster collection")
	pFlags.String("output-elasticsearch-url", "", "URL of the es API")
	pFlags.String("output-elasticsearch-index", "monitor", "es index")

	//pFlags.Int("sync-interval", 30, "Interval to sync the info from db.")

	pFlags.Int("monitor-expire", 7, "Days wait to expire the monitor data, -1 means never expire.")
	pFlags.Int("monitor-interval", 30, "Seconds of interval to monitor.")

	// Use viper to track those flags
	viper.BindPFlag("input.mongo.url", pFlags.Lookup("input-mongo-url"))
	viper.BindPFlag("input.mongo.db_name", pFlags.Lookup("input-mongo-db_name"))
	viper.BindPFlag("input.mongo.col_host", pFlags.Lookup("input-mongo-col_host"))
	viper.BindPFlag("input.mongo.col_cluster", pFlags.Lookup("input-mongo-col_cluster"))

	viper.BindPFlag("output.mongo.url", pFlags.Lookup("output-mongo-url"))
	viper.BindPFlag("output.mongo.db_name", pFlags.Lookup("output-mongo-db_name"))
	viper.BindPFlag("output.mongo.col_host", pFlags.Lookup("output-mongo-col_host"))
	viper.BindPFlag("output.mongo.col_cluster", pFlags.Lookup("output-mongo-col_cluster"))
	viper.BindPFlag("output.elasticsearch.url", pFlags.Lookup("output-elasticsearch-url"))
	viper.BindPFlag("output.elasticsearch.index", pFlags.Lookup("output-elasticsearch-index"))

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
		logger.Infof("Setting logging level=%s\n", loggingLevel)
		logging.SetLevel(logLevel, "cmonit")
	}

	for _, k := range viper.AllKeys() {
		logger.Debugf("%s = %v\n", k, viper.Get(k))
	}

	//open and init input db
	input := new(data.DB)
	defer input.Close()
	if err := input.Init(viper.GetString("input.mongo.url"), viper.GetString("input.mongo.db_name")); err != nil {
		logger.Errorf("Cannot init input db with %s\n", viper.GetString("input.mongo.url"))
		return err
	}
	input.SetCol("host", viper.GetString("input.mongo.col_host"))
	input.SetCol("cluster", viper.GetString("input.mongo.col_cluster"))
	logger.Debugf("Inited input DB session: %s %s", viper.GetString("input.mongo.url"), viper.GetString("input.mongo.db_name"))

	//open and init output db
	var output *data.DB
	outputURL, outputDB := viper.GetString("output.mongo.url"), viper.GetString("output.mongo.db_name")
	if outputURL != "" {
		output = new(data.DB)
		defer output.Close()
		if err := output.Init(outputURL, outputDB); err != nil {
			logger.Errorf("Cannot init output db with %s\n", outputURL)
			return err
		}
		logger.Debugf("Opened output DB session: %s %s", outputURL, outputDB)
		output.SetCol("host", viper.GetString("output.mongo.col_host"))
		output.SetCol("cluster", viper.GetString("output.mongo.col_cluster"))
		output.SetCol("container", viper.GetString("output.mongo.col_container"))
		output.SetIndex("host", "host_id", viper.GetInt("monitor.expire"))
		output.SetIndex("cluster", "cluster_id", viper.GetInt("monitor.expire"))
		output.SetIndex("container", "container_id", viper.GetInt("monitor.expire"))
		logger.Debugf("Inited output DB session: %s %s", outputURL, outputDB)
	}

	// period monitor container stats and write into db
	monitTask(input, output)

	messages := make(chan string)
	defer close(messages)

	<-messages

	return nil
}

// main process will be done within the function
func monitTask(input, output *data.DB) {
	var (
		hosts *[]data.Host
		err   error
		mem   runtime.MemStats
	)
	hms := make(map[string]*agent.HostMonitor)

	for {
		interval := time.Duration(viper.GetInt("monitor.interval"))
		logger.Infof(">>>Start monitor task, interval = %d seconds\n", interval)

		//first sync info
		syncStart := time.Now()
		if hosts, err = input.GetHosts(); err != nil {
			logger.Warning("<<<Failed to sync host info")
			logger.Error(err)
			time.Sleep(interval * time.Second)

			if err = input.ReDial(); err != nil {
				logger.Errorf("Failed to redial db url=%s\n", input.URL)
			}
			logger.Infof("Redialed db=%s\n", input.URL)

			/*
				logger.Infof("Redialed db=%s\n", output.URL)
				if err = output.ReDial(); err != nil {
					logger.Errorf("Failed to redial db url=%s\n", output.URL)
				}*/
			continue
		}
		syncEnd := time.Now()
		syncTime := syncEnd.Sub(syncStart)
		lenHosts := len(*hosts)
		logger.Infof("===Synced task done: %d hosts found\n", lenHosts)
		logger.Debugf("%+v\n", *hosts)

		if lenHosts <= 0 {
			logger.Info("No monit will be started without hosts")
			time.Sleep(interval * time.Second)
			continue
		}

		//now collect data
		monitStart := time.Now()
		c := make(chan string)
		for i := 0; i < lenHosts; i++ {
			h := (*hosts)[i]
			logger.Debugf("Monit task [%d/%d]: start for host=%s", i, lenHosts, h.Name)
			if _, ok := hms[h.DaemonURL]; !ok { //not see the host before
				hm := new(agent.HostMonitor)
				if err := hm.Init(&h, input, output, viper.GetString("output.mongo.col_host")); err != nil {
					logger.Warningf("<<Fail to init host %s", h.Name)
					c <- h.Name
					continue
				}
				logger.Infof("create new hm for host=%s\n", h.Name)
				hms[h.DaemonURL] = hm
			}
			go hms[h.DaemonURL].Monit(h, input, output, c)
		}

		number := 0
		hostNames := []string{}
		for name := range c {
			number++
			hostNames = append(hostNames, name)
			logger.Infof("===Monit task [%d/%d]: done hosts = %v", number, lenHosts, hostNames)
			if number >= lenHosts {
				close(c)
				break
			}
		}
		monitEnd := time.Now()
		monitTime := monitEnd.Sub(monitStart)

		//runtime.GC()

		runtime.ReadMemStats(&mem)
		logger.Infof("<<<End monitor task. sync used %s, monit used %s, interval=%d seconds. Memory usage = %d KB.\n\n", syncTime, monitTime, interval, mem.Alloc/1024)
		time.Sleep(interval * time.Second)
	}
}
