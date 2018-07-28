FROM golang:1.10 AS builder
RUN go get -u golang.org/x/vgo
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 vgo build \
#    -ldflags "-X github.wdf.sap.corp/scp-blockchain/aws-monitor/pkg/version.version=0.3.0 -X github.wdf.sap.corp/scp-blockchain/aws-monitor/pkg/version.gitCommit=$(git rev-parse HEAD)" \
    -o github-app

FROM scratch
WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/github-app /app/github-app
ENTRYPOINT ["/app/github-app"]
