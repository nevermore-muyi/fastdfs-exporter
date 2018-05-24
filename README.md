## Build

You can get the repository yourself and build using `make`:

```
$ go get github.com/ruanchen/fastdfs-exporter
$ cd $GOPATH/src/github.com/ruanchen/fastdfs-exporter
$ make build
$ ./fastdfs-exporter
```

## Using Docker

```
docker run -d \
-e "APISERVER=$APISERVER" \
-e "FASTDFS_POD_NAME=$FASTDFS_POD_NAME"\
registry.cn-hangzhou.aliyuncs.com/nevermore/fastdfs-exporter:v0.1
```

## Configuration

fastdfs_exporter uses environment variables for configuration. Settings:

| Environment variable | default               | description                                 |
| -------------------- | --------------------- | ------------------------------------------- |
| APISERVER            | http://localhost:8080 | url of kubernetes apiserver for kubectl cli |
| FASTDFS_POD_NAME     | fastdfs               | the pod name of fastdfs                     |

## Metrics

All metrics (except golang/prometheus metrics) are prefixed with "fastdfs_".

| metric             | description                             |
| ------------------ | --------------------------------------- |
| group_count        | The expected number of group            |
| config_group_count | The actual number of group              |
| config_storage_num | The expected number of storage          |
| active_state       | Total number of active state storage    |
| wait_sync_state    | Total number of wait_sync state storage |