package agent

import (
	"sort"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/database"
)

// ClusterMonitor is used to collect data from a whole docker host.
// It may include many clusters
type ClusterMonitor struct {
	cluster *database.Cluster //cluster collection
	output  *database.DB      //save out
}

// Monit will return pointer of result to the channel
func (clm *ClusterMonitor) Monit(cluster *database.Cluster, outputDB *database.DB, outputCol string, c chan *database.ClusterStat) {
	logger.Debugf("Cluster %s: Starting monit task\n", cluster.Name)
	if err := clm.Init(cluster, outputDB); err != nil {
		c <- nil
		return
	}
	logger.Debugf("Cluster %s: starting collect data\n", cluster.Name)
	if s, err := clm.CollectData(); err != nil {
		c <- nil
	} else {
		//now get the stat for the cluster, may save to db and return to chan
		c <- s
		outputDB.SaveData(*s, outputCol)
	}
}

//Init will finish the initialization
func (clm *ClusterMonitor) Init(cluster *database.Cluster, output *database.DB) error {
	clm.cluster = cluster
	clm.output = output

	return nil
}

// CollectData will collect information from docker host
func (clm *ClusterMonitor) CollectData() (*database.ClusterStat, error) {
	//for each container, collect result
	//var hasErr bool = false
	logger.Debugf("Cluster %s: starting monit task\n", clm.cluster.Name)
	containers := clm.cluster.Containers
	lenContainers := len(containers)
	logger.Debugf("Cluster %s: Got %d containers\n", clm.cluster.Name, lenContainers)

	// Use go routine to collect data and send result pointer to channel
	ct := make(chan *database.ContainerStat, lenContainers)
	names := []string{}
	for name, id := range containers {
		ctm := new(ContainerMonitor)
		go ctm.Monit(clm.cluster.DaemonURL, id, viper.GetString("output.mongo.col_container"), clm.output, ct)
		names = append(names, name)
	}
	sort.Strings(names)
	// Check results from channel
	number := 0
	csList := []*database.ContainerStat{}
	for s := range ct {
		if s != nil { //collect some data
			csList = append(csList, s)
			logger.Debugf("Cluster %s: Received result for container %s\n", clm.cluster.Name, s.ContainerID)
		}
		number++
		if number >= lenContainers {
			break
		}
	}
	cs := database.ClusterStat{
		ClusterID:        clm.cluster.ID,
		ClusterName:      clm.cluster.Name,
		CPUPercentage:    0.0,
		Memory:           0.0,
		MemoryLimit:      0.0,
		MemoryPercentage: 0.0,
		NetworkRx:        0.0,
		NetworkTx:        0.0,
		BlockRead:        0.0,
		BlockWrite:       0.0,
		PidsCurrent:      0,
		Size:             uint64(len(clm.cluster.Containers)),
		AvgLatency:       0.0,
		MaxLatency:       0.0,
		MinLatency:       0.0,
		Latencies:        []float64{},
		TimeStamp:        time.Now(),
	}
	(&cs).CalculateStat(csList)
	logger.Debugf("Cluster %s: collected data = %+v\n", clm.cluster.Name, cs)
	return &cs, nil
}

//getLatency will calculate the latency among the containers
func (clm *ClusterMonitor) getLatency(containers []string) ([]float64, error) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient(clm.cluster.DaemonURL, "", nil, defaultHeaders)
	if err != nil {
		logger.Warningf("Cannot connect to docker host=%s\n", clm.cluster.DaemonURL)
		return nil, err
	}
	logger.Debugf("Connectted to docker host=%s\n", clm.cluster.DaemonURL)

	for _, c := range containers {
		if r, err := cli.ContainerExecCreate(context.Background(), types.ExecConfig{Container: c}); err != nil {
			logger.Error("Cannot execute docker exec")
			logger.Debug(r.ID)
		}
	}

	return []float64{1.0, 2.0}, nil
}
