package agent

import "github.com/yeasy/cmonit/database"

// ClusterMonitor is used to collect data from a whole docker host.
// It may include many clusters
type ClusterMonitor struct {
	cluster *database.Cluster //cluster collection
	output *database.DB  //save out
}

func (clm *ClusterMonitor) Init (cluster *database.Cluster, output *database.DB) error {
	clm.cluster = cluster
	clm.output = output

	return nil
}

// CollectData will collect information from docker host
func (clm *ClusterMonitor) CollectData(c chan *database.ClusterStat) error {
	//for each container, collect result
	cs := database.ClusterStat {

	}
	c <- &cs
	return nil
}

// ClusterMonitTask will return pointer of result to the channel
func ClusterMonitTask (cluster *database.Cluster, output *database.DB, c chan database.ClusterStat) error {
	clm := new(ClusterMonitor)
	if err := clm.Init(cluster, output); err != nil {
		c <- nil
		return err
	}
	if err := clm.CollectData(c); err != nil {
		c <- nil
		return err
	}
	return nil
}

