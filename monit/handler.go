package monit

import (
	"github.com/op/go-logging"
	"github.com/yeasy/cmonit/util"
)

var logger = logging.MustGetLogger("monit")

type Monitor interface {
	CollectData(db *util.DB) (map[string]interface{}, error)
}
