# Solace PubSub+ Event Broker Operator User Guide

This document provides detailed information for deploying the [Solace PubSub+ Software Event Broker](https://solace.com/products/event-broker/software/) on Kubernetes, using the Solace PubSub+ Event Broker Operator. A basic understanding of [Kubernetes concepts](https://kubernetes.io/docs/concepts/) is assumed.

The following additional set of documentation is also available:

* For a hands-on quick start, refer to the [Quick Start guide](/README.md).
* For the `PubSubPlusEventBroker` custom resource (deployment configuration) parameter options, refer to the [PubSub+ Event Broker Operator Parameters Reference]().
* For version-specific information, refer to the [Operator Release Notes]()

This guide is focused on deploying the event broker using the Operator, which is the preferred way to deploy. Note that a legacy way of [Helm-based deployment]() is also supported.

Contents:



## The Solace PubSub+ Software Event Broker

The [PubSub+ Software Event Broker](https://solace.com/products/event-broker/) of the [Solace PubSub+ Platform](https://solace.com/products/platform/) efficiently streams event-driven information between applications, IoT devices and user interfaces running in the cloud, on-premises, and hybrid environments using open APIs and protocols like AMQP, JMS, MQTT, REST and WebSocket. It can be installed into a variety of public and private clouds, PaaS, and on-premises environments, and brokers in multiple locations can be linked together in an [event mesh](https://solace.com/what-is-an-event-mesh/) to dynamically share events across the distributed enterprise.

## Overview

The PubSub+ Event Broker Operator supports:
- Installation of a PubSub+ Software Event Broker in non-HA or HA mode
- Adjusting the deployment to updated parameters
- Upgrade to a new broker version
- Repair the deployment
- Enable Prometheus monitoring
- Provide status of the deployment

Once the Operator has been installed, deployment of a broker is simply matter of creating a `PubSubPlusEventBroker` manifest that declares the broker properties, in Kubernetes. This is not different from creating any Kubernetes-native resource, for example a Pod.

Kubernetes will pass the manifest to the Operator and the Operator will supervise the deployment from beginning to completion. The Operator will also take corrective action or provide notification if the deployment deviates from the desired state.

## Supported Kubernetes Environments

The Operator supports Kubernetes version 1.23 or later and is generally expected to work in complying Kubernetes environments.

This includes OpenShift as there are provisions in the Operator to detect OpenShift environment and seamlessly adjust defaults. Details will be provided at the appropriate parameters.

##	Architecture

###	Operator

The PubSub+ Operator is following the [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). The diagram gives an overview of this mechanism:

![alt text](/docs/images/OperatorArchitecture.png "Operator overview")

* `PubSubPlusEventBroker` is registered with Kubernetes as a Custom Resource and it becomes a recognised Kubernetes object type.
* The `PubSub+ Event Broker Operator` packaged in a Pod within a Deployment must be in running state. It is configured with a set of Kubernetes namespaces to watch, which can be a list of specified ones or all.
* Creating a `PubSubPlusEventBroker` Custom Resource (CR) in a watched namespace triggers the creation of a new PubSub+ Event Broker deployment that meets the properties specified in the CR manifest.
* Deviation of the deployment from the desired state or a change in the CR spec also triggers the operator to reconcile, that is to adjust the deployment towards the desired state.
* The operator runs reconcile runs in loops, making one adjustment at a time, until the desired state has been reached.
* Note that RBAC settings are required to permit the operator create Kubernetes objects, especially in other namespaces. Refer to the [Security]() section for further details.

The activity of the Operator can be followed from its Pod logs as described in the troubleshooting section.

### Event Broker Deployment

The following diagram illustrates a [Highly Available (HA)](https://docs.solace.com/Features/HA-Redundancy/SW-Broker-Redundancy-and-Fault-Tolerance.htm) PubSub+ Event Broker deployment in Kubernetes. HA deployment requires three brokers in designated roles of Primary, Backup and Monitor in an HA group.

![alt text](/docs/images/BrokerDeployment.png "HA broker deployment")

* At the core, there are the Pods running the broker containers and the associated PVC storage elements, directly managed by dedicated StatefulSets.
* There are a set of shell scripts in the form of a ConfigMap mounted on each broker container. They take care of configuring the broker container at startup and conveying internal broker state to Kubernetes by reporting readiness and signalling which Pod is active and ready for service traffic.
* The listed Secrets are also mounted on the container feeding into the security configuration.
* Signalling Active broker state requires permissions to update pod labels so this needs to be configured through RBAC settings for the deployment.
* Discovery Service enables internal access between brokers and the Broker Service provides external Service Ports. The External service exposes the currently Active broker's services, which can be either the Primary or the Backup broker.

The Operator ensures that all above objects are in place with the exeception of the Pods managed by the StatefulSets as described. This enables that even if the Operator is temporarily out of service, the broker will stay functional and resilient (noting that introducing changes will not be possible during that time) because the Operator only controls the StatefulSets directly.

7	Broker Deployment Considerations
7.1	Topology
7.1.1	High Availability & Disaster recovery
7.1.2	Node Assignment
7.2	Container Images
7.2.1	Broker, exporter
7.3	Scaling
7.3.1	□ Dev, note that it takes precedence over all else!
7.4	Storage
7.5	Exposing broker services
7.6	TLS to access broker services
7.6.1	Enable or disable for an existing deployment is manual only
7.7	Security
7.7.1	Production recommendations
7.7.2	ref to Secrets
7.7.3	ref to Namespace
7.7.4 Ref to Signalling active state requires permissions to update pod labels so this needs to be configured through RBAC settings for the deployment

7.8	Exposing health and performance metrics
8	Broker Deployment Guide
Deployment pre-requisites 
8.1	Install Operator
8.1.1	kubectl
8.1.2	OLM
8.1.3	GitHub repo
8.2	Deploy Broker
8.2.1	Samples
8.3	Operate broker
8.3.1	Validating the deployment
8.3.2	How to connect, etc.
List of services
Expose services
8.4	Update / upgrade broker
8.4.1	Rolling vs. Manual update
8.4.2	Mechanics of picking up changes
8.4.3	AutoReconfiguration-enabled parameters
8.4.4	Maintenance mode
8.5	Undeploy Broker
8.6	Re-Install Broker
9	Troubleshooting

CRD in place
Check operator running
Check operator settings
- namespace
Check operator logs


9.1	Common K8s issues
9.2	Status, Logs, Events, Conditions
9.3	Broker stuck in bad state
9.4	Using of Metrics
10	Migration from Helm-based deployment
10.1	Possible IP address change
