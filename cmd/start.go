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
	"github.com/spf13/cobra"
	"github.com/yeasy/cmonit/util"
	"github.com/spf13/viper"
	"time"
)

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

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func serve(args [] string) error {
	logger.Debug("Call serve() function")

	mongo_url := viper.GetString("mongo")

	db := new(util.DB)
	if _, err := db.Init(mongo_url, "dev"); err != nil{
		logger.Errorf("Cannot init db with %s\n", mongo_url)
		return err
	}

	defer db.Close()

	// period sync data for hosts
	db.GetHosts()
	go syncInfo()
	go monitorTask()

	// period monitor container stats and write into db

	messages := make(chan string)
	<- messages

	return nil
}

func syncInfo() {
	for ;;{
		logger.Info("Run sync task")
		interval := time.Duration(viper.GetInt("sync.interval"))
		time.Sleep(interval * 1000 * time.Millisecond)
	}
}
func monitorTask() {
	for ;;{
		logger.Info("Run monitor task")
		interval := time.Duration(viper.GetInt("monitor.interval"))
		time.Sleep(interval * 1000 * time.Millisecond)
	}
}
