package agent

import (
	"sort"
	"time"

	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/context"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/database"
	"github.com/yeasy/cmonit/util"
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
	containers := clm.cluster.Containers
	lenContainers := len(containers)
	logger.Debugf("Cluster %s: monit %d containers\n", clm.cluster.Name, lenContainers)

	// Use go routine to collect data and send result pointer to channel
	ct := make(chan *database.ContainerStat, lenContainers)
	defer close(ct)
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
			logger.Debugf("Cluster %s/Container %s: monit done\n", clm.cluster.Name, s.ContainerID)
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
	//get the latency here
	latencies, err := clm.calculateLatency(names)
	if err != nil {
		logger.Error("Error to calculate latency\n")
		return &cs, err
	}

	cs.Latencies = latencies
	cs.AvgLatency = util.Avg(latencies)
	cs.MaxLatency = util.Max(latencies)
	cs.MinLatency = util.Min(latencies)

	logger.Debugf("Cluster %s: collected data = %+v\n", clm.cluster.Name, cs)
	return &cs, nil
}

//getLatency will calculate the latency among the containers
func (clm *ClusterMonitor) calculateLatency(containers []string) ([]float64, error) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient(clm.cluster.DaemonURL, "", nil, defaultHeaders)
	if err != nil {
		logger.Warningf("Cannot connect to docker host=%s\n", clm.cluster.DaemonURL)
		return nil, err
	}
	c := make(chan float64)
	defer close(c)
	for i := 0; i < len(containers)-1; i++ {
		for j := i + 1; j < len(containers); j++ {
			go getLantecy(cli, containers[i], containers[j], c)
		}
	}

	number := 0
	result := []float64{}
	for laten := range c {
		result = append(result, laten)
		number++
		if number >= len(containers)*(len(containers)-1)/2 {
			break
		}
	}

	return result, nil
}

func getLantecy(cli *client.Client, src, dst string, c chan float64) {
	//logger.Debugf("%s -> %s\n", src, dst)
	execConfig := types.ExecConfig{
		Container:    src,
		AttachStdout: true,
		Cmd:          []string{"ping", "-c", "1", "-W", "2", dst},
	}
	response, err := cli.ContainerExecCreate(context.Background(), execConfig)
	if err != nil {
		logger.Warning("exec create failure")
		c <- 2000
	}

	execID := response.ID
	if execID == "" {
		logger.Warning("exec ID empty")
		c <- 2000
	}
	res, err := cli.ContainerExecAttach(context.Background(), execID, execConfig)
	defer res.Close()
	if err != nil {
		logger.Error("Cannot attach docker exec")
		c <- 2000
	}
	v := make([]byte, 5000)
	var n int
	n, err = res.Reader.Read(v)
	if err != nil {
		logger.Error("Cannot parse cmd output")
		c <- 2000
	}

	re, err := regexp.Compile(`time=([0-9\.]+) ms`)
	result := re.FindStringSubmatch(string(v[:n]))
	splits := strings.Split(result[1], " ")
	latency, _ := strconv.ParseFloat(splits[len(splits)-1], 64)
	//logger.Warningf("%+v\n", result[1])
	//logger.Warningf("%s\n", laten)
	//logger.Warningf("%f\n", latency)

	c <- latency
}

/*
	execStartCheck := types.ExecStartCheck{
		Detach: true,
		Tty: false,
	}
	err = cli.ContainerExecStart(context.Background(), execID, execStartCheck)
	if err != nil {
		logger.Warning("exec start failure")
		logger.Warning(err)
		return nil, err
	}
*/
