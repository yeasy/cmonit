package database

import (
	"time"

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
	AvgLatency      float64        `bson:"avg_latency,omitempty"`
	MaxLatency      float64        `bson:"max_latency,omitempty"`
	MinLatency      float64        `bson:"min_latency,omitempty"`
	TimeStamp        time.Time     `bson:"timestamp,omitempty"`
}
