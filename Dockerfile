FROM alpine:3.7
EXPOSE 8080
COPY kafka_exporter /kafka_exporter
COPY kafka_exporter_v2 /kafka_exporter_v2
CMD ["/kafka_exporter_v2"]