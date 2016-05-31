package data

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

//ContainerStat is a document of stat info for a container
type ContainerStat struct {
	_ID              bson.ObjectId `bson:"_id,omitempty"`
	ContainerID      string        `bson:"container_id,omitempty"`
	ContainerName    string        `bson:"container_name,omitempty"`
	CPUPercentage    float64       `bson:"cpu_percentage,omitempty"`
	Memory           float64       `bson:"memory_usage,omitempty"`
	MemoryLimit      float64       `bson:"memory_limit,omitempty"`
	MemoryPercentage float64       `bson:"memory_percentage,omitempty"`
	NetworkRx        float64       `bson:"network_rx,omitempty"`
	NetworkTx        float64       `bson:"network_tx,omitempty"`
	BlockRead        float64       `bson:"block_read,omitempty"`
	BlockWrite       float64       `bson:"block_write,omitempty"`
	PidsCurrent      uint64        `bson:"pid_current,omitempty"`
	TimeStamp        time.Time     `bson:"timestamp,omitempty"`
}
