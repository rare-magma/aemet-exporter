FROM docker.io/library/golang:1.22-alpine as builder
WORKDIR /app
ENV CGO_ENABLED=0
COPY main.go go.mod ./
RUN go build -ldflags "-s -w" -o aemet_exporter main.go

FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/aemet_exporter /app/aemet_exporter
ENTRYPOINT ["/app/aemet_exporter"]
