package test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	_ "github.com/docker/engine-api/types/filters"
	"github.com/op/go-logging"
	"golang.org/x/net/context"

	_ "github.com/yeasy/cmonit/agent"
	"github.com/yeasy/cmonit/data"
)

var logger = logging.MustGetLogger("test")

func TestDockerAPI(t *testing.T) {
	// test stuff here...
	number := 10
	var names = []string{
		"575e44a8414b0507cccb9c52",
		"575e44a8414b0507cccb9c53",
		"575e44a8414b0507cccb9c56",
		"575e44a8414b0507cccb9c55",
		"575e44a8414b0507cccb9c54",
		"575e44a8414b0507cccb9c59",
		"575e44a8414b0507cccb9c57",
		"575e44a8414b0507cccb9c5b",
		"575e44a8414b0507cccb9c5a",
	}
	daemonURL := "tcp://192.168.7.62:2375"
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	httpClient := http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 15 * time.Second,
			}).Dial,
			MaxIdleConnsPerHost: 64,
			DisableKeepAlives:   true, // use this to prevent many connections opened
		},
		Timeout: time.Duration(15) * time.Second,
	}
	cli, err := client.NewClient(daemonURL, "v1.22", &httpClient, defaultHeaders)
	if err != nil {
		logger.Errorf("Cannot init connection to docker host=%s\n", daemonURL)
		logger.Error(err)
		return
	}

	ct := make(chan *data.ContainerStat, number)
	defer close(ct)
	monitStart := time.Now()
	monitTime := time.Now().Sub(monitStart)
	for i := 0; i < number; i++ {
		name := names[rand.Intn(len(names))] + "_vp" + strconv.Itoa(rand.Intn(4))
		//go tryAPIStats(cli, daemonURL, name, ct)
		go tryAPIList(cli, daemonURL, name, ct)
		//ctm := new(agent.ContainerMonitor)
		//go ctm.Monit(cli, daemonURL, "container_id", name, "", nil, ct)
	}

	get := 0
	for range ct {
		get++
		monitTime = time.Now().Sub(monitStart)
		if get >= number {
			break
		}
	}
	monitTime = time.Now().Sub(monitStart)
	logger.Infof("time used %s\n", monitTime)
	time.Sleep(time.Second)
	logger.Info("Done")
}

/*
func BenchmarkDockerAPI(b *testing.B) {
    for i := 0; i < b.N; i++ {
	    c := make(chan int)
    }
}
*/

func tryAPIList(cli *client.Client, daemonURL, containerName string, c chan *data.ContainerStat) error {
	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		logger.Error(err)
	}
	_ = containers
	return nil
}

func tryAPIStats(cli *client.Client, daemonURL, containerName string, c chan *data.ContainerStat) error {

	client := cli

	responseBody, err := client.ContainerStats(context.Background(), containerName, false)

	if responseBody != nil {
		defer responseBody.Close()
		defer ioutil.ReadAll(responseBody)
		//defer io.Copy(ioutil.Discard, responseBody)
	}

	if err != nil {
		logger.Errorf("Error to get stats info for %s\n", containerName)
		c <- nil
		return err
	}
	dec := json.NewDecoder(responseBody)
	var v *types.StatsJSON

	if err := dec.Decode(&v); err != nil {
		dec = json.NewDecoder(io.MultiReader(dec.Buffered(), responseBody))
		logger.Warningf("Error to decode stats info for container = %s\n", containerName)
		c <- nil
		return err
	}

	c <- nil
	return nil
}
