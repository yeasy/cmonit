package data

import (
	"errors"
	"time"

	"github.com/op/go-logging"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var logger = logging.MustGetLogger("util")

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
	db.cols = make(map[string]*mgo.Collection, 4)

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

// GetCol retrieve the collection from db
//depreacted
func (db *DB) GetCol(colName string) (*[]interface{}, error) {
	var result []interface{}
	if c, ok := db.cols[colName]; ok {
		err := c.Find(bson.M{}).All(&result)
		return &result, err
	}
	logger.Warningf("collection handler %s is nil, should init first.\n", colName)
	return &result, errors.New("Cannot reach db collection " + colName)
}

// GetClusters retrieve the hosts info from db
func (db *DB) GetClusters(filter map[string]interface{}) (*[]Cluster, error) {
	var clusters []Cluster
	colName := "cluster"
	if c, ok := db.cols[colName]; ok {
		err := c.Find(filter).All(&clusters)
		return &clusters, err
	}
	logger.Warningf("collection handler %s is nil, should init first.\n", colName)
	return &clusters, errors.New("Cannot reach db collection " + colName)
}

// GetHosts retrieve the hosts info from db
func (db *DB) GetHosts() (*[]Host, error) {
	var hosts []Host
	colName := "host"
	if h, ok := db.cols[colName]; ok {
		err := h.Find(bson.M{}).All(&hosts)
		return &hosts, err
	}
	logger.Warningf("collection handler %s is nil, should init first.\n", colName)
	return &hosts, errors.New("Cannot reach db collection " + colName)
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
