package agent

import (
	"time"

	"errors"

	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/data"
)

// HostMonitor is used to collect data from a whole docker host.
// It may include many clusters
type HostMonitor struct {
	host      *data.Host
	inputDB   *data.DB
	outputDB  *data.DB //output db
	outputCol string   //output collection
}

//Init will do initialization
func (hm *HostMonitor) Init(host *data.Host, input, output *data.DB, colName string) error {
	hm.host = host
	hm.inputDB = input
	hm.outputDB = output
	hm.outputCol = colName

	return nil
}

// CollectData will collect information for each cluster at the host
func (hm *HostMonitor) CollectData() (*data.HostStat, error) {
	//var hasErr bool = false
	var clusters *[]data.Cluster
	var err error
	if clusters, err = hm.inputDB.GetClusters(map[string]interface{}{"host_id": hm.host.ID}); err != nil {
		logger.Errorf("Cannot get clusters: %+v\n", err.Error())
		return nil, err
	}
	lenClusters := len(*clusters)
	// Use go routine to collect data and send result pointer to channel
	logger.Debugf("Host %s: monit %d clusters\n", hm.host.Name, lenClusters)
	if lenClusters <= 0 {
		logger.Debugf("%d clusters, just return\n", lenClusters)
		return nil, errors.New("No container found in cluster")
	}
	c := make(chan *data.ClusterStat, lenClusters)
	defer close(c)
	for _, cluster := range *clusters {
		logger.Debugf("Host %s has cluster %s\n", hm.host.Name, cluster.ID)
		clm := new(ClusterMonitor)
		go clm.Monit(cluster, hm.outputDB, viper.GetString("output.mongo.col_cluster"), c)
	}

	// Collect valid results from channel
	number := 0
	csList := []*data.ClusterStat{}
	for s := range c {
		if s != nil { //collect some data
			csList = append(csList, s)
			logger.Debugf("Host %s/Cluster %s: monit done\n", hm.host.Name, s.ClusterID)
		}
		number++
		if number >= lenClusters {
			//close(c)
			break
		}
	}

	if len(csList) <= 0 {
		logger.Warningf("Invalid cluster stat number = %d\n", len(csList))
		return nil, errors.New("Invalid cluster stat number")
	}

	hs := data.HostStat{
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
		TimeStamp:        time.Now().UTC(),
	}
	(&hs).CalculateStat(csList)
	logger.Debugf("Host %s: collected result = %+v\n", hm.host.Name, hs)
	return &hs, nil
}

// Monit will start the monit task on the host
func (hm *HostMonitor) Monit(host data.Host, inputDB, outputDB *data.DB, c chan string) {
	logger.Infof(">>Starting monit host=%s\n", host.Name)
	if err := hm.Init(&host, inputDB, outputDB, viper.GetString("output.mongo.col_host")); err != nil {
		logger.Warningf("<<Fail to init connection to %s", host.Name)
		c <- host.Name
		return
	}
	if hs, err := hm.CollectData(); err != nil {
		logger.Warningf("<<Fail to collect data for host %s\n", host.Name)
	} else {
		if outputDB != nil && outputDB.URL != "" && outputDB.Name != "" && hm.outputCol != "" {
			outputDB.SaveData(hs, hm.outputCol)
			logger.Debugf("Host %s: saved to db %s/%s/%s\n", host.Name, outputDB.URL, outputDB.Name, hm.outputCol)
		}
		if url, index := viper.GetString("output.elasticsearch.url"), viper.GetString("output.elasticsearch.index"); url != "" && index != "" {
			esDoc := make(map[string]interface{})
			esDoc["host_id"] = hs.HostID
			esDoc["host_name"] = hs.HostName
			esDoc["cpu_percentage"] = hs.CPUPercentage
			esDoc["memory_usage"] = hs.Memory
			esDoc["memory_limit"] = hs.MemoryLimit
			esDoc["memory_percentage"] = hs.MemoryPercentage
			esDoc["network_rx"] = hs.NetworkRx
			esDoc["network_tx"] = hs.NetworkTx
			esDoc["block_read"] = hs.BlockRead
			esDoc["block_write"] = hs.BlockWrite
			esDoc["max_latency"] = hs.MaxLatency
			esDoc["avg_latency"] = hs.AvgLatency
			esDoc["min_latency"] = hs.MinLatency
			esDoc["timestamp"] = hs.TimeStamp.Format("2006-01-02 15:04:05")
			data.ESInsertDoc(url, index, "host", esDoc)
			logger.Infof("Host %s: saved to es %s/%s/%s\n", host.Name, url, index, "host")
		}
		logger.Infof("<<End monit for host=%s\n", host.Name)
	}
	c <- host.Name
}
