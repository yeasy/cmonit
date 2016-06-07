package agent

import (
	"io/ioutil"
	"sort"
	"time"
	"net"
	"net/http"

	"regexp"
	"runtime"
	"strconv"
	"strings"

	"errors"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/spf13/viper"
	"github.com/yeasy/cmonit/data"
	"github.com/yeasy/cmonit/util"
	"golang.org/x/net/context"
)

// ClusterMonitor is used to collect data from a whole docker host.
// It may include many clusters
type ClusterMonitor struct {
	cluster      *data.Cluster //cluster collection
	output       *data.DB      //save out
	DockerClient *client.Client
}

// Monit will write pointer of result to the channel
// Even fail, must write nil
func (clm *ClusterMonitor) Monit(cluster data.Cluster, outputDB *data.DB, outputCol string, c chan *data.ClusterStat) {
	logger.Debugf("Cluster %s (%s): Starting monit task\n", cluster.Name, cluster.ID)
	if err := clm.Init(&cluster, outputDB); err != nil {
		logger.Error(err)
		c <- nil
		return
	}
	s, err := clm.CollectData()
	if err != nil {
		logger.Error(err)
		c <- nil
		return
	}
	//now get the stat for the cluster, may save to db and return to chan
	logger.Debugf("Cluster %s: report collected data\n%+v", cluster.Name, *s)
	c <- s

	if outputDB != nil && outputDB.URL != "" && outputDB.Name != "" && outputCol != "" {
		outputDB.SaveData(*s, outputCol)
		logger.Debugf("Cluster %s: saved to db %s/%s/%s\n", cluster.Name, outputDB.URL, outputDB.Name, outputCol)
	}
	if url, index := viper.GetString("output.elasticsearch.url"), viper.GetString("output.elasticsearch.index"); url != "" && index != "" {
		esDoc := make(map[string]interface{})
		esDoc["cluster_id"] = s.ClusterID
		esDoc["cluster_name"] = s.ClusterName
		esDoc["cpu_percentage"] = s.CPUPercentage
		esDoc["memory_usage"] = s.Memory
		esDoc["memory_limit"] = s.MemoryLimit
		esDoc["memory_percentage"] = s.MemoryPercentage
		esDoc["network_rx"] = s.NetworkRx
		esDoc["network_tx"] = s.NetworkTx
		esDoc["block_read"] = s.BlockRead
		esDoc["block_write"] = s.BlockWrite
		esDoc["size"] = s.Size
		esDoc["max_latency"] = s.MaxLatency
		esDoc["avg_latency"] = s.AvgLatency
		esDoc["min_latency"] = s.MinLatency
		esDoc["latencies"] = s.Latencies
		esDoc["timestamp"] = s.TimeStamp.Format("2006-01-02 15:04:05")
		data.ESInsertDoc(url, index, "cluster", esDoc)
		logger.Debugf("Cluster %s: saved to es %s/%s/%s\n", cluster.Name, url, index, "cluster")
	}
	runtime.Goexit()
}

//Init will finish the initialization
func (clm *ClusterMonitor) Init(cluster *data.Cluster, output *data.DB) error {
	clm.cluster = cluster
	clm.output = output

	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	httpClient := http.Client{
		Transport: &http.Transport{
			//MaxIdleConnsPerHost: 32,
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 15 * time.Second,
			}).Dial,
			MaxIdleConnsPerHost: 64,
			DisableKeepAlives:   true, // use this to prevent many connections opened
		},
		Timeout: time.Duration(5) * time.Second,
	}
	cli, err := client.NewClient(clm.cluster.DaemonURL, "", &httpClient, defaultHeaders)
	if err != nil {
		logger.Errorf("Cannot init connection to docker host=%s\n", clm.cluster.DaemonURL)
		logger.Error(err)
		return err
	}
	clm.DockerClient = cli
	return nil
}

// CollectData will collect information from docker host
func (clm *ClusterMonitor) CollectData() (*data.ClusterStat, error) {
	//for each container, collect result
	//var hasErr bool = false
	containers := clm.cluster.Containers
	lenContainers := len(containers)
	logger.Debugf("Cluster %s: monit %d containers\n", clm.cluster.Name, lenContainers)
	if lenContainers <= 0 {
		logger.Debugf("%d containers, just return\n", lenContainers)
		return nil, errors.New("No container found in cluster")
	}

	// Use go routine to collect data and send result pointer to channel
	ct := make(chan *data.ContainerStat, lenContainers)
	defer close(ct)
	names := []string{}
	for name, id := range containers {
		ctm := new(ContainerMonitor)
		go ctm.Monit(clm.DockerClient, id, name, viper.GetString("output.mongo.col_container"), clm.output, ct)
		names = append(names, name)
	}
	sort.Strings(names)
	// Check results from channel
	number := 0
	csList := []*data.ContainerStat{}
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
	if len(csList) != lenContainers {
		logger.Errorf("Cluster %s: only collected %d/%d container data\n", clm.cluster.Name, len(csList), lenContainers)
		return nil, errors.New("Not enough container data collected")
	}
	cs := data.ClusterStat{
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
		TimeStamp:        time.Now().UTC(),
	}
	(&cs).CalculateStat(csList)
	//get the latency here
	if len(names) > 1 {
		latencies, err := clm.calculateLatency(names)
		if err != nil {
			logger.Errorf("Cluster %s: Error to calculate latency\n", clm.cluster.Name)
			return &cs, err
		}

		cs.Latencies = latencies
		cs.AvgLatency = util.Avg(latencies)
		cs.MaxLatency = util.Max(latencies)
		cs.MinLatency = util.Min(latencies)
	}

	logger.Debugf("Cluster %s: collected data = %+v\n", clm.cluster.Name, cs)
	return &cs, nil
}

//getLatency will calculate the latency among the containers
func (clm *ClusterMonitor) calculateLatency(containers []string) ([]float64, error) {
	if len(containers) <= 1 {
		logger.Warningf("Too few %d container to calculate latency\n", len(containers))
		return []float64{}, errors.New("Too few container")
	}
	//defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	//cli, err := client.NewClient(clm.cluster.DaemonURL, "", nil, defaultHeaders)
	//if err != nil {
	//	logger.Warningf("Cannot connect to docker host=%s\n", clm.cluster.DaemonURL)
	//	return nil, err
	//}

	c := make(chan float64)
	for i := 0; i < len(containers)-1; i++ {
		for j := i + 1; j < len(containers); j++ {
			go getLantecy(clm.DockerClient, containers[i], containers[j], c)
		}
	}

	number, result := 0, []float64{}
	for laten := range c {
		result = append(result, laten)
		number++
		if number >= len(containers)*(len(containers)-1)/2 {
			close(c)
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
		return
	}

	execID := response.ID
	if execID == "" {
		logger.Warning("exec ID empty")
		c <- 2000
		return
	}
	res, err := cli.ContainerExecAttach(context.Background(), execID, execConfig)
	defer res.Close()
	defer ioutil.ReadAll(res.Reader)

	if err != nil {
		logger.Error("Cannot attach docker exec")
		c <- 2000
		return
	}
	v := make([]byte, 5000)
	var n int
	n, err = res.Reader.Read(v)
	if err != nil {
		logger.Error("Cannot parse cmd output")
		c <- 2000
		return
	}

	re, err := regexp.Compile(`time=([0-9\.]+) ms`)
	result := re.FindStringSubmatch(string(v[:n]))
	if len(result) >= 2 {
		splits := strings.Split(result[1], " ")
		latency, _ := strconv.ParseFloat(splits[len(splits)-1], 64)
		c <- latency
	} else {
		c <- 2000
	}
	runtime.Goexit()
	return
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
