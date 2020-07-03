FROM alpine:3.7
EXPOSE 8080
COPY app_exporter /app_exporter
COPY commom_exporter /commom_exporter
CMD ["/commom_exporter", "-v"]