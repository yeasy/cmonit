package agent

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/yeasy/cmonit/data"
	"golang.org/x/net/context"
)

//ContainerMonitor is used to collect data from a docker host
type ContainerMonitor struct {
	client        *client.Client
	containerID   string
	containerName string
	outputDB      *data.DB
	DaemonURL     string
}

// Monit will collect data for a container, exactly return a result pointer to chan
func (ctm *ContainerMonitor) Monit(dockerClient *client.Client, daemonURL, containerID, containerName, outputCol string, outputDB *data.DB, c chan *data.ContainerStat) {
	logger.Debugf("Container %s: Start monit task\n", containerName)
	if err := ctm.Init(dockerClient, daemonURL, containerID, containerName, outputCol, outputDB); err != nil {
		c <- nil
		logger.Errorf("Container %s: Error to init monitor\n", containerName)
		logger.Error(err)
		return
	}
	if s, err := ctm.CollectData(); err != nil {
		logger.Errorf("Container %s: Error to collect container data with daemon %s\n", containerName, daemonURL)
		logger.Error(err)
		c <- nil
	} else {
		c <- s
		if outputDB != nil && outputDB.URL != "" && outputDB.Name != "" && outputCol != "" {
			outputDB.SaveData(s, outputCol)
			logger.Debugf("Container %s: saved to db %s/%s/%s\n", containerName, outputDB.URL, outputDB.Name, outputCol)
		}
	}
	//return
}

//Init will finish the setup
//This should be call first before using any other method
func (ctm *ContainerMonitor) Init(dockerClient *client.Client, daemonURL, containerID, containerName, outputCol string, outputDB *data.DB) error {
	ctm.DaemonURL = daemonURL
	if dockerClient != nil {
		ctm.client = dockerClient
	} else {
		defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
		httpClient := http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 15 * time.Second,
				}).Dial,
				MaxIdleConnsPerHost: 64,
				DisableKeepAlives:   true, // use this to prevent many connections opened
			},
			Timeout: time.Duration(60) * time.Second,
		}
		cli, err := client.NewClient(daemonURL, "v1.22", &httpClient, defaultHeaders)
		if err != nil {
			logger.Errorf("Cannot init connection to docker host=%s\n", daemonURL)
			logger.Error(err)
			return err
		}
		ctm.client = cli
	}

	ctm.containerID = containerID
	ctm.containerName = containerName
	ctm.outputDB = outputDB
	return nil
}

// CollectData will collect info for a given container and store into db
// Will return pointer of the record struct
func (ctm *ContainerMonitor) CollectData() (*data.ContainerStat, error) {
	/*
		info, err := ctm.client.Info(context.Background())
		if err != nil {
			logger.Warningf("Cannot get info from docker host\n")
			return err
		}
	*/
	if ctm.client == nil {
		logger.Errorf("Container %s: docker client nil", ctm.containerName)
		return nil, errors.New("docker client nil")
	}

	monitStart := time.Now()
	monitTime := time.Now().Sub(monitStart)

	responseBody, err := ctm.client.ContainerStats(context.Background(), ctm.containerName, false)

	/*
		res, err := http.Get("http://"+ctm.DaemonURL[6:]+"/containers/"+ctm.containerID+"/stats?stream=0")
		if err != nil {
			logger.Errorf("Container %s: Error to get stats info\n", ctm.containerName)
			logger.Error(err)
			return nil, err
		}
		responseBody := res.Body
	*/

	monitTime = time.Now().Sub(monitStart)
	logger.Debugf("Container %s: api call used %s", ctm.containerName, monitTime)

	if responseBody != nil {
		defer responseBody.Close()
		defer ioutil.ReadAll(responseBody)
		//defer io.Copy(ioutil.Discard, responseBody)
	}

	if err != nil {
		logger.Errorf("Container %s: Daemon %s, Error to get stats", ctm.containerName, ctm.DaemonURL)
		return nil, err
	}

	dec := json.NewDecoder(responseBody)
	var v *types.StatsJSON

	if err := dec.Decode(&v); err != nil {
		dec = json.NewDecoder(io.MultiReader(dec.Buffered(), responseBody))
		logger.Warningf("Container %s: Error to decode stats info", ctm.containerName)
		return nil, err
	}

	var memPercent, cpuPercent = 0.0, 0.0
	var previousCPU, previousSystem uint64

	s := data.ContainerStat{
		ContainerID:      ctm.containerID,
		ContainerName:    ctm.containerName,
		CPUPercentage:    0.0,
		Memory:           0.0,
		MemoryLimit:      0.0,
		MemoryPercentage: 0.0,
		NetworkRx:        0.0,
		NetworkTx:        0.0,
		BlockRead:        0.0,
		BlockWrite:       0.0,
		PidsCurrent:      0,
		TimeStamp:        v.Read,
	}
	if v.MemoryStats.Limit != 0 {
		memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}

	previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
	previousSystem = v.PreCPUStats.SystemUsage
	cpuPercent = calculateCPUPercent(previousCPU, previousSystem, v)
	blkRead, blkWrite := calculateBlockIO(v.BlkioStats)
	s.CPUPercentage = cpuPercent
	s.Memory = float64(v.MemoryStats.Usage)
	s.MemoryLimit = float64(v.MemoryStats.Limit)
	s.MemoryPercentage = memPercent
	s.NetworkRx, s.NetworkTx = calculateNetwork(v.Networks)
	s.BlockRead = float64(blkRead)
	s.BlockWrite = float64(blkWrite)
	s.PidsCurrent = v.PidsStats.Current

	logger.Debugf("Container %s: collected data = %+v", ctm.containerName, s)
	return &s, nil
}

func calculateCPUPercent(previousCPU, previousSystem uint64, v *types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func calculateBlockIO(blkio types.BlkioStats) (blkRead uint64, blkWrite uint64) {
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = blkRead + bioEntry.Value
		case "write":
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return
}

func calculateNetwork(network map[string]types.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

// ListContainer will get all existing containers on the host
// @deprecated, just keep for testing
func (ctm *ContainerMonitor) ListContainer() ([]types.Container, error) {
	if ctm.client == nil {
		logger.Warning("Container client is not inited, pls Init first")
		return nil, errors.New("Container Client Not Inited")
	}
	filter := filters.NewArgs()
	filter.Add("label", "monitor=true")
	options := types.ContainerListOptions{All: true, Filter: filter}
	containers, err := ctm.client.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	//for _, c := range containers {
	//	logger.Debug(c)
	//}
	return containers, nil
}
