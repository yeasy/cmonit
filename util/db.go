package util

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/op/go-logging"
	"errors"
)

var logger = logging.MustGetLogger("util")

type Host struct {
	    _ID        bson.ObjectId `bson:"_id,omitempty"`
        Name string
        Capacity int
        Daemon_URL string
        Status string
        Create_TS string
        ID string
        clusters []string
}

// A db, or a table in mongo
type DB struct {
	url string // mongo api url
	db_name string // name of the db
	col_host * mgo.Collection
	col_monitor * mgo.Collection
	session *mgo.Session
}

// Init a db, open session and make collection handler
func (db *DB) Init(db_url string, db_name string) (*mgo.Collection, error) {
	var err error
	db.session, err = mgo.Dial(db_url)
	if err != nil {
		return nil, err
	}
	// Optional. Switch the session to a monotonic behavior.
	db.session.SetMode(mgo.Monotonic, true)

	db.col_host = db.session.DB(db_name).C("host")
	db.col_monitor = db.session.DB(db_name).C("monitor")

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
	} else{
		err := db.col_host.Find(bson.M{}).All(&hosts)
		return hosts, err
	}
}
