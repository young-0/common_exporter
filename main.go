// __author__ = 'jean'
// Created by BJ0628 at 2020/7/3 16:37
// Process by go

package main

import (
	"context"
	"flag"
	"fmt"
	log "github.com/golang/glog"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"time"
)

type threadStruct struct {
	cancel         context.CancelFunc
	localTarget    string // local exporter process running 127.0.0.1:port
	localPort      int    // local exporter process running port
	lastScrapeTime int64  // last scrape time
}

type HandleFnc func(writer http.ResponseWriter, request *http.Request)

var (
	threadList map[string]*threadStruct = make(map[string]*threadStruct) // store slave process info
	paramList  seqStringFlag                                             // slave exporter params list
	waitgroup  sync.WaitGroup

	version             = flag.Bool("version", false, "Print mtail version information.")
	bindAddr            = flag.String("bind-addr", ":8080", "master bind address for the metrics server")
	binFile             = flag.String("exporter-bin-file", "", "slave exporter bin file addr, like /bin/kafka_exporter")
	exporterMonitorAddr = flag.String("exporter-monitor-addr", "", "slave exporter monitor addr, like \"--es.uri=http://%s\", %s is target ip:port")
	exporterListenAddr  = flag.String("exporter-listen-addr", "--web.listen-address=:%d", "slave exporter listen addr, like \"--web.listen-address=:%d\", %d is listen port")

	// Branch as well as Version and Revision identifies where in the git
	// history the build came from, as supplied by the linker when copmiled
	// with `make'.  The defaults here indicate that the user did not use
	// `make' as instructed.
	Branch   string = "invalid:-use-make-to-build"
	Version  string = "invalid:-use-make-to-build"
	Revision string = "invalid:-use-make-to-build"
)

func init() {
	flag.Var(&paramList, "params", "slave exporter params list, may be used multiple times or null; except monitor-addr and listen-addr。")
}

func main() {
	buildInfo := BuildInfo{
		Branch:   Branch,
		Version:  Version,
		Revision: Revision,
	}

	flag.Parse()
	if *version {
		fmt.Println(buildInfo.String())
		os.Exit(0)
	}
	if *binFile == "" || *exporterMonitorAddr == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	var srv http.Server
	idleConnsClosed := make(chan struct{})
	processClearClosed := make(chan int)

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/scrape", logPanics(scrapeHandler))
	srv = http.Server{
		Addr:    *bindAddr,
		Handler: mux,
	}

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Errorf("HTTP server Shutdown: %v", err)
		} else {
			log.Info("Stopping http server...")
		}
		// Stop slave process
		for target, ths := range threadList {
			ths.cancel()
			log.Infof("Stopping slave process: %s", target)
		}
		processClearClosed <- 0
		waitgroup.Wait()
		log.Info("See you next time!")
		close(idleConnsClosed)
	}()
	go ProcessClear(&threadList, processClearClosed)
	log.Infoln("Start main process")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v ", err)
	}
	<-idleConnsClosed
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	// 获取target参数，IP:PORt
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		return
	}
	ipPort := strings.Split(target, ":")
	ip := net.ParseIP(ipPort[0])
	if ip == nil {
		http.Error(w, fmt.Sprintf("Invalid 'target' parameter, parse err: %s ", target), 400)
		return
	}
	log.V(2).Infof("scrape target=%s ", target)

	if ths, found := threadList [target]; found {
		metricsUrl := "http://" + ths.localTarget + "/metrics"
		w.Write(getMetrics(metricsUrl, target))
		ths.lastScrapeTime = time.Now().Unix()
	} else {
		port := getPort(&threadList)
		ctx, cancel := context.WithCancel(context.Background())
		threadList [target] = &threadStruct{
			cancel:         cancel,
			localTarget:    fmt.Sprintf("127.0.0.1:%d", port),
			localPort:      port,
			lastScrapeTime: time.Now().Unix(),
		}
		go exporterThread(target, port, &ctx)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("registration success"))
	}
}

func getPort(threadList *map[string]*threadStruct) int {
	port := 18081 + len(*threadList)
	for {
		var flag int = 0
		for _, ths := range *threadList {
			if port == ths.localPort {
				flag = 1
				break
			}
		}
		if flag == 1 {
			port += 1
		} else {
			return port
		}
	}
}
func getMetrics(metricsUrl string, targetHost string) []byte {
	r, err := http.Get(metricsUrl)
	if err != nil {
		panic(err)
	}
	defer func() { _ = r.Body.Close() }()

	body, _ := ioutil.ReadAll(r.Body)
	return body
}

func exporterThread(target string, port int, ctx *context.Context) {
	log.Infof("Start New threading :%d for %s ", port, target)
	params := []string{fmt.Sprintf(*exporterMonitorAddr, target), fmt.Sprintf(*exporterListenAddr, port),}
	params = append(params, paramList...)
	// cmd := exec.CommandContext(*ctx, binFile, fmt.Sprintf("--web.listen-address=:%d", port), "--kafka.server="+target)
	log.V(2).Infof("%s %v", *binFile, params)
	cmd := exec.CommandContext(*ctx, *binFile, params...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	waitgroup.Add(1)
	err := cmd.Run()
	waitgroup.Done()
	if err != nil {
		log.Errorln("Execute Command failed:" + err.Error())
		return
	}
}

/*
1. 清除失败的exporter thread
2. 清除长期不用的exporter thread
*/
func ProcessClear(threadList *map[string]*threadStruct, quit chan int) {
	d := time.Duration(time.Minute * 2)
	t := time.NewTicker(d)
	defer t.Stop()
	waitgroup.Add(1)
	for {
		select {
		case <-t.C:
			for target, ths := range *threadList {
				// 清除失败的exporter thread
				metricsUrl := "http://" + ths.localTarget + "/metrics"
				r, err := http.Head(metricsUrl)
				if err != nil || r.StatusCode != 200 {
					delete(*threadList, target)
					ths.cancel()
					log.Infof("exporter for %s test failed, delete ", target)
				}
				// 10分钟没有被抓取数据，删除exporter
				unixTime := time.Now().Unix()
				if unixTime-ths.lastScrapeTime > 600 {
					delete(*threadList, target)
					ths.cancel()
					log.Infof("exporter for %s test failed, delete ", target)
				}
			}
		case <-quit:
			log.Info("Stopping ProcessClear manager....")
			waitgroup.Done()
			return
		}
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html>
                <head><title>kafka Master Exporter</title></head>
                <body>
                <h1>kafka Master Exporter</h1>
                <p>/scrape?target=ip:port</p>
                </body>
                </html>`))
}
func logPanics(function HandleFnc) HandleFnc {
	return func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				log.Errorf("[%v] caught panic: %v", request.RemoteAddr, x)
				// 默认出现 panic 只会记录日志，页面就是一个无任何输出的白页面，
				// 可以给页面一个错误信息，如下面的示例返回了一个 500
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		function(writer, request)
	}
}

type seqStringFlag []string

func (f *seqStringFlag) String() string {
	return fmt.Sprint(*f)
}

func (f *seqStringFlag) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		*f = append(*f, v)
	}
	return nil
}

type BuildInfo struct {
	Branch   string
	Version  string
	Revision string
}

func (b BuildInfo) String() string {
	return fmt.Sprintf(
		"common_exporter version %s git revision %s go version %s go arch %s go os %s",
		b.Version,
		b.Revision,
		runtime.Version(),
		runtime.GOARCH,
		runtime.GOOS,
	)
}
