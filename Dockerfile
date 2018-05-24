FROM        quay.io/prometheus/busybox:glibc
MAINTAINER  RuanChen@NeverMore

COPY fastdfs_exporter /bin/fastdfs_exporter
COPY kubectl /bin/kubectl

EXPOSE      10000
ENTRYPOINT  [ "/bin/fastdfs_exporter" ]