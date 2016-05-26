package util

import (
	"errors"
	"time"

	"github.com/op/go-logging"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var logger = logging.MustGetLogger("util")

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
	HostID           string        `bson:"host_id,omitempty"`
	TimeStamp        time.Time     `bson:"timestamp,omitempty"`
}

// DB is a table in mongo
type DB struct {
	url     string // mongo api url
	dbName  string // name of the db
	session *mgo.Session
	cols    map[string]*mgo.Collection
}

// Init a db, open session and make collection handler
func (db *DB) Init(dbURL string, dbName string) error {
	var err error
	db.dbName = dbName
	if db.session, err = mgo.DialWithTimeout(dbURL, time.Duration(3*time.Second)); err != nil {
		logger.Errorf("Failed to dial mongo=%s\n", dbURL)
		return err
	}
	// Optional. Switch the session to a monotonic behavior.
	db.session.SetMode(mgo.Monotonic, true)
	db.cols = make(map[string]*mgo.Collection)

	return nil
}

// Close a db session
func (db *DB) Close() {
	db.session.Close()
}

// SetCol will set the cols points to collections
func (db *DB) SetCol(colKey, colName string) {
	db.cols[colKey] = db.session.DB(db.dbName).C(colName)
}

// SetIndex will set index property
func (db *DB) SetIndex(colKey, indexKey string, expireDays int) error {
	index := mgo.Index{
		Key:        []string{indexKey},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}
	if expireDays > 0 {
		logger.Debugf("Set collection %s expire after %d days\n", colKey, expireDays)
		index.ExpireAfter = time.Duration(expireDays) * 24 * time.Hour
	} else {
		logger.Warningf("Invalid expire = %d days, default to not expire\n", expireDays)
	}

	if err := db.cols[colKey].EnsureIndex(index); err != nil {
		logger.Warningf("Failed to set index properties on collection %s\n", colKey)
		return err
	}

	return nil
}

// GetHosts retrieve the hosts info from db
func (db *DB) GetHosts() (*[]Host, error) {
	var hosts []Host
	if h, ok := db.cols["host"]; ok {
		err := h.Find(bson.M{}).All(&hosts)
		return &hosts, err
	}
	logger.Warning("host collection handler is nil, should init first.")
	return &hosts, errors.New("db collection host is not opened")
}

// SaveData save a record into db's collection
func (db *DB) SaveData(s interface{}, colName string) error {
	if c, ok := db.cols[colName]; ok {
		if err := c.Insert(s); err != nil {
			logger.Warning("Error to insert data")
			logger.Error(err)
			return err
		}
		logger.Debugf("Saved data into %s.%s\n", db.dbName, colName)
		return nil
	}
	logger.Warning("collection handler is nil, should init first.")
	return errors.New("db collection is not opened")
}
