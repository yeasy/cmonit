package database

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

//Cluster is a document in the host collection
type Cluster struct {
	_ID           bson.ObjectId `bson:"_id,omitempty"`
	ConsensusType string        `bson:"consensus_type,omitempty"`
	UserID        []string      `bson:"user_id,omitempty"`
	HostID        []string      `bson:"host_id,omitempty"`
	Name          string        `bson:"name,omitempty"`
	CreateTS      string        `bson:"create_ts,omitempty"`
	ReleaseTS     string        `bson:"release_ts,omitempty"`
	ID            string        `bson:"id,omitempty"`
	APIURL        string        `bson:"api_url,omitempty"`
	Type          string        `bson:"type,omitempty"`
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
	HostID           string        `bson:"host_id,omitempty"`
	TimeStamp        time.Time     `bson:"timestamp,omitempty"`
}
