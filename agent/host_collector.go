package agent

import (
	_ "github.com/op/go-logging"
	"github.com/yeasy/cmonit/database"
	_ "github.com/spf13/viper"
	_ "errors"
	"time"
)

// HostMonitor is used to collect data from a whole docker host.
// It may include many clusters
type HostMonitor struct {
	host *database.Host
	input *database.DB
	output *database.DB //output db
	colName string //output collection
}

func (hm *HostMonitor) Init (host *database.Host, input, output *database.DB, colName string) error {
	hm.host = host
	hm.input = input //.cols[viper.GetString("input.col_host")]
	hm.output = output
	hm.colName = colName

	return nil
}

// CollectData will collect information from docker host
func (hm HostMonitor) CollectData() error {
	logger.Debug("chan finished")
	return nil

	//var hasErr bool = false
	var clusters *[]database.Cluster
	var err error
	if clusters, err = hm.input.GetClusters(); err != nil {
		logger.Warningf("Cannot get clusters %s\n")
		return err
	}
	logger.Debugf("CollectData for host %s, get %d clusters\n", len(clusters))
	c := make(chan * database.ClusterStat, len(clusters))
	for i:=0;i <len(clusters); i++ {
		go test(c)
	}

	// Use go routine to collect data and send to channel
	for _, cluster := range *clusters {
		go ClusterMonitTask(&cluster, hm.output, c)
	}

	// Check results from channel
	hs := database.HostStat{
		HostID: hm.host.ID,
		HostName: hm.host.Name,
		CPUPercentage: 0.0,
		Memory: 0.0,
		MemoryLimit: 0.0,
		MemoryPercentage: 0.0,
		NetworkRx:0.0,
		NetworkTx: 0.0,
		BlockRead: 0.0,
		BlockWrite: 0.0,
		PidsCurrent: 0,
		AvgLatency: 0.0,
		MaxLatency: 0.0,
		MinLatency: 0.0,
		TimeStamp: time.Now(),
	}
	number := 0
	resultList := [] *database.ClusterStat{}
	for s := range c {
		if s != nil { //collect some data
			append(resultList, s)
		}
		number += 1
		if number >= len(clusters) {
			break
		}
	}

	return nil
}

func calculateHostStat(hs *database.HostStat, resultList []*database.ClusterStat) {
	number := len(resultList)
	c <- 1
}
