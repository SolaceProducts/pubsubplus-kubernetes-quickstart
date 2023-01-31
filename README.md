# Solace PubSub+ Event Broker Operator Quick Start Guide

Using Solace PubSub+ Event Broker Operator (Operator) is the Kubernetes-native method to install a Solace PubSub+ Software Event Broker onto a Kubernetes cluster.

The [Solace PubSub+ Platform](https://solace.com/products/platform/)'s [software event broker](https://solace.com/products/event-broker/software/) efficiently streams event-driven information between applications, IoT devices and user interfaces running in the cloud, on-premises, and hybrid environments using open APIs and protocols like AMQP, JMS, MQTT, REST and WebSocket. It can be installed into a variety of public and private clouds, PaaS, and on-premises environments, and brokers in multiple locations can be linked together in an [event mesh](https://solace.com/what-is-an-event-mesh/) to dynamically share events across the distributed enterprise.

Contents:
- [Solace PubSub+ Event Broker Operator Quick Start Guide](#solace-pubsub-event-broker-operator-quick-start-guide)
  - [Overview](#overview)
  - [How to deploy the Solace PubSub+ Software Event Broker onto Kubernetes using the Operator](#how-to-deploy-the-solace-pubsub-software-event-broker-onto-kubernetes-using-the-operator)
    - [1. Get a Kubernetes environment](#1-get-a-kubernetes-environment)
    - [2. Install the Operator](#2-install-the-operator)
      - [a) OperatorHub and OLM Option](#a-operatorhub-and-olm-option)
      - [b) Direct Option](#b-direct-option)
    - [3. Install the Solace PubSub+ Software Event Broker with default configuration](#3-install-the-solace-pubsub-software-event-broker-with-default-configuration)
      - [a) Deployment for Developers](#a-deployment-for-developers)
      - [b) non-HA Deployment](#b-non-ha-deployment)
      - [c) HA Deployment](#c-ha-deployment)
  - [Contributing](#contributing)
  - [Authors](#authors)
  - [License](#license)
  - [Resources](#resources)

## Overview

This project is a best practice template intended for development and demo purposes. The tested and recommended Solace PubSub+ Software Event Broker version is 10.1.

This document provides a quick getting started guide to install a software event broker in various configurations onto a [Kubernetes](https://kubernetes.io/docs/home/) cluster using the PubSub+ Event Broker Operator.

Detailed documentation is provided in the [Solace PubSub+ Event Broker Operator User Guide](PubSubPlusEventBrokerOperator.md). Consult the [Deployment Considerations](PubSubPlusEventBrokerOperator.md#deployment-planning) section of the User Guide when planning your deployment.

This document is applicable to any platform supporting Kubernetes, with specific hints on how to set up a simple [MiniKube or Kind]() deployment on a Linux-based machine. To view examples of other Kubernetes platforms see:

- Deploying a Solace PubSub+ Software Event Broker HA Group onto Amazon EKS (Amazon Elastic Container Service for Kubernetes): follow the [AWS documentation](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html ) to set up EKS then this guide to deploy.
- Deploying a Solace PubSub+ Software Event Broker HA Group onto Azure Kubernetes Service (AKS): follow the [Azure documentation](https://docs.microsoft.com/en-us/azure/aks/ ) to deploy an AKS cluster then this guide to deploy.
- [Deploying a Solace PubSub+ Software Event Broker HA group onto a Google Kubernetes Engine](https://github.com/SolaceProducts/solace-gke-quickstart )
- [Deploying a Solace PubSub+ Software Event Broker HA Group onto an OpenShift 4 platform](https://github.com/SolaceProducts/solace-openshift-quickstart )
- [Install a Solace PubSub+ Software Event Broker onto a Tanzu Kubernetes Cluster](https://github.com/SolaceProducts/solace-pks )

## How to deploy the Solace PubSub+ Software Event Broker onto Kubernetes using the Operator

Solace PubSub+ Software Event Broker can be deployed in either a three-node High-Availability (HA) group or as a single-node standalone deployment. For simple test environments that need only to validate application functionality, a single instance will suffice. Note that in production, or any environment where message loss cannot be tolerated, an HA deployment is required.

In this quick start we go through the steps to deploy a PubSub+ Software Event Broker using the Solace PubSub+ Event Broker Operator.

### 1. Get a Kubernetes environment

Follow your Kubernetes provider's instructions ([additional options available here](https://kubernetes.io/docs/setup/)). Ensure you meet [minimum CPU, Memory and Storage requirements]() for the targeted PubSub+ Software Event Broker configuration size. Important: the broker resource requirements refer to available resources on a [Kubernetes node](https://kubernetes.io/docs/concepts/scheduling-eviction/kube-scheduler/#kube-scheduler).
> Note: If using [MiniKube](https://kubernetes.io/docs/setup/learning-environment/minikube/), use `minikube start` with specifying the options `--memory` and `--cpus` to assign adequate resources to the MiniKube VM. The recommended memory is 1GB plus the minimum requirements of your event broker.

Also have the `kubectl` tool [installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/) locally.

Check to ensure your Kubernetes environment is ready:
```bash
# This shall return worker nodes listed and ready
kubectl get nodes
```

### 2. Install the Operator

The Operator is available from the Registry for Kubernetes Operators, [OperatorHub.io](https://operatorhub.io/). When using OperatorHub, operator lifecycle including installation and upgrades is managed by the Operator Lifecycle Manager (OLM), which needs to be added first or may already be pre-installed on your Kubernetes distribution.

While OLM is the recommended way to install the PubSub+ Event Broker Operator because of the lifecycle-services, a Direct install method is also available that doesn't require OLM.

By completing any of the install options with default settings the Event Broker Operator shall be up and running watching all namespaces, ready for the next step.

>Note: ensure there is only one installation of the Operator at any time to avoid conflicts.

#### a) OperatorHub and OLM Option

Follow the steps from [OperatorHub](https://operatorhub.io/operator/pubsubplus-eventbroker-operator) to setup OLM then to install the PubSub+ Event Broker Operator. Click on the Install button to see the detailed instructions.

#### b) Direct Option

Apply this manifest to setup the Operator in the `pubsubplus-operator-system` namespace:

```bash
# For internal use only, DELETE when publishing
# Pre-requisite: Docker login into the private registry that hosts the Operator image
kubectl create ns pubsubplus-operator-system --save-config
kubectl create secret generic regcred \
  --from-file=.dockerconfigjson=${HOME}/.docker/config.json \
  --type=kubernetes.io/dockerconfigjson \
  -n pubsubplus-operator-system
# Manifest creates a namespace and all K8s resources for the Operator deployment
kubectl apply -f https://raw.githubusercontent.com/SolaceDev/pubsubplus-kubernetes-operator/v1alpha1/deploy/deploy.yaml
# Wait for deployment to complete
kubectl rollout status deployment pubsubplus-eventbroker-operator \
  -n pubsubplus-operator-system
```

### 3. Install the Solace PubSub+ Software Event Broker with default configuration

Three deployment variants will be presented with default small-scale configurations:
*	*Developer*: recommended PubSub+ Software Event Broker for Developers (standalone) - no guaranteed performance
* *Non-HA*: PubSub+ Software Event Broker Standalone, supporting 100 client connections
* *HA*: PubSub+ Software Event Broker with brokers in HA redundancy group, supporting 100 client connections

By default the publicly available [latest Docker image of PubSub+ Software Event Broker Standard Edition](https://hub.Docker.com/r/solace/solace-pubsub-standard/tags/) will be used.

For other PubSub+ Software Event Broker configurations or sizes, refer to the [PubSub+ Event Broker Operator Parameters Reference]() and the [User Guide](PubSubPlusEventBrokerOperator.md).

>Important: While the non-HA and HA deployments will be ready for Production performance, consult the [Security Considerations]() documentation for adequate security hardening in your environment.

#### a) Deployment for Developers

This deployment requires a minimum of 1 CPU and 4 GB of memory available to the event broker pod.
```bash
# Deploy PubSub+ Software Event Broker Standard edition for developers
helm install my-release solacecharts/pubsubplus-dev
```

#### b) non-HA Deployment

A minimum of 2 CPUs and 4 GB of memory must be available to the event broker pod.
```bash
# Deploy PubSub+ Software Event Broker Standard edition, standalone
helm install my-release solacecharts/pubsubplus
```

#### c) HA Deployment

The minimum resource requirements are 2 CPU and 4 GB of memory available to each of the three event broker pods.
```bash
# Deploy PubSub+ Software Event Broker Standard edition, HA
helm install my-release solacecharts/pubsubplus-ha
```

The above options will start the deployment and write related information and notes to the screen.

> Note: When using MiniKube, there is no integrated Load Balancer, which is the default service type. For a workaround, execute `minikube service my-release-pubsubplus-ha` to expose the services. Services will be accessible directly using the NodePort instead of direct Port access, for which the mapping can be obtained from `kubectl describe service my-release-pubsubplus-ha`.

Wait for the deployment to complete following the information printed on the console.

Refer to the detailed PubSub+ Kubernetes documentation for:
* [Validating the deployment](); or
* [Troubleshooting]()
* [Modifying or Upgrading]()
* [Deleting the deployment]()

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Authors

See the list of [contributors](https://github.com/SolaceProducts/pubsubplus-kubernetes-quickstart/graphs/contributors) who participated in this project.

## License

This project is licensed under the Apache License, Version 2.0. - See the [LICENSE](LICENSE) file for details.

## Resources

For more information about Solace technology in general please visit these resources:

- The Solace Developer Portal website at: [solace.dev](https://solace.dev/)
- Understanding [Solace technology](https://solace.com/products/platform/)
- Ask the [Solace community](https://dev.solace.com/community/).