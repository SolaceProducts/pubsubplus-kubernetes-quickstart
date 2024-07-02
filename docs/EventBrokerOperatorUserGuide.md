# Solace PubSub+ Event Broker Operator User Guide

This document provides detailed information for deploying the [Solace PubSub+ Software Event Broker](https://solace.com/products/event-broker/software/) on Kubernetes, using the Solace PubSub+ Event Broker Operator. A basic understanding of [Kubernetes concepts](https://kubernetes.io/docs/concepts/) is assumed.

The following additional set of documentation is also available:

* For a hands-on quick start, refer to the [Quick Start guide](/README.md).
* For the `PubSubPlusEventBroker` custom resource (deployment configuration, or "broker spec") parameter options, refer to the [PubSub+ Event Broker Operator Parameters Reference](/docs/EventBrokerOperatorParametersReference.md).
* For version-specific information, refer to the [Operator Release Notes](https://github.com/SolaceProducts/pubsubplus-kubernetes-quickstart/releases)

This guide is focused on deploying the event broker using the Operator, which is the preferred way to deploy. Note that a [Helm-based deployment](https://github.com/SolaceProducts/pubsubplus-kubernetes-helm-quickstart) is also supported but out of scope for this document.

__Contents:__

- [Solace PubSub+ Event Broker Operator User Guide](#solace-pubsub-event-broker-operator-user-guide)
  - [The Solace PubSub+ Software Event Broker](#the-solace-pubsub-software-event-broker)
  - [Overview](#overview)
  - [Supported Kubernetes Environments](#supported-kubernetes-environments)
  - [Deployment Architecture](#deployment-architecture)
    - [Operator](#operator)
    - [Event Broker Deployment](#event-broker-deployment)
    - [Prometheus Monitoring Support](#prometheus-monitoring-support)
  - [Deployment Planning](#deployment-planning)
    - [Deployment Topology](#deployment-topology)
      - [High Availability](#high-availability)
      - [Node Assignment](#node-assignment)
      - [Enabling Pod Disruption Budget](#enabling-pod-disruption-budget)
    - [Container Images](#container-images)
      - [Using a public registry](#using-a-public-registry)
      - [Using a private registry](#using-a-private-registry)
      - [Pulling images from a private registry](#pulling-images-from-a-private-registry)
    - [Broker Scaling](#broker-scaling)
      - [Vertical Scaling](#vertical-scaling)
        - [Minimum footprint deployment for Developers](#minimum-footprint-deployment-for-developers)
        - [Forward Compatibility for Vertical Scaling](#forward-compatibility-for-vertical-scaling)
    - [Storage](#storage)
      - [Dynamically allocated storage from a Storage Class](#dynamically-allocated-storage-from-a-storage-class)
        - [Using an existing Storage Class](#using-an-existing-storage-class)
        - [Creating a new Storage Class](#creating-a-new-storage-class)
      - [Assigning existing PVC (Persistent Volume Claim)](#assigning-existing-pvc-persistent-volume-claim)
      - [Storage solutions and providers](#storage-solutions-and-providers)
    - [Accessing Broker Services](#accessing-broker-services)
      - [Serving Pod Selection](#serving-pod-selection)
      - [Using a Service Type](#using-a-service-type)
      - [Configuring TLS for broker services](#configuring-tls-for-broker-services)
        - [Setting up TLS](#setting-up-tls)
        - [Rotating the TLS certificate](#rotating-the-tls-certificate)
      - [Using Ingress](#using-ingress)
        - [Configuration examples](#configuration-examples)
        - [HTTP, no TLS](#http-no-tls)
        - [HTTPS with TLS terminate at ingress](#https-with-tls-terminate-at-ingress)
        - [HTTPS with TLS re-encrypt at ingress](#https-with-tls-re-encrypt-at-ingress)
        - [General TCP over TLS with passthrough to broker](#general-tcp-over-tls-with-passthrough-to-broker)
    - [Broker Pod additional properties](#broker-pod-additional-properties)
    - [Security Considerations](#security-considerations)
      - [Operator controlled namespaces](#operator-controlled-namespaces)
      - [Operator RBAC](#operator-rbac)
      - [Broker deployment RBAC](#broker-deployment-rbac)
      - [Operator image from private registry](#operator-image-from-private-registry)
      - [Admin and Monitor Users and Passwords](#admin-and-monitor-users-and-passwords)
      - [Secrets](#secrets)
      - [Broker Security Context](#broker-security-context)
      - [Using Network Policies](#using-network-policies)
  - [Exposing Metrics to Prometheus](#exposing-metrics-to-prometheus)
    - [Enabling and configuring the Broker Metrics Endpoint](#enabling-and-configuring-the-broker-metrics-endpoint)
    - [Available Broker Metrics](#available-broker-metrics)
    - [Connecting with Prometheus](#connecting-with-prometheus)
      - [Reference Prometheus Stack Deployment](#reference-prometheus-stack-deployment)
      - [Creating a ServiceMonitor object](#creating-a-servicemonitor-object)
    - [Grafana Visualization of Broker Metrics](#grafana-visualization-of-broker-metrics)
  - [Broker Deployment Guide](#broker-deployment-guide)
    - [Quick Start](#quick-start)
    - [Validating the deployment](#validating-the-deployment)
    - [Gaining admin access to the event broker](#gaining-admin-access-to-the-event-broker)
      - [Admin Credentials](#admin-credentials)
      - [Management access port](#management-access-port)
      - [Broker CLI access via the load balancer](#broker-cli-access-via-the-load-balancer)
      - [CLI access to individual event brokers](#cli-access-to-individual-event-brokers)
      - [SSH access to individual event brokers](#ssh-access-to-individual-event-brokers)
      - [Testing data access to the event broker](#testing-data-access-to-the-event-broker)
    - [Troubleshooting](#troubleshooting)
      - [General Kubernetes troubleshooting hints](#general-kubernetes-troubleshooting-hints)
      - [Checking the reason for failed resources](#checking-the-reason-for-failed-resources)
      - [Viewing logs](#viewing-logs)
      - [Updating log levels](#updating-log-levels)
      - [Viewing events](#viewing-events)
      - [Pods issues](#pods-issues)
        - [Pods stuck in not enough resources](#pods-stuck-in-not-enough-resources)
        - [Pods stuck in no storage](#pods-stuck-in-no-storage)
        - [Pods stuck in CrashLoopBackoff, Failed, or Not Ready](#pods-stuck-in-crashloopbackoff-failed-or-not-ready)
        - [No Pods listed](#no-pods-listed)
      - [Security constraints](#security-constraints)
    - [Maintenance mode](#maintenance-mode)
    - [Modifying a Broker Deployment including Broker Upgrade](#modifying-a-broker-deployment-including-broker-upgrade)
    - [Rolling vs. Manual Update](#rolling-vs-manual-update)
    - [Update Limitations](#update-limitations)
    - [Deleting a Deployment](#deleting-a-deployment)
    - [Re-Install Broker](#re-install-broker)
  - [Operator Deployment Guide](#operator-deployment-guide)
    - [Install Operator](#install-operator)
      - [From Operator Lifecycle Manager](#from-operator-lifecycle-manager)
      - [From Command Line](#from-command-line)
    - [Validating the Operator deployment](#validating-the-operator-deployment)
    - [Troubleshooting the Operator deployment](#troubleshooting-the-operator-deployment)
    - [Upgrade the Operator](#upgrade-the-operator)
        - [Upgrading the Operator only](#upgrading-the-operator-only)
      - [Upgrade CRD and Operator](#upgrade-crd-and-operator)
  - [Migration from Helm-based deployments](#migration-from-helm-based-deployments)
    - [Migration process](#migration-process)


## The Solace PubSub+ Software Event Broker

[PubSub+ Platform](https://solace.com/products/platform/) is a complete event streaming and management platform for the real-time enterprise. The [PubSub+ Software Event Broker](https://solace.com/products/event-broker/software/) efficiently streams event-driven information between applications, IoT devices, and user interfaces running in the cloud, on-premises, and in hybrid environments using open APIs and protocols like AMQP, JMS, MQTT, REST and WebSocket. It can be installed into a variety of public and private clouds, PaaS, and on-premises environments. Event brokers in multiple locations can be linked together in an [Event Mesh](https://solace.com/what-is-an-event-mesh/) to dynamically share events across the distributed enterprise.

## Overview

The PubSub+ Event Broker Operator supports:
- Installing a PubSub+ Software Event Broker in non-HA or HA mode.
- Adjusting the deployment to updated parameters (with limitations).
- Upgrading to a new broker version.
- Repairing the deployment.
- Enabling Prometheus monitoring.
- Providing status of the deployment.

After you have installed the Operator, you can deploy an event broker by simply creating a `PubSubPlusEventBroker` manifest that declares the broker properties in Kubernetes. This is no different from creating any Kubernetes-native resource, for example a Pod.

Kubernetes passes the manifest to the Operator and the Operator supervises the deployment from beginning to completion. The Operator also takes corrective action or provides notification if the deployment deviates from the desired state.

## Supported Kubernetes Environments

The Operator supports Kubernetes version 1.23 or later and is generally expected to work in complying Kubernetes environments.

This includes OpenShift because there are provisions in the Operator to detect OpenShift environment and seamlessly adjust defaults. Details are provided with the appropriate parameters.

##	Deployment Architecture

###	Operator

The PubSub+ Operator is following the [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). The diagram gives an overview of this mechanism:

![alt text](/docs/images/OperatorArchitecture.png "Operator overview")

* `PubSubPlusEventBroker` is registered with Kubernetes as a Custom Resource and it becomes a recognised Kubernetes object type.
* The `PubSub+ Event Broker Operator` packaged in a Pod within a Deployment must be in running state. It is configured with a set of Kubernetes namespaces to watch, which can be a list of specified ones or all.
* Creating a `PubSubPlusEventBroker` Custom Resource (CR) in a watched namespace triggers the creation of a new PubSub+ Event Broker deployment that meets the properties specified in the CR manifest (also referred to as "broker spec").
* Deviation of the deployment from the desired state or a change in the CR spec also triggers the operator to reconcile, that is to adjust the deployment towards the desired state.
* The operator runs reconcile in loops, making one adjustment at a time, until the desired state has been reached.
* Note that RBAC settings are required to permit the operator create Kubernetes objects, especially in other namespaces. Refer to the [Security](#broker-deployment-rbac) section for further details.

The activity of the Operator can be followed from its Pod logs as described in the [troubleshooting](#troubleshooting) section.

### Event Broker Deployment

The diagram illustrates a [Highly Available (HA)](https://docs.solace.com/Features/HA-Redundancy/SW-Broker-Redundancy-and-Fault-Tolerance.htm) PubSub+ Event Broker deployment in Kubernetes. HA deployment requires three brokers in designated roles of Primary, Backup and Monitor in an HA group.

![alt text](/docs/images/BrokerDeployment.png "HA broker deployment")

* At the core, there are the Pods running the broker containers and the associated Persistent Volume Claim (PVC) storage elements, directly managed by dedicated StatefulSets.
* Secrets are mounted on the containers feeding into the security configuration.
* There are also a set of shell scripts in a ConfigMap mounted on each broker container. They take care of configuring the broker at startup and conveying internal broker state to Kubernetes by reporting readiness and signalling which Pod is active and ready for service traffic. Active status is signalled by setting an `active=true` Pod label.
* A Service exposes the active broker Pod's services at service ports to clients.
* An additional Discovery Service enables internal access between brokers.
* Signaling active broker state requires permissions for a Pod to update its own label so this needs to be configured using RBAC settings for the deployment.

The Operator ensures that all above objects are in place, with the exception of the Pods and storage managed by the StatefulSets. This ensures that even if the Operator is temporarily out of service, the broker stays functional and resilient (noting that introducing changes are note possible during that time) because the StatefulSets control the Pods directly.

A non-HA deployment differs from HA in that: (1) there is only one StatefulSet managing one Pod that hosts the single broker; (2) there is no Discovery Service for internal communication; and (3) there is no pre-shared AuthenticationKey to secure internal communication.

Note: Each event broker deployment conforms to the guidelines for naming objects in Kubernetes. For example you can not have multiple event brokers with the same name in the same namespace, port names must be at least one character and no more than 15 characters long. For more info on other guidelines : [https://kubernetes.io/docs/concepts/overview/working-with-objects/names/](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/)    

### Prometheus Monitoring Support

Support can be enabled for exposing broker metrics to [Prometheus Monitoring](https://prometheus.io/docs/introduction/overview/). Prometheus requires an exporter running that pulls requested metrics from the monitored application - the broker in this case.

![alt text](/docs/images/MonitoringDeployment.png "Monitoring deployment")

* When monitoring is enabled, the Operator adds an Exporter Pod to the broker deployment. The Exporter Pod acts as a bridge between Prometheus and the broker deployment to deliver metrics.
* On one side, the Exporter Pod obtains metrics from the broker using SEMP requests. To access the broker, it uses the username and password from the MonitoringCredentials secret, and uses TLS access to the broker if Broker TLS has been configured.
* The metrics are exposed to Prometheus through the Prometheus Metrics Service via the metrics port. The Metrics port is accessible using TLS if Metrics TLS has been enabled.
* As Kubernetes recommended practice, it is assumed that the Prometheus stack has been deployed using the [Prometheus operator](https://github.com/prometheus-operator/prometheus-operator#overview) in a dedicated Prometheus Monitoring namespace. In this setup, a `ServiceMonitor` custom resource, placed in the Event Broker namespace, defines how Prometheus can access the broker metrics—which service to select and which endpoint to use.
* Prometheus comes installed with strict security by default. Its ClusterRole RBAC settings must be edited to enable watching ServiceMonitor in the Event Broker namespace.

## Deployment Planning

This section describes options that should be considered when planning a PubSub+ Event Broker deployment, especially for Production. 

### Deployment Topology

####	High Availability

The Operator supports deploying a single non-HA broker and also HA deployment for fault tolerance. This can be enabled by setting `spec.redundancy` to `true` in the broker deployment manifest.

#### Node Assignment

No single point of failure is important for HA deployments. Kubernetes by default tries to spread broker pods of an HA redundancy group across Availability Zones. For more deterministic deployments, specific control is enabled using the `spec.nodeAssignment` section of the broker spec for the Primary, Backup and Monitor brokers where Kubernetes standard [Affinity and NodeSelector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) definitions can be provided.

#### Enabling Pod Disruption Budget

In an HA deployment with Primary, Backup, and Monitor nodes, a minimum of two nodes must be available to reach quorum. Specifying a [Pod Disruption Budget](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) is recommended to limit situations where quorum might be lost.

This can be enabled setting the `spec.podDisruptionBudgetForHA` parameter to `true`. This creates a PodDisruptionBudget resource adapted to the broker deployment's needs, that is the number of minimum available pods set to two. Note that the parameter is ignored for a non-HA deployment.

### Container Images

There are two containers used in the deployment with following specifications:

|   | Event Broker  |  Prometheus Exporter |
|---|---|---|
| Image Repository  | `spec.image.repository`  |  `spec.monitoring.image.repository` |
| Image Tag  | `spec.image.tag`  |  `spec.monitoring.image.tag` |
| Pull Policy  | `spec.image.pullPolicy`  | `spec.monitoring.image.pullPolicy` |
| Pull Secrets  | `spec.image.pullSecrets`  | `spec.monitoring.image.pullSecrets` |

The Image Repository and Tag parameters combined specify the image to be used for the deployment. They can either point to an image in a public or a private container registry. Pull Policy and Pull Secrets can be specified the [standard Kubernetes way](https://kubernetes.io/docs/concepts/containers/images/).

Example:
```yaml
  image:
    repository: solace/solace-pubsub-standard
    tag: latest
    pullPolicy: IfNotPresent
    pullSecrets:
      - pull-secret
```

#### Using a public registry

For the broker image, default values are `solace/solace-pubsub-standard/` and `latest`, which is the free PubSub+ Software Event Broker Standard Edition from the [public Solace Docker Hub repo](https://hub.docker.com/r/solace/solace-pubsub-standard/). It is generally recommended to set the image tag to a specific build for traceability purposes.

Similarly, the default exporter image values are `solace/solace-pubsub-prometheus-exporter` and `latest`.

#### Using a private registry

Follow the general steps below to load an image into a private container registry (e.g.: GCR, ECR or Harbor). For specifics, consult the documentation of the registry you are using.

* Prerequisite: local installation of [Podman](https://podman.io/) or [Docker](https://www.docker.com/get-started/)
* Log into the private registry:
```sh
podman login <private-registry> ...
```
* First, load the broker image to the local registry:
>**Important** There are broker container image variants for `Amd64`and `Arm64` architectures. Ensure to use the correct image.
```sh
# Options a) or b) depending on your image source:
## Option a): If you have a local tar.gz image file
podman load -i <solace-pubsub-XYZ>.tar.gz
## Option b): You can use the public Solace container image, such as from Docker Hub
podman pull solace/solace-pubsub-standard:latest # or specific <TagName>
#
# Verify the image has been loaded and note the associated "IMAGE ID"
podman images
```
* Tag the image with a name specific to the private registry and tag:
```sh
podman tag <image-id> <private-registry>/<path>/<image-name>:<tag>
```
* Push the image to the private registry
```sh
podman push <private-registry>/<path>/<image-name>:<tag>
```
Note that additional steps might be required if using signed images.

#### Pulling images from a private registry

An ImagePullSecret might be required if pulling images from a private registry, e.g.: Harbor. 

Here is an example of creating an ImagePullSecret. Refer to your registry's documentation for the specific details of use.

```sh
kubectl create secret docker-registry <pull-secret-name> --dockerserver=<private-registry-server> \
  --docker-username=<registry-user-name> --docker-password=<registry-user-password> \
  --docker-email=<registry-user-email>
```

Then add `<pull-secret-name>` to the list under the `image.pullSecrets` parameter.

### Broker Scaling

The PubSub+ Event Mesh can be scaled vertically and horizontally.

You can horizontally scale your mesh by [connecting multiple broker deployments](https://docs.solace.com/Features/DMR/DMR-Overview.htm#Horizontal_Scaling). This is out of scope for this document.

#### Vertical Scaling

For vertical scaling, you set the maximum capacity of a given broker deployment using [system scaling parameters](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm).

The following scaling parameters can be specified:
* [Maximum Number of Client Connections](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm#max-client-connections), in `spec.systemScaling.maxConnections` parameter
* [Maximum Number of Queue Messages](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm#max-queue-messages), in `spec.systemScaling.maxQueueMessages` parameter
* [Maximum Spool Usage](https://docs.solace.com/Messaging/Guaranteed-Msg/Message-Spooling.htm#max-spool-usage), in `spec.systemScaling.maxSpoolUsage` parameter

In addition, for a given set of scaling parameters, the event broker container CPU and memory requirements must be calculated and provided in the `spec.systemScaling.cpu` and `spec.systemScaling.memory` parameters. Use the [System Resource Calculator](https://docs.solace.com/Admin-Ref/Resource-Calculator/pubsubplus-resource-calculator.html) to determine the CPU and memory requirements for the selected scaling parameters.

Example:
```yaml
spec:
  systemScaling:
    maxConnections: 100
    maxQueueMessages: 100
    maxSpoolUsage: 1000
    messagingNodeCpu: "2"
    messagingNodeMemory: "4025Mi"
```

>Note: Beyond CPU and memory requirements, broker storage size (see [Storage](#storage) section) must also support the provided scaling. The calculator can be used to determine that as well.

Also note, that specifying maxConnections, maxQueueMessages, and maxSpoolUsage on initial deployment overwrites the broker’s default values. On the other hand, doing the same using upgrade on an existing deployment does not overwrite these values on brokers configuration, but it can be used to prepare (first step) for a manual scale up using CLI where these parameter changes would actually become effective (second step).

##### Minimum footprint deployment for Developers

A minimum footprint deployment option is available for development purposes but with no guaranteed performance. The minimum available resources requirements are 1 CPU, 3.4 GiB memory and 7Gi of disk storage additional to the Kubernetes environment requirements.

To activate, set `spec.developer` to `true`.

>Important: If set to `true`, `spec.developer` has precedence over any `spec.systemScaling` vertical scaling settings.

##### Forward Compatibility for Vertical Scaling

Use `extraEnvVars`, which allows configuration of [additional properties for broker pods](#broker-pod-additional-properties) to ensure forward compatibility of `scalingParameters`.

For example from 10.7.x of Solace PubSub+ Software Event Broker, new scaling parameters have been added.

* [Maximum Number of Kafka Bridges](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm#max-kafka-bridges)
* [Maximum Number of Kafka Broker Connections](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm#max-kafka-broker-connections)

Configure these parameters outside of `systemScaling` using `extraEnvVars`, as shown below:

```yaml
apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  name: ha-scaling-param-extra
spec:
  extraEnvVars:
    - name: system_scaling_maxkafkabrokerconnectioncount
      value: "300"
    - name: system_scaling_maxkafkabridgecount
      value: "10"
  systemScaling:
    messagingNodeMemory: "8025Mi"
    messagingNodeCpu: "2"
    maxSpoolUsage: 500
    maxQueueMessages: 100
    maxConnections: 1000
  redundancy: false
```

When the broker pod has started, confirm consistency of values with the command `show system` after having [CLI access to individual event broker](#cli-access-to-individual-event-brokers).

```
ha-scaling-param-extra-pubsubplus-p-0> show system

System Uptime: 0d 0h 1m 13s
Last Restart Reason: Unknown reason

Scaling:
Max Connections: 1000
Max Queue Messages: 100M
Max Kafka Bridges: 10
Max Kafka Broker Connections: 300

Topic Routing:
Subscription Exceptions: Enabled
Subscription Exceptions Defer: Enabled
```

>Note: Please verify that the value aligns with the `scalingParameter` values. This custom method of configuring `scalingParameters` is only effective if it's supported as environment variables and values consistent with what the Solace PubSub+ Software Event Broker expects. Use the [Resource-Calculator](https://docs.solace.com/Admin-Ref/Resource-Calculator/pubsubplus-resource-calculator.html) as guide.

### Storage

The [PubSub+ deployment uses disk storage](https://docs.solace.com/Software-Broker/Configuring-Storage.htm) for logging, configuration, guaranteed messaging, and storing diagnostic and other information, allocated from Kubernetes volumes.

For a given set of [scaling](#vertical-scaling), use the [Solace online System Resource Calculator](https://docs.solace.com/Admin-Ref/Resource-Calculator/pubsubplus-resource-calculator.html) to determine the required storage size.

The broker pods can use following storage options:
* Dynamically allocated storage from a Kubernetes Storage Class (default)
* Static storage using a Persistent Volume Claim linked to a Persistent Volume
* Ephemeral storage

>Note: Ephemeral storage is generally not recommended. It might be acceptable for temporary deployments understanding that all configuration and messages are lost with the loss of the broker pod.

#### Dynamically allocated storage from a Storage Class

The recommended default allocation is using Kubernetes [Dynamic Volume Provisioning](https://kubernetes.io/docs/concepts/storage/dynamic-provisioning/) utilizing [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/). 

The StatefulSet controlling a broker pod creates a Persistent Volume Claim (PVC) specifying the requested size and the Storage Class of the volume; a Persistent Volume (PV) is allocated from the storage class pool that meets these requirements. Both the PVC and PV names are linked to the broker pod's name. If you delete the event broker pod(s) or even the entire deployment, the PVC and the allocated PV are not deleted, so potentially complex configuration is preserved. The PVC and PV are re-mounted and reused with the existing configuration when a new pod starts (controlled by the StatefulSet, automatically matched to the old pod even in an HA deployment) or when a deployment with the same as the old name is started. Explicitly delete a PVC if you no longer need it—this deletes the corresponding PV. For more information, see to [Deleting a Deployment](#deleting-a-deployment).

Example:
```yaml
spec:
  storage:
    messagingNodeStorageSize: 30Gi
    monitorNodeStorageSize: 3Gi
    # dynamic allocation
    useStorageClass: standard
```

For message processing brokers (this includes the single broker in non-HA deployment), the requested storage size is set using the `spec.storage.messagingNodeStorageSize` parameter. If not specified then the default value of `30Gi` is used. If the storage size is set to `0` then `useStorageClass` is disregarded and pod-local ephemeral storage is used.

When deploying PubSub+ in an HA redundancy group, monitoring broker nodes have minimal storage requirements compared to working nodes. It is recommended to leave the `spec.storage.monitorNodeStorageSize` parameter unspecified or at default. Although monitoring nodes will work with zero persistent (ephemeral) storage it is recommended to allocate the minimum so diagnostic information remains available with the loss of the monitoring pod.

##### Using an existing Storage Class

Set the `spec.storage.useStorageClass` parameter to use a particular storage class or leave this parameter to default undefined to allocate from your platform's "default" storage class - ensure it exists.
```bash
# Check existing storage classes
kubectl get storageclass
```

##### Creating a new Storage Class

Create a [new storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/#provisioner) if no existing storage class meets your needs and then specify to use that storage class. Refer to your Kubernetes environment's documentation if a StorageClass needs to be created or to understand the differences if there are multiple options. Example:
```yaml
# AWS fast storage class example
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast
provisioner: kubernetes.io/aws-ebs
parameters:
  type: io1
  fsType: xsf
```

If using NFS, or generally if allocating from a defined Kubernetes [Persistent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistent-volumes), specify a `storageClassName` in the PV manifest as in this NFS example, then set the `spec.storage.useStorageClass` parameter to the same:
```yaml
# Persistent Volume example
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv0003
spec:
  storageClassName: my-nfs
  capacity:
    storage: 15Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Recycle
  mountOptions:
    - hard
    - nfsvers=4.1
  nfs:
    path: /tmp
    server: 172.17.0.2
```
> Note: NFS is currently supported for development and demo purposes. If using NFS also set the `spec.storage.slow` parameter to `true`:
```yaml
spec:
  storage:
    messagingNodeStorageSize: 5Gi
    useStorageClass: my-nfs
    slow: true
```

#### Assigning existing PVC (Persistent Volume Claim)

You can to use an existing PVC with its associated PV for storage, but it must be taken into account that the deployment tries to use any existing, potentially incompatible, configuration data on that volume. The PV size must also meet the broker scaling requirements.

PVCs must be assigned individually to the brokers in an HA deployment. Assign a PVC to the Primary in case of non-HA.
```yaml
spec:
  storage:
    customVolumeMount:
      - name: Primary
        persistentVolumeClaim:
          claimName: my-primary-pvc-name
      - name: Backup
        persistentVolumeClaim:
          claimName: my-backup-pvc-name
      - name: Monitor
        persistentVolumeClaim:
          claimName: my-monitor-pvc-name
```

Note: Whenever existing PVC is reused, the deployment should maintain the same name to keep DNS configurations in sync. An out of sync DNS configuration will produce unintended consequences.

#### Storage solutions and providers

The PubSub+ Software Event Broker has been tested to work with Portworx, Ceph, Cinder (Openstack) and vSphere storage for Kubernetes as documented [here](https://docs.solace.com/Cloud/Deployment-Considerations/resource-requirements-k8s.htm#supported-storage-solutions).

Regarding providers, note that for [EKS](https://docs.solace.com/Cloud/Deployment-Considerations/installing-ps-cloud-k8s-eks-specific-req.htm) and [GKE](https://docs.solace.com/Cloud/Deployment-Considerations/installing-ps-cloud-k8s-gke-specific-req.htm#storage-class), `xfs` produced the best results during tests. [AKS](https://docs.solace.com/Cloud/Deployment-Considerations/installing-ps-cloud-k8s-aks-specific-req.htm) users can opt for `Local Redundant Storage (LRS)` redundancy which produced the best results when compared with other types available on Azure.

### Accessing Broker Services

Broker services (messaging, management) are available through the service ports of the [Broker Service](#event-broker-deployment) object created as part of the deployment.

Clients can access the service ports directly through a configured [standard Kubernetes service type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types). Alternatively, services can be mapped to Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress). These options are discussed in details in the upcoming [Using Service Type](#using-a-service-type) and [Using Ingress](#using-ingress) sections.
>Note: An OpenShift-specific alternative of exposing services through Routes is described in the [PubSub+ Openshift Deployment Guide](https://github.com/SolaceProducts/pubsubplus-openshift-quickstart/blob/master/docs/PubSubPlusOpenShiftDeployment.md).

Enabling TLS for services is recommended. For details, see [Configuring TLS for Services](#configuring-tls-for-broker-services).

Regardless the way to access services, the Service object is always used and it determines when and which broker Pod provides the actual service as explained in the next section.

#### Serving Pod Selection

The first criteria for a broker Pod to be selected for service is its readiness—if readiness is failing, Kubernetes stops sending traffic to the pod until it passes again.

The second, additional criteria is the pod label set to `active=true`.

Both pod readiness and label are updated periodically (every 5 seconds), triggered by the pod readiness probe. This probe invokes the `readiness_check.sh` script which is mounted on the broker container.

The requirements for a broker pod to satisfy both criteria are:
* The broker must be in Guaranteed Active service state, that is providing [Guaranteed Messaging Quality-of-Service (QoS) level of event messages persistence](https://docs.solace.com/PubSub-Basics/Guaranteed-Messages.htm). If service level is degraded even to [Direct Messages QoS](https://docs.solace.com/PubSub-Basics/Direct-Messages.htm) this is no longer sufficient.
* Management service must be up at the broker container level at port 8080.
* In an HA deployment, networking must enable the broker pods to communicate with each-other at the internal ports using the Service-Discovery service.
* The Kubernetes service account associated with the deployment must have sufficient rights to patch the pod's label when the active event broker is service ready
* The broker pods must be able to communicate with the Kubernetes API at kubernetes.default.svc.cluster.local at port $KUBERNETES_SERVICE_PORT. You can find out the address and port by SSH into the pod.

In summary, a deployment is ready for service requests when there is a broker pod that is running, `1/1` ready, and the pod's label is `active=true`. An exposed service port forwards traffic to that active event broker node. Pod readiness and labels can be checked with the command:
```
kubectl get pods --show-labels
```

#### Using a Service Type

[PubSub+ services](https://docs.solace.com/Configuring-and-Managing/Default-Port-Numbers.htm#Software) can be exposed using one of the following [Kubernetes service types](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types) by specifying the `spec.service.type` parameter:

* LoadBalancer (default) - a load balancer, typically externally accessible depending on the K8s provider.
* NodePort - maps PubSub+ services to a port on a Kubernetes node; external access depends on access to the Kubernetes node.
* ClusterIP - internal access only from within K8s.

To support [Internal load balancers](https://kubernetes.io/docs/concepts/services-networking/service/#internal-load-balancer), a provider-specific service annotation can be added by defining the `spec.service.annotations` parameter.

The `spec.service.ports` parameter defines the broker ports/services exposed. It specifies the event broker `containerPort` that provides the service and the mapping to the `servicePort` where the service can be accessed when using LoadBalancer or ClusterIP - however there is no control over the port number mapping when using NodePort. By default most broker service ports are exposed, refer to the ["pubsubpluseventbrokers" Custom Resource definition](/config/crd/bases/pubsubplus.solace.com_pubsubpluseventbrokers.yaml).

Example:
```yaml
spec:
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-internal: "true"
    ports:
      - servicePort: 55555
        containerPort: 55555
        protocol: TCP
        name: tcp-smf
      - ...
```

#### Configuring TLS for broker services

##### Setting up TLS

Default broker deployment does not have TLS over TCP enabled to access broker services. Although the exposed `spec.service.ports` include ports for secured TCP, only the insecure ports can be used by default.

To enable accessing services over TLS a server key and certificate must be configured.

It is assumed that you will use a third-party provider to create a server key and certificate for the event broker. The key and certificate must meet the requirements described in the [Solace Documentation](https://docs.solace.com/Configuring-and-Managing/Managing-Server-Certs.htm). If the server key is password protected it must be transformed to an unencrypted key, e.g.:  `openssl rsa -in encryedprivate.key -out unencryed.key`.

The server key and certificate must be packaged in a Kubernetes secret, for example by [creating a TLS secret](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets). Example:
```yaml
kubectl create secret tls <my-tls-secret> --key="<my-server-key-file>" --cert="<my-certificate-file>"
```

This secret name and related parameters must be specified in the broker spec:
```
spec:
  tls:
    enabled: true
    serverTlsConfigSecret: test-tls
    certFilename:    # optional, default if not provided: tls.crt 
    certKeyFilename: # optional, default if not provided: tls.key
```

> Note: Ensure that all filenames match those reported when you run `kubectl describe secret <my-tls-secret>`.

Important: It is not possible to update an existing deployment (created without TLS) to enable TLS using the [update deployment](#modifying-a-broker-deployment-including-broker-upgrade) procedure. In this case, for the first time, certificates must be [manually loaded and set up](https://docs.solace.com/Configuring-and-Managing/Managing-Server-Certs.htm) on each broker node. After that it is possible to use update with a secret specified.

##### Rotating the TLS certificate

In the event the server key or certificate must be rotated the TLS Config Secret must be updated or recreated with the new contents. Alternatively a new secret can be created and the broker spec can be updated with that secret's name.

If you are reusing an existing TLS secret, the new contents are automatically mounted on the broker containers. The Operator is already watching the configured secret for any changes and automatically initiates a rolling pod restart to take effect. Deleting the existing TLS secret does not result in immediate action, but broker pods will not start if the specified TLS secret does not exist.

> Note: A pod restart results in provisioning the server certificate from the secret again, so it reverts back from any other server certificate that might have been provisioned on the broker through another mechanism.

#### Using Ingress

The `LoadBalancer` or `NodePort` service types can be used to expose all services from one PubSub+ broker (one-to-one relationship). [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress) can be used to enable efficient external access from a single external IP address to multiple PubSub+ services, potentially provided by multiple brokers.

The following table gives an overview of how external access can be configured for PubSub+ services via Ingress.

| PubSub+ service / protocol, configuration and requirements | HTTP, no TLS | HTTPS with TLS terminate at ingress | HTTPS with TLS re-encrypt at ingress | General TCP over TLS with passthrough to broker |
| -- | -- | -- | -- | -- |
| **Notes:** | -- | Requires TLS config on Ingress-controller | Requires TLS config on broker AND TLS config on Ingress-controller | Requires TLS config on broker. Client must use SNI to provide target host |
| WebSockets, MQTT over WebSockets | Supported | Supported | Supported | Supported (routing via SNI) |
| REST | Supported with restrictions: if publishing to a Queue, only root path is supported in Ingress rule or must use [rewrite target](https://github.com/kubernetes/ingress-nginx/blob/main/docs/examples/rewrite/README.md) annotation. For Topics, the initial path would make it to the topic name. | Supported, see prev. note | Supported, see prev. note | Supported (routing via SNI) |
| SEMP | Not recommended to expose management services without TLS | Supported with restrictions: (1) Only root path is supported in Ingress rule or must use [rewrite target](https://github.com/kubernetes/ingress-nginx/blob/main/docs/examples/rewrite/README.md) annotation; (2) Non-TLS access to SEMP [must be enabled](https://docs.solace.com/Configuring-and-Managing/configure-TLS-broker-manager.htm) on broker | Supported with restrictions: only root path is supported in Ingress rule or must use [rewrite target](https://github.com/kubernetes/ingress-nginx/blob/main/docs/examples/rewrite/README.md) annotation | Supported (routing via SNI) |
| SMF, SMF compressed, AMQP, MQTT | - | - | - | Supported (routing via SNI) |
| SSH* | - | - | - | - |

*SSH has been listed here for completeness only, external exposure not recommended.

##### Configuration examples

All examples assume NGINX used as ingress controller ([documented here](https://kubernetes.github.io/ingress-nginx/)), selected because NGINX is supported by most K8s providers. For [other ingress controllers](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/#additional-controllers) refer to their respective documentation.

To deploy the NGINX Ingress Controller, refer to the [Quick start in the NGINX documentation](https://kubernetes.github.io/ingress-nginx/deploy/#quick-start). After successful deployment get the ingress External-IP or FQDN with the following command:

`kubectl get service ingress-nginx-controller --namespace=ingress-nginx`

This is the IP (or the IP address the FQDN resolves to) of the ingress where external clients must target their request and any additional DNS-resolvable hostnames, used for name-based virtual host routing, must also be configured to resolve to this IP address. If using TLS then the host certificate Common Name (CN) and/or Subject Alternative Name (SAN) must be configured to match the respective FQDN.

For options to expose multiple services from potentially multiple brokers, review the [Types of Ingress from the Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#types-of-ingress).
 
The next examples provide Ingress manifests that can be applied using `kubectl apply -f <manifest-yaml>`. Then check that an external IP address (ingress controller external IP) has been assigned to the rule/service and also that the host/external IP is ready for use because it could take a some time for the address to be populated.

```
kubectl get ingress
NAME                              CLASS   HOSTS
ADDRESS         PORTS   AGE
example.address                   nginx   frontend.host
20.120.69.200   80      43m
```

##### HTTP, no TLS

The following example configures ingress to [access PubSub+ REST service](https://docs.solace.com/Services/Configuring-EventBroker-for-REST.htm). Replace `<my-pubsubplus-service>` with the name of the service of your deployment (hint: the service name is similar to your pod names). The port name must match the `service.ports` name in the broker spec file.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: http-plaintext-example
spec:
  ingressClassName: nginx
  rules:
  - http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: <my-pubsubplus-service>
            port:
              name: tcp-rest
```

External requests must be targeted to the ingress External-IP at the HTTP port (80) and the specified path.

##### HTTPS with TLS terminate at ingress

Additional to above, this requires specifying a target virtual DNS-resolvable host (here `https-example.foo.com`), which resolves to the ingress External-IP, and a `tls` section. The `tls` section provides the possible hosts and corresponding [TLS secret](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls) that includes a private key and a certificate. The certificate must include the virtual host FQDN in its CN and/or SAN, as described above. Hint: [TLS secrets can be easily created from existing files](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets).

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: https-ingress-terminated-tls-example
spec:
  ingressClassName: nginx
  tls:
  - hosts:
      - https-example.foo.com
    secretName: testsecret-tls
  rules:
  - host: https-example.foo.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: <my-pubsubplus-service>
            port:
              name: tcp-rest
```

External requests must be targeted to the ingress External-IP through the defined hostname (here `https-example.foo.com`) at the TLS port (443) and the specified path.


##### HTTPS with TLS re-encrypt at ingress

This only differs from above in that the request is forwarded to a TLS-encrypted PubSub+ service port. The broker must have TLS configured but there are no specific requirements for the broker certificate as the ingress does not enforce it.

The difference in the Ingress manifest is an NGINX-specific annotation marking that the backend is using TLS, and the service target port in the last line - it refers now to a TLS backend port:

```yaml
metadata:
  :
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: HTTPS
  :
spec:
  :
  rules:
  :
            port:
              name: tls-rest
```

##### General TCP over TLS with passthrough to broker

In this case the ingress does not terminate TLS; it only provides routing to the broker Pod based on the hostname provided in the SNI extension of the Client Hello at TLS connection setup. Because it passes TLS traffic directly through to the broker as opaque data, any TCP-based protocol using TLS as transport is enabled for ingress.

The TLS passthrough capability must be explicitly enabled on the NGINX ingress controller, because it is off by default. This can be done by editing the `ingress-nginx-controller` "Deployment" in the `ingress-nginx` namespace.
1. Open the controller for editing: `kubectl edit deployment ingress-nginx-controller --namespace ingress-nginx`
2. Search where the `nginx-ingress-controller` arguments are provided, insert `--enable-ssl-passthrough` to the list and save. For more information refer to the [NGINX User Guide](https://kubernetes.github.io/ingress-nginx/user-guide/tls/#ssl-passthrough). Also note the potential performance impact of using SSL Passthrough mentioned here.

The Ingress manifest specifies "passthrough" by adding the `nginx.ingress.kubernetes.io/ssl-passthrough: "true"` annotation.

The deployed PubSub+ broker(s) must have TLS configured with a certificate that includes DNS names in CN and/or SAN, that match the host used. In the example, the broker server certificate can specify the host `*.broker1.bar.com`, so multiple services can be exposed from `broker1`, distinguished by the host FQDN.

The protocol client must support SNI. It depends on the client if it uses the server certificate CN or SAN for host name validation. Most recent clients use SAN, for example the PubSub+ Java API requires host DNS names in the SAN when using SNI.

With above, an ingress example looks following:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-passthrough-tls-example
  annotations:
    nginx.ingress.kubernetes.io/ssl-passthrough: "true"
spec:
  ingressClassName: nginx
  rules:
  - host: smf.broker1.bar.com
    http:
      paths:
      - backend:
          service:
            name: <my-pubsubplus-service>
            port:
              name: tls-smf
        path: /
        pathType: ImplementationSpecific
```
External requests must be targeted to the ingress External-IP through the defined hostname (here `smf.broker1.bar.com`) at the TLS port (443) with no path required.

### Broker Pod additional properties

The broker spec enables additional configurations that affect each broker pod or the broker containers of the deployment.

Example:
```yaml
spec:
  podLabels:
    "my-label": "my-label-value"
  podAnnotations:
    "my-annotation": "my-annotation-value"
  extraEnvVars:
    - name: TestType
      value: "GithubAction"
  extraEnvVarsCM: "my-env-variables-configmap"
  extraEnvVarsSecret: "my-env-variables-secret"
  timezone: UTC
```
Additional Pod labels and annotations can be specified using `spec.podLabels` and `spec.podAnnotations`.

Additional environment variables can be passed to the broker container in the form of
* a name-value list of variables, using `spec.extraEnvVars`
* providing the name of a ConfigMap that contains env variable names and values, using `spec.extraEnvVarsCM`
* providing the name of a secret that contains env variable names and values, using `spec.extraEnvVarsSecret`

One of the primary use of environment variables is to define [configuration keys](https://docs.solace.com/Software-Broker/Configuration-Keys-Reference.htm) that are consumed and applied at the broker initial deployment. It must be noted that configuration keys are ignored thereafter so they won't take effect even if updated later.

Finally, the timezone can be passed to the the event broker container.

### Security Considerations

The default installation of the Operator is optimized for easy deployment and getting started in a wide range of Kubernetes environments even by developers. Production use requires more tightened security. This section provides considerations for that.

#### Operator controlled namespaces

The Operator can be configured with a list of namespaces to watch, so it picks up all broker specs created in the watched namespaces and create deployments there. However all other namespaces are ignored.

Watched namespaces can be configured by providing the comma-separated list of namespaces in the `WATCH_NAMESPACE` environment variable defined in the container spec of the [Deployment](#operator) which controls the Operator pod. Assigning an empty string (default) means watching all namespaces.

It is recommended to restrict the watched namespaces for Production use. It is generally also recommended to not include the Operator's own namespace in the list because it is easier to separate RBAC settings for the operator from the broker's deployment - see next section.

#### Operator RBAC

The Operator requires CRUD permissions to manage all broker deployment resource types (e.g.: secrets) and the broker spec itself. This is defined in a ClusterRole which is bound to the Operator's service account using a ClusterRoleBinding if using the default Operator deployment. This enables the Operator to manage any of those resource types in all namespaces even if they don't belong to a broker deployment.

This needs to be restricted in a Production environment by creating a service account for the Operator in each watched namespace and use RoleBinding to bind the defined ClusterRole in each.

#### Broker deployment RBAC

A broker deployment only needs permission to update pod labels. This is defined in a Role, and a RoleBinding is created to the ServiceAccount used for the deployment. Note that without this permission the deployment will not work.

#### Operator image from private registry

The default deployment of the Operator pulls the Operator image from a public registry. If a Production deployment needs to pull the Operator image from a private registry then the [Deployment](#operator) which controls the Operator pod requires `imagePullSecrets` added for that repo:

```
kind: Deployment
metadata:
  name: operator
  ...
spec:
  template:
    spec:
      imagePullSecrets:
        - name: regcred
      ...
      containers:
      - name: manager
        image: private/controller:latest
      ...
```

#### Admin and Monitor Users and Passwords

A management `admin` and a `monitor` user are created at the broker initial deployment, with [global access level](https://docs.solace.com/Admin/SEMP/SEMP-API-Archit.htm#Role-Bas) "admin" and "read-only", respectively. The passwords are auto-generated if not provided and are stored in Operator-generated secrets.

It is also possible to provide pre-existing secrets containing the respective passwords as in the following example:
```yaml
spec:
  adminCredentialsSecret: my-admin-credetials-secret
  monitoringCredentialsSecret: my-monitoring-credetials-secret
```

The secrets must contain following data files: `username_admin_password` and `username_monitor_password` with password contents, respectively.

>**Important**: These secrets are used at initial broker deployment to setup passwords. Changing the secret contents later will not result in password updates in the broker. However changing the secret contents at a later point is service affecting as scripts in the broker container itself are using the passwords stored to access the broker's own management service. To fix a password discrepancy, log into each broker pod and use [CLI password change](https://docs.solace.com/Admin/Configuring-Internal-CLI-User-Accounts.htm#Changing-CLI-User-Passwords) to set the password for the user account to the same as in the secret.

#### Secrets

Using secrets for storing passwords, TLS server keys and certificates in the broker deployment namespace follows Kubernetes recommendations.

In a Production environment additional steps are required to ensure there is only authorized access to these secrets. Is recommended to follow industry Kubernetes security best practices including setting tight RBAC permissions for the namespace and harden security of the API server's underlying data store (etcd).

#### Broker Security Context

The following container-level security context configuration is automatically set by the operator:

```
capabilities:
  drop:
    - ALL
privileged: false
runAsNonRoot: true
allowPrivilegeEscalation: false
```

Following additional settings are configurable using broker spec parameters:
```
spec:
  securityContext:
    runAsUser: 1000001
    fsGroup: 1000002
```
Above are generally the defaults if not provided. It must be noted that the Operator detects whether the current Kubernetes environment is OpenShift. In that case, if not provided, the default `runAsUser` and `fsGroup` are set to unspecified because otherwise they would conflict with the OpenShift "restricted" Security Context Constraint settings for a project.

On top of pod securityContext, container securityContext can also be configured. It will be set on pubsubplus container. 
```
spec:
  container:
    securityContext:
      runAsNonRoot: true
      runAsGroup: 1000001
      runAsUser: 1000001
      allowPrivilegeEscalation: false
      privileged: false
      capabilities:
        drop:
          - ALL
      seLinuxOptions:
        level: s0:c123,c456
        role: object_r
        type: svirt_sandbox_file_t
        user: system_u
      seccompProfile:
        type: RuntimeDefault
```

#### Using Network Policies

In a controlled environment it might be necessary to configure a [NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/ ) to enable [required communication](#serving-pod-selection) between the broker nodes as well as between the broker container and the API server to set the Pod label.

##	Exposing Metrics to Prometheus

Refer to the [Prometheus Monitoring Support section](#prometheus-monitoring-support) for an overview of how metrics are exposed.

This section describes how to enable and configure the metrics exporter and the available metrics from the broker deployment, configure Prometheus to use that, and finally an example setup of Grafana to visualize broker metrics.

### Enabling and configuring the Broker Metrics Endpoint

To enable monitoring with all defaults, simply add `spec.monitoring.enabled: true` to the broker spec. This sets up a metrics service endpoint through a Prometheus Metrics Service that offers a REST API that responds with broker metrics to GET requests.

The next more advanced example shows a configuration with additional configuration to specify that the exporter image is pulled from a private repo using a pull secret, the service type is set to Kubernetes internal `ClusterIP`, and also TLS is enabled for the service with key and certificate contained in Secret `monitoring-tls`. The way to create the Secret is the same as for the [broker TLS configuration](#configuring-tls-for-broker-services).
```yaml
spec:
  monitoring:
    enabled: true
    image:
      repository: solace/pubsubplus-prometheus-exporter
      tag: latest
      pullSecrets:
      - name: regcred
    metricsEndpoint:
      listenTLS: true
      serviceType: ClusterIP   # This is the default, exposes service within Kubernetes only      
      endpointTlsConfigSecret: monitoring-tls
```

### Available Broker Metrics

The broker metrics are exposed through the Prometheus Metrics Service REST API (GET only) at port 9628:
```bash
kubectl describe svc <eventbroker-deployment-name>-pubsubplus-prometheus-metrics
```

There are two sets of metrics exposed through two paths:
* Standard: [http://`<service-ip>`:9628/solace-std]()
* Additional Details: [http://`<service-ip>`:9628/solace-det]()
>Note: The `<service-ip>` address is the Kubernetes internal ClusterIP address of the Prometheus Metrics Service.  Use `kubectl port-forward svc/<eventbroker-deployment-name>-pubsubplus-prometheus-metrics 9628` to expose it through your `localhost` for testing.

The following table lists the metrics exposed by the paths:

| Path | Definition | Name | Type |
| --- | --- | --- | --- |
| **`solace-std`** |
|| Max number of Local Bridges | solace_bridges_max_num_local_bridges | gauge
|| Max number of Remote Bridges | solace_bridges_max_num_remote_bridges | gauge
|| Max number of Bridges | solace_bridges_max_num_total_bridges | gauge
|| Max total number of Remote Bridge Subscription | solace_bridges_max_num_total_remote_bridge_subscriptions | gauge
|| Number of Local Bridges | solace_bridges_num_local_bridges | gauge
|| Number of Remote Bridges | solace_bridges_num_remote_bridges | gauge
|| Number of Bridges | solace_bridges_num_total_bridges | gauge
|| Total number of Remote Bridge Subscription | solace_bridges_num_total_remote_bridge_subscriptions | gauge
|| Config Sync Ownership (0-Master, 1-Slave, 2-Unknown) | solace_configsync_table_ownership | gauge
|| Config Sync State (0-Down, 1-Up, 2-Unknown, 3-In-Sync, 4-Reconciling, 5-Blocked, 6-Out-Of-Sync) | solace_configsync_table_syncstate | gauge
|| Config Sync Time in State | solace_configsync_table_timeinstateseconds | counter
|| Config Sync Resource (0-Router, 1-Vpn, 2-Unknown, 3-None, 4-All) | solace_configsync_table_type | gauge
|| Average compute latency | solace_system_compute_latency_avg_seconds | gauge
|| Current compute latency | solace_system_compute_latency_cur_seconds | gauge
|| Maximum compute latency | solace_system_compute_latency_max_seconds | gauge
|| Minimum compute latency | solace_system_compute_latency_min_seconds | gauge
|| Average disk latency | solace_system_disk_latency_avg_seconds | gauge
|| Current disk latency | solace_system_disk_latency_cur_seconds | gauge
|| Maximum disk latency | solace_system_disk_latency_max_seconds | gauge
|| Minimum disk latency | solace_system_disk_latency_min_seconds | gauge
|| Average mate link latency | solace_system_mate_link_latency_avg_seconds | gauge
|| Current mate link latency | solace_system_mate_link_latency_cur_seconds | gauge
|| Maximum mate link latency | solace_system_mate_link_latency_max_seconds | gauge
|| Minimum mate link latency | solace_system_mate_link_latency_min_seconds | gauge
|| Redundancy configuration (0-Disabled, 1-Enabled, 2-Shutdown) | solace_system_redundancy_config | gauge
|| Is local node the active messaging node? (0-not active, 1-active). | solace_system_redundancy_local_active | gauge
|| Redundancy role (0=Backup, 1=Primary, 2=Monitor, 3-Undefined). | solace_system_redundancy_role | gauge
|| Is redundancy up? (0=Down, 1=Up). | solace_system_redundancy_up | gauge
|| Total disk usage in percent | solace_system_spool_disk_partition_usage_active_percent | gauge
|| Total disk usage of mate instance in percent | solace_system_spool_disk_partition_usage_mate_percent | gauge
|| Utilization of spool files in percent | solace_system_spool_files_utilization_percent | gauge
|| Spool configured max disk usage | solace_system_spool_quota_bytes | gauge
|| Spool configured max number of messages | solace_system_spool_quota_msgs | gauge
|| Spool total persisted usage | solace_system_spool_usage_bytes | gauge
|| Spool total number of persisted messages | solace_system_spool_usage_msgs | gauge
|| Solace Version as WWWXXXYYYZZZ  | solace_system_version_currentload | gauge
|| Broker uptime in seconds  | solace_system_version_uptime_totalsecs | gauge
|| Was the last scrape of Solace broker successful? | solace_up | gauge
|| Number of connections | solace_vpn_connections | gauge
|| Total number of AMQP connections | solace_vpn_connections_service_amqp | gauge
|| Total number of SMF connections | solace_vpn_connections_service_smf | gauge
|| VPN is enabled | solace_vpn_enabled | gauge
|| VPN is a management VPN | solace_vpn_is_management_vpn | gauge
|| Local status (0=Down, 1=Up) | solace_vpn_local_status | gauge
|| VPN is locally configured | solace_vpn_locally_configured | gauge
|| VPN is operational | solace_vpn_operational | gauge
|| Maximum number of connections | solace_vpn_quota_connections | gauge
|| Replication Admin Status (0-shutdown, 1-enabled, 2-n/a) | solace_vpn_replication_admin_state | gauge
|| Replication Config Status (0-standby, 1-active, 2-n/a) | solace_vpn_replication_config_state | gauge
|| Replication Tx Replication Mode (0-async, 1-sync) | solace_vpn_replication_transaction_replication_mode | gauge
|| Spool configured max disk usage | solace_vpn_spool_quota_bytes | gauge
|| Spool total persisted usage | solace_vpn_spool_usage_bytes | gauge
|| Spool total number of persisted messages | solace_vpn_spool_usage_msgs | gauge
|| Total unique local subscriptions count | solace_vpn_total_local_unique_subscriptions | gauge
|| Total unique remote subscriptions count | solace_vpn_total_remote_unique_subscriptions | gauge
|| Total unique subscriptions count | solace_vpn_total_unique_subscriptions | gauge
|| Total subscriptions count | solace_vpn_unique_subscriptions | gauge
| **`solace-det`** |
|| Is client a slow subscriber? (0=not slow, 1=slow) | solace_client_slow_subscriber | gauge
|| Number of clients bound to queue | solace_queue_binds | gauge
|| Number of discarded received messages | solace_client_rx_discarded_msgs_total | counter
|| Number of discarded received messages | solace_vpn_rx_discarded_msgs_total | counter
|| Number of discarded transmitted messages | solace_client_tx_discarded_msgs_total | counter
|| Number of discarded transmitted messages | solace_vpn_tx_discarded_msgs_total | counter
|| Number of received bytes | solace_client_rx_bytes_total | counter
|| Number of received bytes | solace_vpn_rx_bytes_total | counter
|| Number of received messages | solace_client_rx_msgs_total | counter
|| Number of received messages | solace_vpn_rx_msgs_total | counter
|| Number of transmitted bytes | solace_client_tx_bytes_total | counter
|| Number of transmitted bytes | solace_vpn_tx_bytes_total | counter
|| Number of transmitted messages | solace_client_tx_msgs_total | counter
|| Number of transmitted messages | solace_vpn_tx_msgs_total | counter
|| Queue spool configured max disk usage in bytes | solace_queue_spool_quota_bytes | gauge
|| Queue spool total of all spooled messages in bytes | solace_queue_byte_spooled | gauge
|| Queue spool total of all spooled messages | solace_queue_msg_spooled | gauge
|| Queue spool usage in bytes | solace_queue_spool_usage_bytes | gauge
|| Queue spooled number of messages | solace_queue_spool_usage_msgs | gauge
|| Queue total msg redeliveries | solace_queue_msg_redelivered | gauge
|| Queue total msg retransmitted on transport | solace_queue_msg_retransmited | gauge
|| Queue total number of messages delivered to dmq due to exceeded max redelivery | solace_queue_msg_max_redelivered_dmq | gauge
|| Queue total number of messages delivered to dmq due to ttl expiry | solace_queue_msg_ttl_dmq | gauge
|| Queue total number of messages discarded due to exceeded max redelivery | solace_queue_msg_max_redelivered_discarded | gauge
|| Queue total number of messages discarded due to spool shutdown | solace_queue_msg_shutdown_discarded | gauge
|| Queue total number of messages discarded due to ttl expiry | solace_queue_msg_ttl_discarded | gauge
|| Queue total number of messages exceeded the max message size | solace_queue_msg_max_msg_size_exceeded | gauge
|| Queue total number of messages exceeded the spool usage | solace_queue_msg_spool_usage_exceeded | gauge
|| Queue total number of messages failed delivery to dmq due to exceeded max redelivery | solace_queue_msg_max_redelivered_dmq_failed | gauge
|| Queue total number of messages that failed delivery to dmq due to ttl expiry | solace_queue_msg_ttl_dmq_failed | gauge
|| Queue total number that was deleted | solace_queue_msg_total_deleted | gauge
|| Was the last scrape of Solace broker successful | solace_up | gauge


### Connecting with Prometheus

With the metrics endpoint of the broker deployment enabled and up, it is matter of configuring Prometheus to add this endpoint to its list of scraped targets. The way to configure Prometheus is highly dependent on how it has been deployed including whether it is inside or outside the Kubernetes cluster.

For reference, this guide shows how to set up a Prometheus deployment, created and managed by the Prometheus Operator. Consult your documentation and adjust the procedure if your Prometheus environment differs.

#### Reference Prometheus Stack Deployment

This section describes the setup of a reference Prometheus stack that includes Prometheus and Grafana (and also other Prometheus components not used here). We use the [kube-prometheus project](https://github.com/prometheus-operator/kube-prometheus) which not only includes the Prometheus Operator, but also Grafana. There are some adjustments/workarounds needed as described below.

Steps:
1. Git clone the `kube-prometheus` project. These steps were tested with the tagged version, later versions might well work too.
```
git clone https://github.com/prometheus-operator/kube-prometheus.git --tag v0.12.0
```
2. Follow `kube-prometheus` Quickstart steps: https://github.com/prometheus-operator/kube-prometheus#quickstart . These steps deploy the required operators and create a Prometheus stack in the `monitoring` namespace.
3. Patch the `prometheus-k8s` ClusterRole to enable access to the event broker metrics service. Run `kubectl edit ClusterRole prometheus-k8s` and append following to the `rules` section, then save:
```
- apiGroups:
  - ""
  resources:
  - services
  - pods
  - endpoints
  verbs:
  - get
  - list
  - watch
```
4. The datasource for Grafana needs to be adjusted to use the `prometheus-operated` service from the `monitoring` namespace. This is configured in the `grafana-datasources` secret in the same namespace. Run `kubectl edit secret grafana-datasources -n monitoring`, then replace the `data.datasources.yaml` section as follows, then save:
```
data:
  datasources.yaml: ewogICAgImFwaVZlcnNpb24iOiAxLAogICAgImRhdGFzb3VyY2VzIjogWwogICAgICAgIHsKICAgICAgICAgICAgImFjY2VzcyI6ICJwcm94eSIsCiAgICAgICAgICAgICJlZGl0YWJsZSI6IGZhbHNlLAogICAgICAgICAgICAibmFtZSI6ICJwcm9tZXRoZXVzIiwKICAgICAgICAgICAgIm9yZ0lkIjogMSwKICAgICAgICAgICAgInR5cGUiOiAicHJvbWV0aGV1cyIsCiAgICAgICAgICAgICJ1cmwiOiAiaHR0cDovL3Byb21ldGhldXMtb3BlcmF0ZWQubW9uaXRvcmluZy5zdmM6OTA5MCIsCiAgICAgICAgICAgICJ2ZXJzaW9uIjogMQogICAgICAgIH0KICAgIF0KfQ==
```
>Note: Because this is data stored in a Kubernetes secret, you must provide Base64-encoded data. Use an [online Base64 decode tool](https://www.base64decode.org/) to reveal the unencoded content of the data above.
5. Restart the pods in the `monitoring` namespace to pick up the changes:
```bash
kubectl delete pods --all -n monitoring
# wait for all pods come back up all ready
kubectl get pods --watch -n monitoring
```

Now both Prometheus and Grafana are running. Their Web Management UIs are exposed through the services `prometheus-k8s` at port 9090 and `grafana` at port 3000 in the `monitoring` namespace. Because these services are of type ClusterIP one of the options is to use Kubectl port-forwarding to access them:
```
kubectl port-forward svc/prometheus-k8s 9090 -n monitoring &
kubectl port-forward svc/grafana 3000 -n monitoring &
```
Point your browser to [localhost:9090](http://localhost:9090) for Prometheus and to [localhost:3000](http://localhost:3000) for Grafana. An initial login might be required using the credentials `admin/admin`.

#### Creating a ServiceMonitor object

With the adjustments discussed above, the Prometheus Operator is now watching all namespaces for `ServiceMonitor` custom resource objects. A `ServiceMonitor` defines which metrics services must be added to the Prometheus targets. It is namespace scoped so it must be added to the namespace where the event broker has been deployed.

Example:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-monitor
spec:
  endpoints:
    - interval: 10s
      path: /solace-std
      port: tcp-metrics
    - interval: 10s
      path: /solace-det
      port: tcp-metrics
  jobLabel: pubsubplus-metrics
  selector:
    matchLabels:
      app.kubernetes.io/name: pubsubpluseventbroker
      app.kubernetes.io/component: metricsexporter
      app.kubernetes.io/instance: <eventbroker-deployment-name>
```
This adds the deployment's metrics service (by matching labels) to the Prometheus targets. Refresh the Prometheus Status "Targets" in the Prometheus Web Management UI to see the newly added target.

The ServiceMonitor's selector can be adjusted to match all broker deployments in the namespace by removing `instance` from the matched labels. Also, multiple endpoints can be listed to obtain the combination of metrics from those Exporter paths.

The `ServiceMonitor` example above specifies that three target endpoints are scraped to get the combination of all metrics available from the broker deployment. The metrics endpoints can be accessed at the port named `tcp-metrics` and at the PubSub+ Exporter path, e.g.: `/solace-std` is added to the scrape request REST API calls.

### Grafana Visualization of Broker Metrics

In the Grafana Web Management UI, select "Dashboards"->"Import dashboard"->"Upload JSON File". Upload `deploy/grafana_example.json`. This opens a sample Grafana dashboard. The following image shows this sample rendered after running some messaging traffic through the broker deployment.

To create or customize your own dashboard refer to the [Grafana documentation](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/).

![alt text](/docs/images/GrafanaDashboard.png "Grafana dashboard example")

## Broker Deployment Guide

### Quick Start

Refer to the [Quick Start guide](/README.md) in the root of this repo. It also provides information about deployment pre-requisites and tools.

Example:
```sh
# Initial deployment
kubectl apply -f <initial-broker-spec>.yaml
# Wait for the deployment to come up ...
```


###	Validating the deployment

You can validate your deployment on the command line. In this example an HA configuration is deployed with name `ha-example`, created using the [Quick Start](#quick-start).

```sh
prompt:~$ kubectl get statefulsets,services,pods,pvc,pv
NAME                                       READY   AGE
statefulset.apps/ha-example-pubsubplus-b   1/1     1h 
statefulset.apps/ha-example-pubsubplus-m   1/1     1h 
statefulset.apps/ha-example-pubsubplus-p   1/1     1h 

NAME                                               TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)
                                                                                                                                                                                AGE
service/ha-example-pubsubplus                      LoadBalancer   10.124.2.72     35.238.219.112   2222:31209/TCP,8080:31536/TCP,1943:30396/TCP,51234:31106/TCP,55003:31764/TCP,55443:32625/TCP,55556:30149/TCP,8008:30054/TCP,1443:32480/TCP,9000:31032/TCP,9443:30728/TCP,5672:31944/TCP,5671:30878/TCP,1883:31123/TCP,8883:31873/TCP,8000:31970/TCP,8443:32172/TCP   25h
service/ha-example-pubsubplus-discovery            ClusterIP      None            <none>           8080/TCP,8741/TCP,8300/TCP,8301/TCP,8302/TCP
                                                                                                                                                                                1h 
service/ha-example-pubsubplus-prometheus-metrics   ClusterIP      10.124.15.107   <none>           9628/TCP
                                                                                                                                                                                1h 
service/kubernetes                                 ClusterIP      10.124.0.1      <none>           443/TCP
                                                                                                                                                                                1h 

NAME                                                             READY   STATUS    RESTARTS   AGE
pod/ha-example-pubsubplus-b-0                                    1/1     Running   0          1h 
pod/ha-example-pubsubplus-m-0                                    1/1     Running   0          1h 
pod/ha-example-pubsubplus-p-0                                    1/1     Running   0          1h 
pod/ha-example-pubsubplus-prometheus-exporter-5cdfcd64b4-dbl2j   1/1     Running   0          1h 

NAME                                                   STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
persistentvolumeclaim/data-ha-example-pubsubplus-b-0   Bound    pvc-6de2275b-9731-417b-9e54-341dec2ffa40   30Gi       RWO            standard-rwo   1h 
persistentvolumeclaim/data-ha-example-pubsubplus-m-0   Bound    pvc-3c1f3799-fa82-45a2-883a-d5ed52637783   3Gi        RWO            standard-rwo   1h 
persistentvolumeclaim/data-ha-example-pubsubplus-p-0   Bound    pvc-4d05e27e-007d-4a4f-a08a-93a2c41005c1   30Gi       RWO            standard-rwo   1h 

NAME                                                        CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                    STORAGECLASS   REASON   AGE
persistentvolume/pvc-3c1f3799-fa82-45a2-883a-d5ed52637783   3Gi        RWO            Delete           Bound    default/data-ha-example-pubsubplus-m-0   standard-rwo            1h 
persistentvolume/pvc-4d05e27e-007d-4a4f-a08a-93a2c41005c1   30Gi       RWO            Delete           Bound    default/data-ha-example-pubsubplus-p-0   standard-rwo            1h 
persistentvolume/pvc-6de2275b-9731-417b-9e54-341dec2ffa40   30Gi       RWO            Delete           Bound    default/data-ha-example-pubsubplus-b-0   standard-rwo            1h 

prompt:~$ kubectl describe service ha-example-pubsubplus
Name:                     ha-example-pubsubplus
Namespace:                default
Labels:                   app.kubernetes.io/instance=ha-example
                          app.kubernetes.io/managed-by=solace-pubsubplus-operator
                          app.kubernetes.io/name=pubsubpluseventbroker
Annotations:              cloud.google.com/neg: {"ingress":true}
                          lastAppliedConfig/brokerService: 3a87fe83d04ddd7f
Selector:                 active=true,app.kubernetes.io/instance=ha-example,app.kubernetes.io/name=pubsubpluseventbroker
Type:                     LoadBalancer
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       10.124.2.72
IPs:                      10.124.2.72
LoadBalancer Ingress:     35.238.219.112
Port:                     tcp-ssh  2222/TCP
TargetPort:               2222/TCP
NodePort:                 tcp-ssh  31209/TCP
Endpoints:                10.120.1.6:2222
Port:                     tcp-semp  8080/TCP
TargetPort:               8080/TCP
NodePort:                 tcp-semp  31536/TCP
Endpoints:                10.120.1.6:8080
Port:                     tls-semp  1943/TCP
TargetPort:               1943/TCP
NodePort:                 tls-semp  30396/TCP
Endpoints:                10.120.1.6:1943
:
:
```

There are three StatefulSets controlling each broker node in an HA redundancy group, with naming conventions `<deployment-name>-pubsubplus-p` for Primary, `...-b` for Backup and `...-m` for Monitor brokers. Similarly, the broker pods are named `<deployment-name>-pubsubplus-p-0`, `...-b-0` and `...-m-0`. In case of a non-HA deployment there is one StatefulSet with the naming convention of `...-p`.

Generally, all services including management and messaging are accessible through a Load Balancer. In the above example `35.238.219.112` is the Load Balancer's external Public IP to use.

> Note: When using MiniKube or other minimal Kubernetes provider, there may be no integrated Load Balancer available, which is the default service type. For a workaround, either refer to the [MiniKube documentation for LoadBalancer access](https://minikube.sigs.k8s.io/docs/handbook/accessing/#loadbalancer-access) or use [local port forwarding to the service port](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/#forward-a-local-port-to-a-port-on-the-pod): `kubectl port-forward service/$BROKER_SERVICE_NAME <target-port-on-localhost>:<service-port-on-load-balancer> &`. Then access the service at `localhost:<target-port-on-localhost>`

### Gaining admin access to the event broker

The [PubSub+ Broker Manager](https://docs.solace.com/Admin/Broker-Manager/PubSub-Manager-Overview.htm) is the recommended simplest way to administer the event broker for common tasks.

#### Admin Credentials

The default admin username is `admin`. A password can be provided encoded in a Kubernetes secret in the broker spec parameter `spec.adminCredentialsSecret`. If not provided then a random password is generated at initial deployment and stored in a secret named `<eventbroker-deployment-name>-pubsubplus-admin-creds`.

For example you can create a secret `my-admin-secret` with `MyP@ssword` before deployment then pass its name to the broker spec:
```
echo 'MyP@ssword' | kubectl create secret generic my-admin-secret --from-file=username_admin_password=/dev/stdin
```

To obtain the admin password from a secret use:
```
kubectl get secret my-admin-secret -o jsonpath='{.data.username_admin_password}' | base64 -d
```

#### Management access port

Use the Load Balancer's external Public IP at port 8080 to access management services including PubSub+ Broker Manager, SolAdmin and SEMP access.

#### Broker CLI access via the load balancer

One option to access the event broker's CLI console is to SSH into the broker as the `admin` user using the Load Balancer's external Public IP, at port 2222:

```
prompt:~$ ssh -p 2222 admin@35.238.219.112
The authenticity of host '[35.238.219.112]:2222 ([35.238.219.112]:2222)' can't be established.
ECDSA key fingerprint is SHA256:iBVfUuHRh7r8stH4fv3CCzv7966UEK/ZfHTh2Yt79No.
Are you sure you want to continue connecting (yes/no/[fingerprint])? yes
Warning: Permanently added '[35.238.219.112]:2222' (ECDSA) to the list of known hosts.
Solace PubSub+ Standard
Password:

Solace PubSub+ Standard Version 10.2.1.32

This Solace product is proprietary software of
Solace Corporation. By accessing this Solace product
you are agreeing to the license terms and conditions
located at http://www.solace.com/license-software

Copyright 2004-2022 Solace Corporation. All rights reserved.

To purchase product support, please contact Solace at:
https://solace.com/contact-us/

Operating Mode: Message Routing Node

ha-example-pubsubplus-p-0>
```

This enables access only to the active Pod's CLI.

#### CLI access to individual event brokers

In an HA deployment, CLI access might be needed to any of the brokers, not only the active one.

* The simplest option from the Kubernetes command line console is
```
kubectl exec -it <broker-pod-name> -- cli
```

* Loopback to SSH directly on the pod

```
kubectl exec -it <broker-pod-name> -- bash -c "ssh -p 2222 admin@localhost"
```

* Loopback to SSH on your host with a port-forward map

```
kubectl port-forward <broker-pod-name> 62222:2222 &
ssh -p 62222 admin@localhost
```

This can also be mapped to multiple event brokers in the HA deployment via port-forward:

```
kubectl port-forward <primary-broker-pod-name> 8081:8080 &
kubectl port-forward <backup-broker-pod-name> 8082:8080 &
kubectl port-forward <monitor-broker-pod-name> 8083:8080 &
```

#### SSH access to individual event brokers

For direct access, use:

```sh
kubectl exec -it <broker-pod-name> -- bash
```

#### Testing data access to the event broker

The newly created event broker instance comes with a [basic configuration](https://docs.solace.com/Software-Broker/SW-Broker-Configuration-Defaults.htm) of a `default` client username with no authentication on the `default` message VPN.

An easy first test is using the [PubSub+ Broker Manager's built-in Try-Me tool](https://docs.solace.com/Admin/Broker-Manager/PubSub-Manager-Overview.htm?Highlight=manager#Test-Messages). Try-Me is based on JavaScript making use of the WebSockets API for messaging at port 8008.

To test data traffic using other supported APIs, visit the Solace Developer Portal [APIs & Protocols](https://www.solace.dev/ ). Under each option there is a Publish/Subscribe tutorial that will help you get started and provide the specific default port to use.

Use the external Public IP to access the deployment at the port required for the protocol.

### Troubleshooting

#### General Kubernetes troubleshooting hints
https://kubernetes.io/docs/tasks/debug/

#### Checking the reason for failed resources

Run `kubectl get statefulsets,services,pods,pvc,pv` to get an understanding of the state, then drill down to get more information on a failed resource to reveal  possible Kubernetes resourcing issues, e.g.:
```sh
kubectl describe pvc <pvc-name>
```

#### Viewing logs

The Operator, Broker, and Prometheus Exporter pods all provide logs that might be useful to understand issues.

Detailed logs from the currently running container in a pod:
```sh
kubectl logs <pod-name> -f  # use -f to follow live
```

It is also possible to get the logs from a previously terminated or failed container:
```sh
kubectl logs <pod-name> -p
```

Filtering on bringup logs (helps with initial troubleshooting):
```sh
kubectl logs <pod-name> | grep [.]sh
```

#### Updating log levels

The Operator uses a [zap-based](https://pkg.go.dev/go.uber.org/zap) logger. This means when
deploying the operator to a cluster you can set additional flags using an args array in your
operator’s container spec.
One such flags allows the log level to be updated.

One can extract the current log level with:

```sh
kubectl get deployment <deployment-name>  --namespace <namespace> -o=json | jq '.spec.template.spec.containers[0].args' 
```

The log levels available are: `--zap-log-level=debug`, `--zap-log-level=info`
and `--zap-log-level=error`.

Other configurations to note which can be added to the args array in the operator’s container spec
are:

`--zap-encoder`: To set log encoding. Options are `json` or `console`.

`--zap-stacktrace-level` : To set level at and above which stacktraces are captured. Options
are `info` or `error`.

Note that for OLM deployments however, a manual update to the Deployment will be reverted since OLM
automatically manages the CRD. This has to be done with the approved means of patching OLM
deployments.

#### Viewing events

Kubernetes collects [all events for a cluster in one pool](https://pwittrock.github.io/docs/tasks/debug-application-cluster/events-stackdriver). This includes events related to the PubSub+ deployment.

It is recommended to watch events when creating or upgrading a Solace deployment. Events clear after about an hour. You can query all available events:

```sh
kubectl get events -w # use -w to watch live
```

#### Pods issues

##### Pods stuck in not enough resources

If pods stay in pending state and `kubectl describe pods` reveals there are not enough memory or CPU resources, check the [resource requirements of the targeted scaling tier](#broker-scaling) of your deployment and ensure adequate node resources are available.

##### Pods stuck in no storage

Pods might also stay in pending state because [storage requirements](#storage) cannot be met. Check `kubectl get pv,pvc`. PVCs and PVs should be in bound state and if not then use `kubectl describe pvc` for any issues.

Unless otherwise specified, a default storage class must be available for default PubSub+ deployment configuration.
```bash
kubectl get storageclasses
```

##### Pods stuck in CrashLoopBackoff, Failed, or Not Ready

Pods stuck in CrashLoopBackoff, or Failed, or Running but not Ready "active" state, usually indicate an issue with available Kubernetes node resources or with the container OS or the event broker process start.

* Try to understand the reason following earlier hints in this section.
* Try to recreate the issue by deleting and then reinstalling the deployment - ensure to remove related PVCs if applicable because they would mount volumes with existing, possibly outdated or incompatible database - and watch the [logs](#viewing-logs) and [events](#viewing-events) from the beginning. Look for ERROR messages preceded by information that might reveal the issue.

##### No Pods listed

If no pods are listed related to your deployment check the StatefulSets for any clues:
```
kubectl describe statefulset | grep <broker-deployment-name>
```

#### Security constraints

Your Kubernetes environment's security constraints might also impact successful deployment. Review the [Security considerations](#security-considerations) section.

### Maintenance mode

When the Operator is running it is constantly stewarding the broker deployment artifacts and intervene in case of any deviation.

_Maintenance_ _mode_ enables that in special cases users can "turn off" the operator's control for a broker deployment. This can be done by adding a `solace.com/pauseReconcile=true` label to the broker spec:

```sh
# Activate maintenance mode by adding the pauseReconcile label
kubectl label eb <broker-deployment-name> solace.com/pauseReconcile=true
# Operator will now ignore this deployment
# ... 
# Remove the label to activate Operator's control again
kubectl label eb <broker-deployment-name> solace.com/pauseReconcile-
```

###	Modifying a Broker Deployment including Broker Upgrade

Modification of the broker deployment (or update) can be initiated by applying an updated broker spec with modified parameter values. Upgrade is a special modification where the broker's `spec.image.repository` and/or `spec.image.tag` has been modified.

>Note: There are limitations; some parameters cannot be modified or the updated values will be ignored. For details on these exceptions, see the [Update Limitations](#update-limitations) section.

Applying a modified manifest example:
```sh
# Initial deployment
kubectl apply -f <initial-broker-spec>.yaml
# Wait for the deployment to come up ...
#
# Update
kubectl apply -f <modified-broker-spec>.yaml
```

It is also possible to directly edit the current deployment spec (manifest).

Edit manifest example:
```
# Initial deployment
kubectl apply -f <initial-broker-spec>.yaml
# Wait for the deployment to come up ...
#
# Update
kubectl edit eventbroker <broker-deployment-name>
# Make changes to parameters then save
```

### Rolling vs. Manual Update

By default an update triggers the restart of the broker pods:
* In a non-HA deployment the single broker pod is restarted.
* In an HA deployment the three broker pods of the HA redundancy group are restarted in a **rolling** update: first the Monitor Broker pod, then the pod hosting the redundancy Standby Broker, and finally the pod hosting the currently Active Broker. When the currently active broker is terminated for update in the final step, an [automatic redundancy activity switch](https://docs.solace.com/Features/HA-Redundancy/SW-Broker-Redundancy-and-Fault-Tolerance.htm#Failure) happens where the already updated standby takes activity.

Users wishing to manually control the pod restarts can activate **manual** updates by specifying `spec.updateStrategy: manualPodRestart` in the broker spec. In this case the user is responsible for initiating the termination of the individual pods at their discretion. Removing or setting the value back to `automatedRolling` reverts to the rolling update mode.

### Update Limitations

The following table lists parameters for which update using [Modify Deployment](#modifying-a-broker-deployment-including-broker-upgrade) is not supported in the current Operator release.

| Parameter | Notes |
| --- | ---
| `spec.adminCredentialsSecret` | Changing the secret name or contained password does not update the password on the broker but does result in broker pods getting out of readiness. It requires an additional [manual action to update the admin password using CLI](https://docs.solace.com/Admin/Configuring-Internal-CLI-User-Accounts.htm?Highlight=admin%20password#Changing-CLI-User-Passwords) on *each* broker.
| `spec.monitoringCredentialsSecret` | Similarly to the `adminCredentialsSecret`, additional manual action is required to update the password of the `minitor` user on each broker. |
| `spec.preSharedAuthKeySecret` | Any updates are ignored, [manual change of key is required using CLI](https://docs.solace.com/Features/HA-Redundancy/Pre-Shared-Keys-SMB.htm?Highlight=pre-shared#How2).|
| `spec.systemScaling.maxConnections` | Scaling up a broker deployment requires two steps: first, to update the deployment with the desired target `systemScaling` and next, to [manually update scaling using the CLI](https://docs.solace.com/Software-Broker/Set-Scaling-Params-HA.htm#Step_2__Increase_the_Value_of_the_Scaling_Parameter(s)) on each broker. |
| `spec.systemScaling.maxQueueMessages` | As for `maxConnections`. |
| `spec.systemScaling.maxSpoolUsage` | As for `maxConnections`, but here storage size might need to be increased. Follow the specific instructions of your Kubernetes or storage provider to [manually expand the volume claims](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#expanding-persistent-volumes-claims) for the PVCs used. |
| `spec.redundancy` | Changing a broker deployment from non-HA to HA or HA to non-HA is not supported by simply updating this parameter. |

>**Important**: If you are using ephemeral storage for Monitor nodes, the monitor picks up any of the above updates. However, message routing broker nodes do not—you must manually bring all broker nodes back in sync after a monitor restart.

###	Deleting a Deployment

You can delete event broker deployment by deleting the EventBroker manifest:

```sh
kubectl delete eventbroker <broker-deployment-name>
# Check what has remained from the deployment
kubectl get statefulsets,services,pods,pvc,pv
# It might take some time for broker resources to delete
```
This initiates the deletion of the event broker deployment. 

> Important: PVCs and related PVs associated with the broker's persistent storage are preserved even after deleting the EventBroker manifest because they might contain important broker configuration. They must be deleted manually if required.

###	Re-Install Broker

As described in the previous section, broker persistent storage is not automatically deleted.

In this case the deployment can be reinstalled and continue from the point before the `delete eventbroker` command was executed by [running `kubectl apply` again](#quick-start), using the same deployment name and parameters as the previous run. This includes explicitly providing the same admin password as before.

##	Operator Deployment Guide

### Install Operator

There are two recommended options to acquire the PubSub+ Event Broker Operator:
* Operator Lifecycle Manager (OLM)
* Command Line direct install

#### From Operator Lifecycle Manager

The Operator Lifecycle Manager (OLM) tool can be used to install, update, and manage the lifecycle of community operators available from [OperatorHub](https://operatorhub.io).

Follow the steps from [OperatorHub](https://operatorhub.io/operator/pubsubplus-eventbroker-operator) to setup OLM and to install the PubSub+ Event Broker Operator. Click on the Install button to see the detailed instructions.

The default namespace is `operators` for operators installed from OperatorHub.

#### From Command Line

Use the `deploy.yaml` from the [PubSub+ Event Broker Operator GitHub project](https://github.com/SolaceProducts/pubsubplus-kubernetes-quickstart). It includes a collection of manifests for all the Kubernetes resources that must be created.

The following example creates a default deployment. Edit the `deploy.yaml` before applying to customize options:
```sh
# Download manifest for possible edit
wget https://github.com/SolaceProducts/pubsubplus-kubernetes-quickstart/blob/main/deploy/deploy.yaml
# Edit manifest as required
# Manifest creates a namespace and all K8s resources for the Operator deployment
kubectl apply -f deploy.yaml
# Wait for deployment to complete
kubectl get pods -n pubsubplus-operator-system --watch
```

Customization options:
* Operator namespace: replace the default `pubsubplus-operator-system`
* Operator image: replace the default `solace/solace-pubsub-eventbroker-operator:latest`
* Allowed namespaces for broker deployment: replace default `""` value for `WATCH_NAMESPACE` env variable. Default `""` means all namespaces. Provide the name of a single namespace or a comma-separated list of namespaces.
* ImagePullSecret used when pulling the operator image from a private repo: use or replace the name `regcred`. In this case the Operator namespace must exist with the ImagePullSecret created there before applying `deploy.yaml`.

### Validating the Operator deployment

First, check if the `PubSubPlusEventBroker` Custom Resource Definition (CRD) is in place. This is a global Kubernetes resource so no namespace is required.
```sh
prompt:~$ kubectl get crd pubsubpluseventbrokers.pubsubplus.solace.com
NAME                                           CREATED AT
pubsubpluseventbrokers.pubsubplus.solace.com   2023-02-14T15:24:22Z
```

Next, check the operator deployment. The following example assumes that the operator has been deployed in the `pubsubplus-operator-system` namespace, adjust the command for a different namespace.

```
prompt:~$ kubectl get deployments -n pubsubplus-operator-system
NAME                              READY   UP-TO-DATE   AVAILABLE   AGE
pubsubplus-eventbroker-operator   1/1     1            1           3h
```

### Troubleshooting the Operator deployment

If the deployment is not ready then check whether the Operator pod is running at all:
```
kubectl get pods -n pubsubplus-operator-system
```

Get the description of the deployment and the operator pod and look for any issues:
```
kubectl describe deployment pubsubplus-eventbroker-operator -n pubsubplus-operator-system
kubectl describe pod pubsubplus-eventbroker-operator-XXX-YYY -n pubsubplus-operator-system
```

In the Operator Pod description check the `WATCH_NAMESPACE` environment variable. The default `""` value means that all namespaces are watched, otherwise the namespaces are listed where the Operator is allowed to create a broker deployment.

You should also verify that there are adequate RBAC permissions for the Operator; for details, see the [Security section](#security-considerations).

For additional hints refer to the [Broker Troubleshooting](#troubleshooting) section.

### Upgrade the Operator

A given version of the Operator has a dependency on the PubSubPlusEventBroker Custom Resource Definition (CRD) version it can interpret. The CRD can be viewed as a schema. A new version of a CRD might not be compatible with an older Operator version. Therefore it is generally recommended to use the latest possible version of the Operator. Installing a newer CRD can be expected to be backwards compatible for existing EventBroker resources, but requires an Operator upgrade to at least the same version or later.

##### Upgrading the Operator only

You can use OLM to manage installing new versions of the Operator as they become available. The default install of the PubSub+ Event Broker Operator is set to perform automatic updates. This can be changed to `Manual` by editing the broker subscription in the `operators` namespace.

If the Operator has been installed directly from the command line then update `deploy.yaml` to the new operator image tag, run `kubectl apply -f <updated-deploy.yaml>`, and then validate the updated deployment.

#### Upgrade CRD and Operator

OLM automatically manages the CRD and Operator updates.

A direct installation requires taking `deploy.yaml` from the correctly tagged version of the [PubSub+ Event Broker Operator GitHub project](https://github.com/SolaceProducts/pubsubplus-kubernetes-quickstart), because it includes the corresponding version of the CRD. 

>Note: Although the goal is to keep the CRD API versions backwards compatible, it might become necessary to introduce a new API version. In that case, detailed upgrade instructions will be provided in the Release Notes.

##	Migration from Helm-based deployments

Existing deployments that were created using the `pubsubplus` Helm chart can be ported to Operator control. In-service migration is not supported; broker shutdown is required.

Consider the following:
* The key elements holding the broker configuration and messaging data are the PVs and associated PVCs. They must be assigned to the new deployment. The broker spec allows [specifying individual PVCs for each broker](#assigning-existing-pvc-persistent-volume-claim) in a deployment.
* The Helm deployment used a single StatefulSet model with HA broker pods named `<deployment>-pubsubplus-0`, `...-1` and `...-2` for Primary, Backup and Monitor broker nodes, respectively. The new Operator deployment model creates a dedicated StatefulSet to each, with pod names `<deployment>-pubsubplus-p-0`, `...-b-0` and `...-m-0`.
* The already configured admin password must be provided to the new deployment in a secret, refer to the [Users and Passwords section](#admin-and-monitor-users-and-passwords).
* Naming the Operator-managed broker deployment the same as the Helm-based deployment helps to keep the service name (and hence the DNS name) the same, although the external IP address is expected to change if you're using LoadBalancer.
* The TLS secret for broker TLS configuration can be reused.

### Migration process

1. Using the existing Helm-based deployment:
* Take note of the `admin` user password. 
* Follow the [documentation](https://docs.solace.com/Admin/Configuring-Internal-CLI-User-Accounts.htm) to create a global read-only `monitor` user and configure a password.
* Take note of the PVCs used. Their naming scheme is `data-<deplyment-name>-pubsubplus-0`, `...-1` and `...-2` for Primary, Backup and Monitor.
2. [Create secrets](#admin-and-monitor-users-and-passwords) for the `admin` and `monitoring` users, respectively.
3. Shut down the existing deployment using `helm delete <deployment-name>`. This deletes the broker deployment but not the PV/PVCs.
4. Create a new broker spec. This example shows one for an HA deployment:
```yaml
apiVersion: pubsubplus.solace.com/v1beta1
kind: PubSubPlusEventBroker
metadata:
  name: <deployment-name> # ensure this is matching the original to keep dns configurations in sync
spec:
  redundancy: true  # "false" for non-HA
  image:
    repository: solace/solace-pubsub-standard  # ensure this is matching the original
    tag: latest                                # ensure this is matching the original
  systemScaling:
    messagingNodeCpu: "2"                      # ensure this is matching the original
    messagingNodeMemory: "3410Mi"              # ensure this is matching the original
  adminCredentialsSecret: created-admin-credetials-secret
  monitoringCredentialsSecret: created-monitoring-credetials-secret
  tls:
    enabled: true
    serverTlsConfigSecret: existing-tls-secret
  storage:
    customVolumeMount:
      - name: Primary
        persistentVolumeClaim:
          claimName: helm-primary-pvc-name
      - name: Backup
        persistentVolumeClaim:
          claimName: helm-backup-pvc-name
      - name: Monitor
        persistentVolumeClaim:
          claimName: helm-monitor-pvc-name
  # Add any other parameter from the original deployment as required
```
5. Apply the broker spec. This creates a new deployment using the specified resources:
```
kubectl apply -f new-broker-spec.yaml
```
No further steps are required for non-HA deployments. Simply wait for the deployment to come up as ready.

For HA deployments, wait for the pods to come up as running. However, they will never become ready. This is because the redundancy group addresses must be updated because the pods have new names:
| Old broker pod name | New broker pod name |
| --- | --- |
| `<deployment-name>-pubsubplus-0` | `<deployment-name>-pubsubplus-p-0` |
| `<deployment-name>-pubsubplus-1` | `<deployment-name>-pubsubplus-b-0` |
| `<deployment-name>-pubsubplus-2` | `<deployment-name>-pubsubplus-m-0` |

[Log into](#ssh-access-to-individual-event-brokers) each broker pod and follow the [Solace documentation](https://docs.solace.com/Features/HA-Redundancy/Configuring-HA-Groups.htm#Configur2) to configure the HA redundancy group `connect-via` settings.

Example for the monitor node:
```
my-pubsubplus-m-0> en
my-pubsubplus-m-0# conf
my-pubsubplus-m-0(configure)# redundancy
my-pubsubplus-m-0(configure/redundancy)# shutdown
my-pubsubplus-m-0(configure/redundancy)# show redundancy group   <== Existing config
Node Router-Name   Node Type       Address           Status
-----------------  --------------  ----------------  ---------
mypubsubplusha0    Message-Router  my-pubsubplus     Offline
                                     -0.my-pubsubpl
                                     us-discover
                                     y.default.svc
mypubsubplusha1    Message-Router  my-pubsubplus     Offline
                                     -1.my-pubsubpl
                                     us-discover
                                     y.default.svc
mypubsubplusha2*   Monitor         my-pubsubplus     Offline
                                     -2.my-pubsubpl
                                     us-discover
                                     y.default.svc

* - indicates the current node
my-pubsubplus-m-0(configure/redundancy)# group
my-pubsubplus-m-0(configure/redundancy/group)# node mypubsubplusha0
my-pubxsubplus-m-0(configure/redundancy/group/node)# connect-via my-pubsubplus-p-0.my-pubsubplus-discovery.default.svc
my-pubsubplus-m-0(configure/redundancy/group/node)# exit
my-pubsubplus-m-0(configure/redundancy/group)# node mypubsubplusha1
my-pubsubplus-m-0(configure/redundancy/group/node)# connect-via my-pubsubplus-b-0.my-pubsubplus-discovery.default.svc
my-pubsubplus-m-0(configure/redundancy/group/node)# exit
my-pubsubplus-m-0(configure/redundancy/group)# node mypubsubplusha2
my-pubsubplus-m-0(configure/redundancy/group/node)# connect-via my-pubsubplus-m-0.my-pubsubplus-discovery.default.svc
my-pubsubplus-m-0(configure/redundancy/group/node)# exit
my-pubsubplus-m-0(configure/redundancy/group)# exit
my-pubsubplus-m-0(configure/redundancy)# no shutdown
my-pubsubplus-m-0(configure/redundancy)# show redundancy
Configuration Status     : Enabled
Redundancy Status        : Down
…
my-pubsubplus-m-0(configure/redundancy)# show redundancy
Configuration Status     : Enabled
Redundancy Status        : Up
```