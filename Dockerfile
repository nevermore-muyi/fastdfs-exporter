FROM        quay.io/prometheus/busybox:glibc
MAINTAINER  RuanChen@NeverMore

COPY fastdfs_exporter kubectl active.sh wait.sh groupcount.sh /bin/
RUN chmod 755 /bin/kubectl
WORKDIR /bin

EXPOSE      10000
ENTRYPOINT  [ "/bin/fastdfs_exporter" ]