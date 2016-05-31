package data

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

//Cluster is a document in the host collection
type Cluster struct {
	_ID           bson.ObjectId     `bson:"_id,omitempty"`
	ReleaseTS     time.Time         `bson:"release_ts,omitempty"`
	ConsensusType string            `bson:"consensus_type,omitempty"`
	HostID        []string          `bson:"host_id,omitempty"`
	UserID        []string          `bson:"user_id,omitempty"`
	CreateTS      time.Time         `bson:"create_ts,omitempty"`
	Name          string            `bson:"name,omitempty"`
	ID            string            `bson:"id,omitempty"`
	Containers    map[string]string `bson:"containers,omitempty"`
	APIURL        string            `bson:"api_url,omitempty"`
	DaemonURL     string            `bson:"daemon_url,omitempty"`
}

//ClusterStat is a document of stat info for a cluster
type ClusterStat struct {
	_ID              bson.ObjectId `bson:"_id,omitempty"`
	ClusterID        string        `bson:"cluster_id,omitempty"`
	ClusterName      string        `bson:"cluster_name,omitempty"`
	CPUPercentage    float64       `bson:"cpu_percentage,omitempty"`
	Memory           float64       `bson:"memory_usage,omitempty"`
	MemoryLimit      float64       `bson:"memory_limit,omitempty"`
	MemoryPercentage float64       `bson:"memory_percentage,omitempty"`
	NetworkRx        float64       `bson:"network_rx,omitempty"`
	NetworkTx        float64       `bson:"network_tx,omitempty"`
	BlockRead        float64       `bson:"block_read,omitempty"`
	BlockWrite       float64       `bson:"block_write,omitempty"`
	PidsCurrent      uint64        `bson:"pid_current,omitempty"`
	Size             uint64        `bson:"size,omitempty"`
	MaxLatency       float64       `bson:"max_latency,omitempty"`
	MinLatency       float64       `bson:"min_latency,omitempty"`
	AvgLatency       float64       `bson:"avg_latency,omitempty"`
	Latencies        []float64     `bson:"latencies,omitempty"`
	TimeStamp        time.Time     `bson:"timestamp,omitempty"`
}

// CalculateStat will get the stat result for a cluster
func (s *ClusterStat) CalculateStat(csList []*ContainerStat) {
	number := len(csList)
	for _, cs := range csList {
		s.CPUPercentage += cs.CPUPercentage
		s.Memory += cs.Memory
		s.MemoryLimit += cs.MemoryLimit
		s.MemoryPercentage += cs.MemoryPercentage
		s.NetworkRx += cs.NetworkRx
		s.NetworkTx += cs.NetworkTx
		s.BlockRead += cs.BlockRead
		s.BlockWrite += cs.BlockWrite
		s.PidsCurrent += cs.PidsCurrent
		s.Size = uint64(number)
	}
}
