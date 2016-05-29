package agent

import (
	"time"

	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/database"
)

// HostMonitor is used to collect data from a whole docker host.
// It may include many clusters
type HostMonitor struct {
	host      *database.Host
	inputDB   *database.DB
	outputDB  *database.DB //output db
	outputCol string       //output collection
}

//Init will do initialization
func (hm *HostMonitor) Init(host *database.Host, input, output *database.DB, colName string) error {
	hm.host = host
	hm.inputDB = input
	hm.outputDB = output
	hm.outputCol = colName

	return nil
}

// CollectData will collect information for each cluster at the host
func (hm *HostMonitor) CollectData() (*database.HostStat, error) {
	//var hasErr bool = false
	logger.Debugf("Host %s: Starting get clusters\n", hm.host.Name)
	var clusters *[]database.Cluster
	var err error
	if clusters, err = hm.inputDB.GetClusters(); err != nil {
		logger.Errorf("Cannot get clusters: %+v\n", err.Error())
		return nil, err
	}
	lenClusters := len(*clusters)
	logger.Debugf("Host %s: Got %d clusters\n", hm.host.Name, lenClusters)
	c := make(chan *database.ClusterStat, lenClusters)

	// Use go routine to collect data and send result pointer to channel
	logger.Debugf("Host %s: starting monit task\n", hm.host.Name)
	for _, cluster := range *clusters {
		clm := new(ClusterMonitor)
		go clm.Monit(&cluster, hm.outputDB, viper.GetString("output.mongo.col_cluster"), c)
	}

	// Collect valid results from channel
	number := 0
	csList := []*database.ClusterStat{}
	for s := range c {
		if s != nil { //collect some data
			csList = append(csList, s)
			logger.Debugf("Host %s: Received result for cluster %s\n", hm.host.Name, s.ClusterID)
		}
		number++
		if number >= lenClusters {
			break
		}
	}

	hs := database.HostStat{
		HostID:           hm.host.ID,
		HostName:         hm.host.Name,
		CPUPercentage:    0.0,
		Memory:           0.0,
		MemoryLimit:      0.0,
		MemoryPercentage: 0.0,
		NetworkRx:        0.0,
		NetworkTx:        0.0,
		BlockRead:        0.0,
		BlockWrite:       0.0,
		PidsCurrent:      0,
		AvgLatency:       0.0,
		MaxLatency:       0.0,
		MinLatency:       0.0,
		TimeStamp:        time.Now(),
	}
	(&hs).CalculateStat(csList)
	logger.Debugf("Host %s: collected result = %+v\n", hm.host.Name, hs)
	return &hs, nil
}

// Monit will start the monit task on the host
func (hm *HostMonitor) Monit(host *database.Host, inputDB, outputDB *database.DB) {
	if err := hm.Init(host, inputDB, outputDB, viper.GetString("output.mongo.col_host")); err != nil {
		logger.Warningf("<<<Fail to init connection to %s", host.Name)
		return
	}
	logger.Debugf("host handler inited for host=%s\n", host.Name)
	if hs, err := hm.CollectData(); err != nil {
		logger.Warningf("<<<Fail to collect data from %s\n", host.Name)
	} else {
		logger.Debugf("<<<Collected and Saved data for host=%s\n", host.Name)
		outputDB.SaveData(hs, hm.outputCol)
	}
}
