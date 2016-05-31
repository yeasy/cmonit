package data

import (
	"github.com/jmcvetta/napping"
)

// ESInsertDoc will insert a doc to elasticsearch
func ESInsertDoc(esURL, esIndex, esType string, doc map[string]interface{}) {

	result := make(map[string]string)
	url := "http://" + esURL + "/" + esIndex + "/" + esType
	resp, err := napping.Post(url, &doc, &result, nil)
	if err != nil {
		logger.Warningf("Error to send data to es=%s/%s/%s\n", esURL, esIndex, esType)
		logger.Warning(err)
	}
	if resp.Status() == 200 {
		logger.Debug(result)
	}
}
