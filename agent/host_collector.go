package agent

import (
	"time"

	"errors"
	"net"
	"net/http"

	"strings"

	"github.com/docker/engine-api/client"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/data"
)

// HostMonitor is used to collect data from a whole docker host.
// It may include many clusters
type HostMonitor struct {
	host         *data.Host
	inputDB      *data.DB
	outputDB     *data.DB //output db
	outputCol    string   //output collection
	dockerClient *client.Client
}

//Init will do initialization
func (hm *HostMonitor) Init(host *data.Host, input, output *data.DB, colName string) error {
	logger.Debugf("Init host=%s", host.Name)
	hm.host = host
	hm.inputDB = input
	hm.outputDB = output
	hm.outputCol = colName

	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}

	httpClient := http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			MaxIdleConnsPerHost: 64,
			//DisableKeepAlives:   true, // use this to prevent many connections opened
			TLSHandshakeTimeout: 5 * time.Second,
		},
		Timeout: time.Duration(30) * time.Second,
	}
	cli, err := client.NewClient(host.DaemonURL, "v1.22", &httpClient, defaultHeaders)
	if err != nil {
		logger.Errorf("Cannot init connection to docker host=%s\n", host.DaemonURL)
		logger.Error(err)
		return err
	}

	hm.dockerClient = cli

	logger.Infof("Inited connection with host=%s", host.DaemonURL)
	return nil
}

// CollectData will collect information for each cluster at the host
func (hm *HostMonitor) CollectData() (*data.HostStat, error) {
	//var hasErr bool = false
	var clusters *[]data.Cluster
	var err error
	if clusters, err = hm.inputDB.GetClusters(map[string]interface{}{"host_id": hm.host.ID}); err != nil {
		logger.Errorf("Host %s: Cannot get clusters: %+v\n", hm.host.Name, err.Error())
		return nil, err
	}
	lenClusters := len(*clusters)
	// Use go routine to collect data and send result pointer to channel
	logger.Debugf("Host %s: has %d clusters\n", hm.host.Name, lenClusters)
	if lenClusters <= 0 {
		logger.Debugf("Host %s: only %d clusters, just return\n", hm.host.Name, lenClusters)
		return nil, errors.New("No cluster in host")
	}
	c := make(chan *data.ClusterStat, lenClusters)
	defer close(c)
	for _, cluster := range *clusters {
		logger.Debugf("Host %s: start monitor cluster %s\n", hm.host.Name, cluster.ID)
		if strings.HasPrefix(cluster.UserID, "__") {
			logger.Debugf("Host %s: cluster %s is in unstable status, ignore\n", hm.host.Name, cluster.ID)
			c <- nil
		} else {
			clm := new(ClusterMonitor)
			go clm.Monit(cluster, hm.outputDB, viper.GetString("output.mongo.col_cluster"), hm.dockerClient, c)
		}
	}

	// Collect valid results from channel
	number := 0
	csList := []*data.ClusterStat{}
	for s := range c {
		if s != nil { //collect some data
			csList = append(csList, s)
			logger.Debugf("Host %s/Cluster %s [%d/%d]: monit done\n", hm.host.Name, s.ClusterID, number, lenClusters)
		}
		number++
		logger.Debugf("Host %s/Cluster [%d/%d]: monit done\n", hm.host.Name, number, lenClusters)
		if number >= lenClusters {
			break
		}
	}

	if len(csList) != lenClusters {
		logger.Errorf("Host %s: only collected %d/%d cluster\n", hm.host.Name, len(csList), lenClusters)
		return nil, errors.New("Not enough cluster data is collected")
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
	if host.Status != "active" {
		logger.Infof("Host %s: Inactive, just return", host.Name)
		c <- host.Name
		return
	}

	logger.Infof(">>Host %s: Starting monit with %d clusters...", host.Name, len(host.Clusters))
	/*
		if err := hm.Init(&host, inputDB, outputDB, viper.GetString("output.mongo.col_host")); err != nil {
			logger.Warningf("<<Fail to init connection to %s", host.Name)
			c <- host.Name
			return
		}*/
	monitStart := time.Now()
	monitTime := time.Now().Sub(monitStart)
	if hs, err := hm.CollectData(); err != nil {
		logger.Warningf("<<Host %s: Fail to collect data!\n", host.Name)
		logger.Error(err)
	} else {
		if outputDB != nil && outputDB.URL != "" && outputDB.Name != "" && hm.outputCol != "" {
			outputDB.SaveData(hs, hm.outputCol)
			logger.Infof("Host %s: saved to DB=%s/%s/%s\n", host.Name, outputDB.URL, outputDB.Name, hm.outputCol)
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
			logger.Infof("Host %s: saved to ES=%s/%s/%s\n", host.Name, url, index, "host")
		}

		monitTime = time.Now().Sub(monitStart)
		logger.Infof("<<Host %s: End monit with %s\n", host.Name, monitTime)
	}
	c <- host.Name
	return
}
