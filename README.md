# A tool: convert a single exporter into multi-instance monitoring

Most prometheus exporter tools only monitor a single instance. In prometheus in k8s, I want to automatically discover and monitor multiple instances.

I don't have much time to modify each exporter, and change each exporter from single-instance monitoring to multi-instance monitoring. Because this project was created.

He is very easy to use. In my current environment, I use it to monitor kafka, zookeeper, elasticsearch, mongo, and run in a k8s environment.
```
  -alsologtostderr
        log to standard error as well as files
  -bind-addr string
        master bind address for the metrics server (default ":8080")
  -exporter-bin-file string
        slave exporter bin file addr, like /bin/kafka_exporter
  -exporter-listen-addr string
        slave exporter listen addr, like "--web.listen-address=:%d", %d is listen port (default "--web.listen-address=:%d")
  -exporter-monitor-addr string
        slave exporter monitor addr, like "--es.uri=http://%s", %s is target ip:port
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -params value
        slave exporter params list, may be used multiple times or null; except monitor-addr and listen-addr。
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging

```

The parameters you need to pay attention to are as follows:

```
  -bind-addr string
        master bind address for the metrics server (default ":8080")
  -exporter-bin-file string
        slave exporter bin file addr, like /bin/kafka_exporter
  -exporter-listen-addr string
        slave exporter listen addr, like "--web.listen-address=:%d", %d is listen port (default "--web.listen-address=:%d")
  -exporter-monitor-addr string
        slave exporter monitor addr, like "--es.uri=http://%s", %s is target ip:port
```

Monitoring example with kakfa, use [kafka_exporter](https://github.com/danielqsj/kafka_exporter):

Run single kafka exporter:

`/kafka_exporter --web.listen-address=:18081 --kafka.server=10.200.21.11:9092`

use common_exporter running in k8s:

`/common_exporter -logtostderr -v 10 -bind-addr :8080 -exporter-bin-file '/kafka_exporter' -exporter-monitor-addr '--kafka.server=%s'`

