# Solace PubSub+ Event Broker Operator Quick Start
[![Actions Status](https://github.com/SolaceDev/pubsubplus-kubernetes-operator/actions/workflows/build-test-main.yml/badge.svg?branch=v1.0.0)](https://github.com/SolaceDev/pubsubplus-kubernetes-operator/actions?query=workflow%3Abuild+branch%3Av1.0.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/solacedev/pubsubplus-kubernetes-operator)](https://goreportcard.com/report/github.com/solacedev/pubsubplus-kubernetes-operator)
![Coverage](https://img.shields.io/badge/Coverage-76.4%25-brightgreen)

The Solace PubSub+ Event Broker Operator (or simply the Operator) is a Kubernetes-native method to install and manage a Solace PubSub+ Software Event Broker on a Kubernetes cluster.

[PubSub+ Platform](https://solace.com/products/platform/) is a complete event streaming and management platform for the real-time enterprise. The [PubSub+ Software Event Broker](https://solace.com/products/event-broker/software/) efficiently streams event-driven information between applications, IoT devices, and user interfaces running in the cloud, on-premises, and in hybrid environments using open APIs and protocols like AMQP, JMS, MQTT, REST and WebSocket. It can be installed into a variety of public and private clouds, PaaS, and on-premises environments. Event brokers in multiple locations can be linked together in an [Event Mesh](https://solace.com/what-is-an-event-mesh/) to dynamically share events across the distributed enterprise.

__Contents:__
- [Solace PubSub+ Event Broker Operator Quick Start](#solace-pubsub-event-broker-operator-quick-start)
  - [Overview](#overview)
    - [Additional Documentation](#additional-documentation)
  - [How to deploy the PubSub+ Software Event Broker onto Kubernetes using the Operator](#how-to-deploy-the-pubsub-software-event-broker-onto-kubernetes-using-the-operator)
    - [1. Get a Kubernetes environment](#1-get-a-kubernetes-environment)
    - [2. Install the Operator](#2-install-the-operator)
      - [a) OperatorHub and OLM Option](#a-operatorhub-and-olm-option)
      - [b) Direct Option](#b-direct-option)
    - [3. PubSub+ Software Event Broker Deployment Examples](#3-pubsub-software-event-broker-deployment-examples)
      - [a) Example Minimum-footprint Deployment for Developers](#a-example-minimum-footprint-deployment-for-developers)
      - [b) Example non-HA Deployment](#b-example-non-ha-deployment)
      - [c) Example HA Deployment](#c-example-ha-deployment)
      - [d) Deployment with Prometheus Monitoring Enabled](#d-deployment-with-prometheus-monitoring-enabled)
    - [4. Test the deployment](#4-test-the-deployment)
    - [Additional information](#additional-information)
  - [Contributing](#contributing)
  - [Authors](#authors)
  - [License](#license)
  - [Resources](#resources)

## Overview

This project is a best practice template intended for development and demo purposes. The tested and recommended PubSub+ Software Event Broker version is 10.3.

This document provides a quick getting started guide to install a software event broker in various configurations onto a [Kubernetes](https://kubernetes.io/docs/home/) cluster using the PubSub+ Event Broker Operator. Note that a [Helm-based deployment](https://github.com/SolaceProducts/pubsubplus-kubernetes-helm-quickstart) of the broker is also supported but out of scope for this document.

These instructions apply to any platform supporting Kubernetes, and include specific hints for setting up a simple [MiniKube](https://kubernetes.io/docs/tasks/tools/#minikube) or [Kind](https://kubernetes.io/docs/tasks/tools/#kind) deployment on a Linux-based machine.

The following Kubernetes platforms have been tested:

- Google Kubernetes Engine (GKE)
- OpenShift 4 Platform on AWS


### Additional Documentation

Detailed documentation is provided in the [Solace PubSub+ Event Broker Operator User Guide](docs/EventBrokerOperatorUserGuide.md). In particular, consult the [Deployment Planning](docs/EventBrokerOperatorUserGuide.md#deployment-planning) section of the User Guide when planning your deployment.

## How to deploy the PubSub+ Software Event Broker onto Kubernetes using the Operator

The PubSub+ Software Event Broker can be deployed in either a three-node High-Availability (HA) group or as a single-node standalone deployment. For a simple test environment used only for validating application functionality, a standalone deployment is sufficient. Note that in production, or any environment where message loss cannot be tolerated, an HA deployment is required.

In this quick start we go through the steps to deploy a PubSub+ Software Event Broker using the PubSub+ Event Broker Operator.

### 1. Get a Kubernetes environment

Follow your Kubernetes provider's instructions ([additional options available here](https://kubernetes.io/docs/setup/)). Ensure you meet [minimum CPU, Memory, and Storage requirements](https://docs.solace.com/Software-Broker/System-Resource-Calculator.htm) for the targeted PubSub+ Software Event Broker configuration size. 

__Important__: The broker resource requirements refer to available resources on a [Kubernetes node](https://kubernetes.io/docs/concepts/scheduling-eviction/kube-scheduler/#kube-scheduler).

> Note: If you're using [MiniKube](https://kubernetes.io/docs/setup/learning-environment/minikube/), use `minikube start`, specifying the options `--memory` and `--cpus` to assign adequate resources to the MiniKube VM. The recommended memory is 1GB plus the minimum requirements of your event broker.

You must also have the `kubectl` tool [installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/) locally.

To verify that your Kubernetes environment is ready, run the following command:
```bash
# This command returns the list of worker nodes and their status
kubectl get nodes
```

### 2. Install the Operator

The Operator is available from the Registry for Kubernetes Operators, [OperatorHub.io](https://operatorhub.io/). With OperatorHub, the operator lifecycle, including installation and upgrades, is managed by the Operator Lifecycle Manager (OLM). Depending on your Kubernetes distribution, the OLM may already be pre-installed. If it is not, you must add it before you install the Operator.

Although the OLM is the recommended way to install the PubSub+ Event Broker Operator because of the lifecycle services it provides, a simpler, direct install method is also available that doesn't require OLM.

After you complete any of the following install options with the default settings, the Event Broker Operator will be [up and running, watching all namespaces for `PubSubPlusEventBroker` Custom Resources](docs/EventBrokerOperatorUserGuide.md#operator), and ready for the next steps. 

>Note: Ensure there is only one installation of the Operator at any time to avoid conflicts.

#### a) OperatorHub and OLM Option

Follow the steps from [OperatorHub](https://operatorhub.io/operator/pubsubplus-eventbroker-operator) to first setup OLM, then to install the PubSub+ Event Broker Operator. Click on the Install button to see the detailed instructions.

By default this method installs the Operator in the `operators` namespace.

#### b) Direct Option

The following commands directly install the Operator:

```bash
# Download manifest for possible edit
wget https://raw.githubusercontent.com/SolaceProducts/pubsubplus-kubernetes-operator/main/deploy/deploy.yaml
# Manifest creates a namespace and all K8s resources for the Operator deployment
kubectl apply -f deploy.yaml
# Wait for deployment to complete
kubectl get pods -n pubsubplus-operator-system --watch
```

By default this method installs the Operator in the `pubsubplus-operator-system` namespace.

### 3. PubSub+ Software Event Broker Deployment Examples

The section includes examples for the following deployment variants, with default small-scale configurations:
- [a) Example Minimum-footprint Deployment for Developers](#a-example-minimum-footprint-deployment-for-developers)—Recommended minimal standalone PubSub+ Software Event Broker for Developers. No guaranteed performance
- [b) Example non-HA Deployment](#b-example-non-ha-deployment)—Standalone, production-ready PubSub+ Software Event Broker with performance supporting up to 100 client connections
- [c) Example HA Deployment](#c-example-ha-deployment)—PubSub+ Software Event Broker with brokers in HA redundancy group, production-ready performance supporting up to 100 client connections
- [d) Deployment with Prometheus Monitoring Enabled](#d-deployment-with-prometheus-monitoring-enabled)—an example non-HA deployment with Prometheus monitoring enabled

By default the [latest publicly available Docker image](https://hub.docker.com/r/solace/solace-pubsub-standard/tags/) of the PubSub+ Software Event Broker Standard Edition is used.

For other PubSub+ Software Event Broker configurations, refer to the [PubSub+ Event Broker Operator Parameters Reference](/docs/EventBrokerOperatorParametersReference.md) and the [User Guide](/docs/EventBrokerOperatorUserGuide.md).

>Important: Although the non-HA and HA deployments have performance that is suitable for Production, we recommend that you consult the [Security Considerations](/docs/EventBrokerOperatorUserGuide.md#security-considerations) documentation for information about adequate security hardening in your environment.

#### a) Example Minimum-footprint Deployment for Developers

This minimal non-HA deployment requires 1 CPU and 4 GB of memory available to the event broker pod.
```bash
# Create deployment manifest
echo "
apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  name: dev-example
spec:
  developer: true" > developer.yaml
# Then apply it
kubectl apply -f developer.yaml
# Wait for broker deployment pods to be ready
kubectl get pods --show-labels --watch
# Check service-ready
kubectl wait --for=condition=ServiceReady eventbroker dev-example
```

#### b) Example non-HA Deployment

A minimum of 2 CPUs and 4 GB of memory must be available to the event broker pod.
```bash
# Create deployment manifest
echo "
apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  name: non-ha-example
spec:
  redundancy: false  # Default, not strictly required
" > nonha.yaml
# Then apply it
kubectl apply -f nonha.yaml
# Wait for broker deployment pods to be ready
kubectl get pods --show-labels --watch
# Check service-ready
kubectl wait --for=condition=ServiceReady eventbroker non-ha-example
```

#### c) Example HA Deployment

The minimum resource requirements are 2 CPUs and 4 GB of memory available to each of the three event broker pods.
```bash
# Create deployment manifest
echo "
apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  name: ha-example
spec:
  redundancy: true
" > ha.yaml
# Then apply it
kubectl apply -f ha.yaml
# Wait for broker deployment pods to be ready
kubectl get pods --show-labels --watch
# Check service-ready and then HA-ready
kubectl wait --for=condition=ServiceReady eventbroker ha-example
kubectl wait --for=condition=HAReady eventbroker ha-example
```

#### d) Deployment with Prometheus Monitoring Enabled

This is the same as the non-HA deployment example with Prometheus monitoring enabled.
```bash
# Create deployment manifest
echo "
apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  name: non-ha-monitoring-enabled-example
spec:
  monitoring:
    enabled: true
" > nonha-with-monitoring.yaml
# Then apply it
kubectl apply -f nonha-with-monitoring.yaml
# Wait for broker deployment pods to be ready plus monitoring pod running
kubectl get pods --show-labels --watch
# Check service-ready and then HA-ready
kubectl wait --for=condition=ServiceReady eventbroker non-ha-monitoring-enabled-example
kubectl wait --for=condition=MonitoringReady eventbroker non-ha-monitoring-enabled-example
```
For more information about Prometheus monitoring, see [Exposing Metrics to Prometheus](/docs/EventBrokerOperatorUserGuide.md#exposing-metrics-to-prometheus) in the detailed PubSub+ Operator documentation. 

### 4. Test the deployment

The examples in the preceding section create a deployment. You can use the commands that follow to check the event broker deployment status, get information about the service name and type to access the broker services, and the obtain the secret that contains the credentials to be used for admin access.

The following examples use `dev-example` as the deployment name. When you run these commands, use your deployment's name instead.

```
kubectl describe eventbroker dev-example
```

* Obtain the management admin password:
```
ADMIN_SECRET_NAME=$(kubectl get eventbroker dev-example -o jsonpath='{.status.broker.adminCredentialsSecret}')
# This command returns the management "admin" user's password
kubectl get secret $ADMIN_SECRET_NAME -o jsonpath='{.data.username_admin_password}' | base64 -d
```

* Obtain the IP address to access the broker services:
```
BROKER_SERVICE_NAME=$(kubectl get eventbroker dev-example -o jsonpath='{.status.broker.serviceName}')
# This command returns the broker service's external IP address
kubectl get svc $BROKER_SERVICE_NAME -o jsonpath='{.status.loadBalancer.ingress}'
```

> Note: When using MiniKube or other minimal Kubernetes provider, there may be no integrated Load Balancer available, which is the default service type. For a workaround, either refer to the [MiniKube documentation for LoadBalancer access](https://minikube.sigs.k8s.io/docs/handbook/accessing/#loadbalancer-access) or use [local port forwarding to the service port](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/#forward-a-local-port-to-a-port-on-the-pod): `kubectl port-forward service/$BROKER_SERVICE_NAME <target-port-on-localhost>:<service-port-on-load-balancer> &`. Then access the service at `localhost:<target-port-on-localhost>`

* Access the PubSub+ Broker Manager

In your browser, navigate to the IP address you obtained, using port 8080: 
```
http://<ip-address>:8080
```
 Login as user `admin` with the management admin password you obtained.

> If required use above Load Balancer access workaround for service port `8080`.

* Use the Broker Manager [built-in Try-Me](https://docs.solace.com/Admin/Broker-Manager/PubSub-Manager-Overview.htm?Highlight=manager#Test-Messages) tool to test messaging.

> If required use above Load Balancer access workaround for service port `8008`.


### Additional information

Refer to the detailed PubSub+ Event Broker Operator documentation for:
* [Validating the deployment](docs/EventBrokerOperatorUserGuide.md#validating-the-deployment)
* [Troubleshooting](docs/EventBrokerOperatorUserGuide.md#troubleshooting)
* [Modifying or Upgrading](docs/EventBrokerOperatorUserGuide.md#modifying-a-broker-deployment-including-broker-upgrade)
* [Deleting the deployment](docs/EventBrokerOperatorUserGuide.md#deleting-a-deployment)

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Authors

See the list of [contributors](https://github.com/SolaceProducts/pubsubplus-kubernetes-operator/graphs/contributors ) who participated in this project.

## License

This project is licensed under the Apache License, Version 2.0. - See the [LICENSE](LICENSE) file for details.

## Resources

For more information about Solace technology in general please visit these resources:

- The Solace Developer Portal website at: [solace.dev](https://solace.dev/)
- Understanding [Solace technology](https://solace.com/products/platform/)
- Ask the [Solace community](https://solace.community/).