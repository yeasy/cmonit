package util

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/op/go-logging"
	"errors"
	"time"
	"github.com/spf13/viper"
)

var logger = logging.MustGetLogger("util")

//a document in the host collection
type Host struct {
	_ID        bson.ObjectId `bson:"_id,omitempty"`
	ID         string `bson:"id,omitempty"`
	Name       string `bson:"name,omitempty"`
	Capacity   uint64 `bson:"capacity,omitempty"`
	Daemon_URL string `bson:"daemon_url,omitempty"`
	Status     string `bson:"status,omitempty"`
	Create_TS  string `bson:"create_ts,omitempty"`
	Clusters   []string `bson:"clusters,omitempty"`
}

//a document in the monitor collection
type MonitorStat struct {
	_ID              bson.ObjectId `bson:"_id,omitempty"`
	ContainerID      string `bson:"container_id,omitempty"`
	ContainerName    string `bson:"container_name,omitempty"`
	CPUPercentage    float64 `bson:"cpu_percentage,omitempty"`
	Memory           float64 `bson:"memory_usage,omitempty"`
	MemoryLimit      float64 `bson:"memory_limit,omitempty"`
	MemoryPercentage float64 `bson:"memory_percentage,omitempty"`
	NetworkRx        float64 `bson:"network_rx,omitempty"`
	NetworkTx        float64 `bson:"network_tx,omitempty"`
	BlockRead        float64 `bson:"block_read,omitempty"`
	BlockWrite       float64 `bson:"block_write,omitempty"`
	PidsCurrent      uint64 `bson:"pid_current,omitempty"`
	HostID           string `bson:"host_id,omitempty"`
	TimeStamp        time.Time `bson:"timestamp,omitempty"`
}

// A db, or a table in mongo
// A db, or a table in mongo
type DB struct {
	url         string // mongo api url
	db_name     string // name of the db
	col_host    *mgo.Collection
	col_monitor *mgo.Collection
	session     *mgo.Session
}

// Init a db, open session and make collection handler
func (db *DB) Init(db_url string, db_name string) (*mgo.Collection, error) {
	var err error
	db.session, err = mgo.Dial(db_url)
	if err != nil {
		logger.Errorf("Failed to dial mongo=%s\n", db_url)
		return nil, err
	}
	// Optional. Switch the session to a monotonic behavior.
	db.session.SetMode(mgo.Monotonic, true)

	db.col_host = db.session.DB(db_name).C("host")
	db.col_monitor = db.session.DB(db_name).C("monitor")
	index := mgo.Index{
		Key: []string{"container_id"},
		Unique: false,
		DropDups: false,
		Background: true,
		Sparse: true,
	}
	if monitor_expire := viper.GetInt("monitor.expire"); monitor_expire > 0 {
		logger.Debugf("Set monitor collection expire after %d days\n", monitor_expire)
		index.ExpireAfter = time.Duration(monitor_expire)*24*time.Hour
	}
	err = db.col_monitor.EnsureIndex(index)
	if err != nil{
		logger.Warning("Failed to set index properties on the monitor collection")
	}

	db.db_name = db_name

	return db.col_host, nil
}

// Close a db session
func (db *DB) Close() {
	db.session.Close()
}

// Retrieve the info from the host info collection
func (db *DB) GetHosts() ([]Host, error) {
	var hosts []Host
	if db.col_host == nil {
		logger.Warning("host collection handler is nil, should init first.")
		return hosts, errors.New("db collection host is not opened")
	} else {
		err := db.col_host.Find(bson.M{}).All(&hosts)
		return hosts, err
	}
}

//save a record into db
func (db *DB) SaveData(s *MonitorStat) {
	if err := db.col_monitor.Insert(s); err != nil {
		logger.Warning("Error to insert data")
		logger.Error(err)
	} else {
		logger.Debugf("Saved data for container=%s at host=%s\n", s.ContainerID, s.HostID)
	}
}
