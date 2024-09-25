# Build the manager binary
FROM golang:1.22.7 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi9-minimal:9.4-1227.1725849298

LABEL name="solace/pubsubplus-eventbroker-operator"
LABEL vendor="Solace Corporation"
LABEL version="1.3.0"
LABEL release="1.3.0"
LABEL summary="Solace PubSub+ Event Broker Kubernetes Operator"
LABEL description="The Solace PubSub+ Event Broker Kubernetes Operator deploys and manages the lifecycle of PubSub+ Event Brokers"

WORKDIR /
COPY THIRD-PARTY-LICENSES.md /licenses/THIRD-PARTY-LICENSES.md
COPY LICENSE /licenses/LICENSE
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
