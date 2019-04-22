FROM quay.io/deis/lightweight-docker-go:v0.6.0
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/lovethedrake/prototype
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY vendor/ vendor/
RUN go build -o bin/brigade-worker ./cmd/brigade-worker

FROM scratch
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/github.com/lovethedrake/prototype/bin/ /brigade-worker/bin/
CMD ["/brigade-worker/bin/brigade-worker"]
