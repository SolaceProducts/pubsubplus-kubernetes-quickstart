# Build the manager binary
FROM golang:1.18 as builder

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
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

RUN microdnf install -y curl-minimal-7.76.1-26.el9_3.2 libcurl-minimal-7.76.1-26.el9_3.2

LABEL name="solace/pubsubplus-eventbroker-operator"
LABEL vendor="Solace Corporation"
LABEL version="1.0.2"
LABEL release="1.0.2"
LABEL summary="Solace PubSub+ Event Broker Kubernetes Operator"
LABEL description="The Solace PubSub+ Event Broker Kubernetes Operator deploys and manages the lifecycle of PubSub+ Event Brokers"

WORKDIR /
COPY THIRD-PARTY-LICENSES.md /licenses/THIRD-PARTY-LICENSES.md
COPY LICENSE /licenses/LICENSE
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
