// main.go
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"os"
	"os/exec"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	//"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "fastdfs"
)

type FastDFSData struct {
	groupCount    int
	initState     int
	syncState     int
	waitSyncState int
	activeState   int
	deletedState  int
	offlineState  int
	onlineState   int
}

var (
	nodeLabels = []string{"node"}
	groupCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "group_count"),
		"How many group counts were up at the last query.",
		nodeLabels, nil,
	)
	initState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "init_state"),
		"How many nodes were on init_state at the last query.",
		nodeLabels, nil,
	)
	syncState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "sync_state"),
		"How many nodes were on sync_state at the last query.",
		nodeLabels, nil,
	)
	waitSyncState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "wait_sync_state"),
		"How many nodes were on wait_sync_state at the last query.",
		nodeLabels, nil,
	)
	activeState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "active_state"),
		"How many nodes were on active_state at the last query.",
		nodeLabels, nil,
	)
	deletedState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "deleted_state"),
		"How many nodes were on deleted_state at the last query.",
		nodeLabels, nil,
	)
	offlineState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "offline_state"),
		"How many nodes were on offline_state at the last query.",
		nodeLabels, nil,
	)
	onlineState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "online_state"),
		"How many nodes were on online_state at the last query.",
		nodeLabels, nil,
	)
)

type Exporter struct {
	podname string
}

func NewExporter(podname string) (*Exporter, error) {
	return &Exporter{
		podname: podname,
	}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- groupCount
	ch <- initState
	ch <- syncState
	ch <- waitSyncState
	ch <- activeState
	ch <- deletedState
	ch <- offlineState
	ch <- onlineState
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	fastData := FastDFSData{}
	parseFastDFSCommand(&fastData)
	ch <- prometheus.MustNewConstMetric(
		groupCount, prometheus.GaugeValue, float64(fastData.groupCount), "groupCount",
	)
	ch <- prometheus.MustNewConstMetric(
		activeState, prometheus.GaugeValue, float64(fastData.activeState), "activeState",
	)
	ch <- prometheus.MustNewConstMetric(
		waitSyncState, prometheus.GaugeValue, float64(fastData.waitSyncState), "waitSyncState",
	)
}

func execFastDFSCommand(fastData *FastDFSData) {
	stdoutBuffer1 := &bytes.Buffer{}
	fastdfsExec1 := exec.Command("bash", "groupcount.sh")
	fastdfsExec1.Stdout = stdoutBuffer1
	err1 := fastdfsExec1.Run()
	if err1 != nil {
		fmt.Printf("bash groupcount.sh got error: %v", err1)
	}
	aa, _ := strconv.Atoi(strings.Replace(stdoutBuffer1.String(), "\n", "", -1))
	fastData.groupCount = aa

	stdoutBuffer2 := &bytes.Buffer{}
	fastdfsExec2 := exec.Command("bash", "active.sh")
	fastdfsExec2.Stdout = stdoutBuffer2
	err2 := fastdfsExec2.Run()
	if err2 != nil {
		fmt.Printf("bash active.sh got error: %v", err2)
	}
	bb, _ := strconv.Atoi(strings.Replace(stdoutBuffer2.String(), "\n", "", -1))
	fastData.activeState = bb

	stdoutBuffer3 := &bytes.Buffer{}
	fastdfsExec3 := exec.Command("bash", "wait.sh")
	fastdfsExec3.Stdout = stdoutBuffer3
	err3 := fastdfsExec3.Run()
	if err3 != nil {
		fmt.Printf("bash wait.sh got error: %v", err3)
	}
	cc, _ := strconv.Atoi(strings.Replace(stdoutBuffer3.String(), "\n", "", -1))
	fastData.waitSyncState = cc

}

func parseFastDFSCommand(fastData *FastDFSData) {
	fastdfsExec := exec.Command("kubectl", "exec", "fastdfs-group0-storage0-0", "/usr/bin/fdfs_monitor", "/etc/fdfs/storage.conf")
	outfile, fileerr := os.Create("./out.txt")
	if fileerr != nil {
		fmt.Println("Error create out.txt")
	}
	defer outfile.Close()

	stdoutPipe, err := fastdfsExec.StdoutPipe()
	writer := bufio.NewWriter(outfile)
	err = fastdfsExec.Start()
	if err != nil {
		fmt.Println("fastdfsExec kubectl exec Error")
	}
	io.Copy(writer, stdoutPipe)
	defer writer.Flush()
	fastdfsExec.Wait()
	execFastDFSCommand(fastData)
}

func init() {
	prometheus.MustRegister(version.NewCollector("fastdfs_exporter"))
}

func main() {

	var (
		podname       = kingpin.Flag("podname", "Pod Name For FastDFS.").Default("fastdfs-group0-storage0-0").String()
		metricsPath   = kingpin.Flag("web.path", "Path under which to expose metrics.").Default("/metrics").String()
		listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":10000").String()
		num           int
	)

	//log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("fastdfs_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	//log.Infoln("Starting fastdfs_exporter", version.Info())
	//log.Infoln("Build context", version.BuildContext())

	exporter, err := NewExporter(*podname)
	if err != nil {
		//log.Errorf("Creating new Exporter went wrong, ... \n%v", err)
		fmt.Printf("Creating new Exporter went wrong, ... \n%v", err)
	}
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		num, err = w.Write([]byte(`<html>
			<head><title>FastDFS Exporter v` + version.Version + `</title></head>
			<body>
			<h1>FastDFS Exporter v` + version.Version + `</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			//log.Fatal(num, err)
		}
	})

	//log.Infoln("Listening on", *listenAddress)
	fmt.Println("Listening on", *listenAddress)
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		//log.Fatal(err)
	}

}
