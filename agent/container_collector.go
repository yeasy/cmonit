package agent

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/yeasy/cmonit/database"
	"golang.org/x/net/context"
)

//ContainersMonitor is used to collect data from a docker host
type ContainerMonitor struct {
	client *client.Client
	containerID string
	colName string
	output *database.DB
}

//Init will finish the setup
//This should be call first before using any other method
func (cm *ContainerMonitor) Init(daemonURL, containerID, colName string, output *database.DB) error {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient(daemonURL, "", nil, defaultHeaders)
	if err != nil {
		logger.Warningf("Cannot connect to docker host=%s\n", daemonURL)
		return err
	}
	logger.Debugf("Connectted to docker host=%s\n", daemonURL)
	cm.client = cli
	cm.containerID = containerID
	cm.colName = colName
	cm.output = output
	return nil
}

// ListContainer will get all existing containers on the host
// @deprecated, just keep for testing
func (cm *ContainerMonitor) ListContainer() ([]types.Container, error) {
	if cm.client == nil {
		logger.Warning("Container client is not inited, pls Init first")
		return nil, errors.New("Container Client Not Inited")
	}
	filter := filters.NewArgs()
	filter.Add("label", "monitor=true")
	options := types.ContainerListOptions{All: true, Filter: filter}
	containers, err := cm.client.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	//for _, c := range containers {
	//	logger.Debug(c)
	//}
	return containers, nil
}

// CollectDataForContainer will collect info for a given container and store into db
// Will return pointer of the record struct
func (cm *ContainerMonitor) CollectData() (*database.ContainerStat, error) {
	logger.Debugf("stats container=%s\n", cm.containerID)
	responseBody, err := cm.client.ContainerStats(context.Background(), cm.containerID, false)

	if err != nil {
		logger.Warningf("Error to get stats info for %s\n", cm.containerID)
		return nil, err
	}
	//responseBody, err := ioutil.ReadAll(result)

	//if err != nil {
	//	logger.Error(err.Error())
	//	return
	//}
	defer responseBody.Close()
	dec := json.NewDecoder(responseBody)
	var v *types.StatsJSON

	if err := dec.Decode(&v); err != nil {
		dec = json.NewDecoder(io.MultiReader(dec.Buffered(), responseBody))
		logger.Warningf("Error to decode stats info for container = %s\n", cm.containerID)
		return nil, err
	}

	var memPercent = 0.0
	var cpuPercent = 0.0
	var previousCPU uint64
	var previousSystem uint64

	s := database.ContainerStat{
		ContainerID:      cm.containerID,
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

	logger.Debugf("stats = %v\n", s)
	cm.output.SaveData(&s, cm.colName)
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
