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
	waitSyncState    int
	activeState      int
}

type FastDFSConfig struct {
	ApiserverAddress string
	PodName          string
	NameSpace        string
}

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

	config        FastDFSConfig
	defaultConfig = FastDFSConfig{
		ApiserverAddress: "http://localhost:8080",
		PodName:          "fastdfs",
		NameSpace:        "default",
	}
)

func NewExporter(podname string) (*Exporter, error) {
	return &Exporter{
		podname: podname,
	}, nil
}

func initConfig() {
	config = defaultConfig
	if apiserver := os.Getenv("APISERVER"); apiserver != "" {
		config.ApiserverAddress = apiserver
	}
	if podName := os.Getenv("FASTDFS_POD_NAME"); podName != "" {
		config.PodName = podName
	}
	if nameSpace := os.Getenv("NAMESPACE"); nameSpace != "" {
		config.NameSpace = nameSpace
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- configGroupNum
	ch <- configStorageNum
	ch <- groupCount
	ch <- waitSyncState
	ch <- activeState
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
	fastdfsExec1 := exec.Command("sh", "groupcount.sh")
	fastdfsExec1.Stdout = stdoutBuffer1
	err1 := fastdfsExec1.Run()
	if err1 != nil {
		log.Error(err1)
	}
	aa, _ := strconv.Atoi(strings.Replace(stdoutBuffer1.String(), "\n", "", -1))
	fastData.groupCount = aa

	stdoutBuffer2 := &bytes.Buffer{}
	fastdfsExec2 := exec.Command("sh", "active.sh")
	fastdfsExec2.Stdout = stdoutBuffer2
	err2 := fastdfsExec2.Run()
	if err2 != nil {
		log.Error(err2)
	}
	bb, _ := strconv.Atoi(strings.Replace(stdoutBuffer2.String(), "\n", "", -1))
	fastData.activeState = bb

	stdoutBuffer3 := &bytes.Buffer{}
	fastdfsExec3 := exec.Command("sh", "wait.sh")
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
	fastdfsExec := exec.Command("kubectl", "-s", config.ApiserverAddress, "exec", config.PodName, "cat", "/etc/fdfs/FastDFS.json", "-n", config.NameSpace)
	fastdfsExec.Stdout = stdoutBuffer
	err := fastdfsExec.Run()
	if err != nil {
		log.Error(err)
	}
	configDataParse(stdoutBuffer, fastData)

}

func parseFastDFSCommand(fastData *FastDFSData) {
	log.Infoln("Config ", config)
	fastdfsExec := exec.Command("kubectl", "-s", config.ApiserverAddress, "exec", config.PodName, "/usr/bin/fdfs_monitor", "/etc/fdfs/storage.conf", "-n", config.NameSpace)
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
		listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":10000").String()
		num           int
	)
	initConfig()
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("fastdfs_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting fastdfs_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := NewExporter("fastdfs")
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
			<p><a href='` + "/metrics" + `'>Metrics</a></p>
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
