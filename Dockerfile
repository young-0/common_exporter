FROM alpine:3.7
EXPOSE 8080
COPY common_exporter /commom_exporter
CMD ["/common_exporter", "-v"]