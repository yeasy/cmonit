package agent

import (
	"github.com/op/go-logging"
	"github.com/yeasy/cmonit/database"
)

var logger = logging.MustGetLogger("monit")

// Monitor is used to collect data
type Monitor interface {
	CollectData(db *database.DB) (map[string]interface{}, error)
}
