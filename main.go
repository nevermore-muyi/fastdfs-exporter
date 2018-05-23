// main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"os"
	"os/exec"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "fastdfs"
)

type FastDFSData struct {
	configGroupNum   int
	configStorageNum int
	groupCount       int
	initState        int
	syncState        int
	waitSyncState    int
	activeState      int
	deletedState     int
	offlineState     int
	onlineState      int
}

var (
	nodeLabels     = []string{"node"}
	configGroupNum = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "config_group_count"),
		"How many group counts were int the config file.",
		nodeLabels, nil,
	)
	configStorageNum = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "config_storage_num"),
		"How many storage were up int the config file.",
		nodeLabels, nil,
	)
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

type ConfigInfoJSON struct {
	Node_Hosts         []string `json:"node_Hosts"`
	Nginx_IP           string   `json:"nginx_IP"`
	Tracker_Server_Num int      `json:"tracker_Server_Num"`
	Group_Num          int      `json:"group_Num"`
	Storage_Num        int      `json:"storage_Num"`
	FastDfs_Data       int      `json:"fastDfs_Data"`
}

func NewExporter(podname string) (*Exporter, error) {
	return &Exporter{
		podname: podname,
	}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- configGroupNum
	ch <- configStorageNum
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
		configGroupNum, prometheus.GaugeValue, float64(fastData.configGroupNum), namespace,
	)
	ch <- prometheus.MustNewConstMetric(
		configStorageNum, prometheus.GaugeValue, float64(fastData.configStorageNum), namespace,
	)
	ch <- prometheus.MustNewConstMetric(
		groupCount, prometheus.GaugeValue, float64(fastData.groupCount), namespace,
	)
	ch <- prometheus.MustNewConstMetric(
		activeState, prometheus.GaugeValue, float64(fastData.activeState), namespace,
	)
	ch <- prometheus.MustNewConstMetric(
		waitSyncState, prometheus.GaugeValue, float64(fastData.waitSyncState), namespace,
	)
}

func execFastDFSCommand(fastData *FastDFSData) {
	stdoutBuffer1 := &bytes.Buffer{}
	fastdfsExec1 := exec.Command("bash", "groupcount.sh")
	fastdfsExec1.Stdout = stdoutBuffer1
	err1 := fastdfsExec1.Run()
	if err1 != nil {
		log.Error(err1)
	}
	aa, _ := strconv.Atoi(strings.Replace(stdoutBuffer1.String(), "\n", "", -1))
	fastData.groupCount = aa

	stdoutBuffer2 := &bytes.Buffer{}
	fastdfsExec2 := exec.Command("bash", "active.sh")
	fastdfsExec2.Stdout = stdoutBuffer2
	err2 := fastdfsExec2.Run()
	if err2 != nil {
		log.Error(err2)
	}
	bb, _ := strconv.Atoi(strings.Replace(stdoutBuffer2.String(), "\n", "", -1))
	fastData.activeState = bb

	stdoutBuffer3 := &bytes.Buffer{}
	fastdfsExec3 := exec.Command("bash", "wait.sh")
	fastdfsExec3.Stdout = stdoutBuffer3
	err3 := fastdfsExec3.Run()
	if err3 != nil {
		log.Error(err3)
	}
	cc, _ := strconv.Atoi(strings.Replace(stdoutBuffer3.String(), "\n", "", -1))
	fastData.waitSyncState = cc

}

func configDataParse(cmdOutBuff io.Reader, fastData *FastDFSData) {
	var config ConfigInfoJSON
	b, err := ioutil.ReadAll(cmdOutBuff)
	if err != nil {
		log.Error(err)
	}
	err = json.Unmarshal(b, &config)

	aa := config.Group_Num
	bb := config.Storage_Num
	fastData.configGroupNum = aa
	fastData.configStorageNum = bb
}

func execFastConfigCommand(fastData *FastDFSData) {
	stdoutBuffer := &bytes.Buffer{}
	fastdfsExec := exec.Command("kubectl", "exec", "fastdfs-group0-storage0-0", "cat", "/etc/fdfs/FastDFS.json")
	fastdfsExec.Stdout = stdoutBuffer
	err := fastdfsExec.Run()
	if err != nil {
		log.Error(err)
	}
	configDataParse(stdoutBuffer, fastData)

}

func parseFastDFSCommand(fastData *FastDFSData) {
	fastdfsExec := exec.Command("kubectl", "exec", "fastdfs-group0-storage0-0", "/usr/bin/fdfs_monitor", "/etc/fdfs/storage.conf")
	outfile, fileerr := os.Create("./out.txt")
	if fileerr != nil {
		log.Error(fileerr)
	}
	defer outfile.Close()

	stdoutPipe, err := fastdfsExec.StdoutPipe()
	writer := bufio.NewWriter(outfile)
	err = fastdfsExec.Start()
	if err != nil {
		log.Error(err)
	}
	io.Copy(writer, stdoutPipe)
	defer writer.Flush()
	fastdfsExec.Wait()
	execFastConfigCommand(fastData)
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

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("fastdfs_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting fastdfs_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := NewExporter(*podname)
	if err != nil {
		log.Errorf("Creating new Exporter went wrong, ... \n%v", err)
	}
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		num, err = w.Write([]byte(`<html>
			<head><title>FastDFS Exporter` + version.Version + `</title></head>
			<body>
			<h1>FastDFS Exporter v` + version.Version + `</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			log.Fatal(num, err)
		}
	})

	log.Infoln("Listening on", *listenAddress)
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}

}
