package agent

import (
	"github.com/op/go-logging"
	"github.com/yeasy/cmonit/data"
)

var logger = logging.MustGetLogger("monit")

// Monitor is used to collect data
type Monitor interface {
	CollectData(db *data.DB) (map[string]interface{}, error)
}
