package database

import (
	"time"

	"math"

	"gopkg.in/mgo.v2/bson"
)

//Host is a document in the host collection
type Host struct {
	_ID       bson.ObjectId `bson:"_id,omitempty"`
	DaemonURL string        `bson:"daemon_url,omitempty"`
	Clusters  []string      `bson:"clusters,omitempty"`
	Name      string        `bson:"name,omitempty"`
	Status    string        `bson:"status,omitempty"`
	Capacity  uint64        `bson:"capacity,omitempty"`
	CreateTS  string        `bson:"create_ts,omitempty"`
	ID        string        `bson:"id,omitempty"`
	Type      string        `bson:"type,omitempty"`
}

//HostStat is a document of stat info for a cluster
type HostStat struct {
	_ID              bson.ObjectId `bson:"_id,omitempty"`
	HostID           string        `bson:"host_id,omitempty"`
	HostName         string        `bson:"host_name,omitempty"`
	CPUPercentage    float64       `bson:"cpu_percentage,omitempty"`
	Memory           float64       `bson:"memory_usage,omitempty"`
	MemoryLimit      float64       `bson:"memory_limit,omitempty"`
	MemoryPercentage float64       `bson:"memory_percentage,omitempty"`
	NetworkRx        float64       `bson:"network_rx,omitempty"`
	NetworkTx        float64       `bson:"network_tx,omitempty"`
	BlockRead        float64       `bson:"block_read,omitempty"`
	BlockWrite       float64       `bson:"block_write,omitempty"`
	PidsCurrent      uint64        `bson:"pid_current,omitempty"`
	AvgLatency       float64       `bson:"avg_latency,omitempty"`
	MaxLatency       float64       `bson:"max_latency,omitempty"`
	MinLatency       float64       `bson:"min_latency,omitempty"`
	TimeStamp        time.Time     `bson:"timestamp,omitempty"`
}

// CalculateStat will get the stat result for a cluster
func (s *HostStat) CalculateStat(csList []*ClusterStat) {
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
		s.AvgLatency += cs.AvgLatency
		s.MaxLatency = math.Max(s.MaxLatency, cs.MaxLatency)
		if cs.MinLatency < s.MinLatency || s.MinLatency == 0.0 {
			s.MinLatency = cs.MinLatency
		}
	}
	s.AvgLatency /= float64(number)
}
