# Solace PubSub+ Event Broker Operator User Guide

This document provides detailed information for deploying the [Solace PubSub+ Software Event Broker](https://solace.com/products/event-broker/software/) on Kubernetes, using the Solace PubSub+ Event Broker Operator. A basic understanding of [Kubernetes concepts](https://kubernetes.io/docs/concepts/) is assumed.

The following additional set of documentation is also available:

* For a hands-on quick start, refer to the [Quick Start guide](/README.md).
* For the `PubSubPlusEventBroker` custom resource (deployment configuration) parameter options, refer to the [PubSub+ Event Broker Operator Parameters Reference]().
* For version-specific information, refer to the [Operator Release Notes]()

This guide is focused on deploying the event broker using the Operator, which is the preferred way to deploy. Note that a legacy way of [Helm-based deployment]() is also supported.

Contents:

- [Solace PubSub+ Event Broker Operator User Guide](#solace-pubsub-event-broker-operator-user-guide)
  - [The Solace PubSub+ Software Event Broker](#the-solace-pubsub-software-event-broker)
  - [Overview](#overview)
  - [Supported Kubernetes Environments](#supported-kubernetes-environments)
  - [Deployment Architecture](#deployment-architecture)
    - [Operator](#operator)
    - [Event Broker Deployment](#event-broker-deployment)
    - [Prometheus Monitoring](#prometheus-monitoring)
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
      - [Broker secrets](#broker-secrets)
      - [Broker Security Context](#broker-security-context)
      - [Using Network Policies](#using-network-policies)
  - [Exposing health and performance metrics](#exposing-health-and-performance-metrics)
  - [Deployment Guide](#deployment-guide)
    - [Deployment pre-requisites](#deployment-pre-requisites)
      - [Platform and tools](#platform-and-tools)
        - [Install the `kubectl` command-line tool](#install-the-kubectl-command-line-tool)
        - [Perform any necessary Kubernetes platform-specific setup](#perform-any-necessary-kubernetes-platform-specific-setup)
    - [Install Operator](#install-operator)
      - [OLM](#olm)
        - [Standard OperatorHub install](#standard-operatorhub-install)
        - [Manual install with options, from GitHub repo Deploy](#manual-install-with-options-from-github-repo-deploy)
      - [kubectl](#kubectl)
    - [Deploy Broker](#deploy-broker)
      - [Samples](#samples)
      - [Validating the deployment](#validating-the-deployment)
      - [How to connect, etc.](#how-to-connect-etc)
    - [Operate broker](#operate-broker)
    - [Update / upgrade broker](#update--upgrade-broker)
    - [Undeploy Broker](#undeploy-broker)
    - [Re-Install Broker](#re-install-broker)
    - [Troubleshooting](#troubleshooting)
  - [Migration from Helm-based deployment](#migration-from-helm-based-deployment)


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

##	Deployment Architecture

###	Operator

The PubSub+ Operator is following the [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). The diagram gives an overview of this mechanism:

![alt text](/docs/images/OperatorArchitecture.png "Operator overview")

* `PubSubPlusEventBroker` is registered with Kubernetes as a Custom Resource and it becomes a recognised Kubernetes object type.
* The `PubSub+ Event Broker Operator` packaged in a Pod within a Deployment must be in running state. It is configured with a set of Kubernetes namespaces to watch, which can be a list of specified ones or all.
* Creating a `PubSubPlusEventBroker` Custom Resource (CR) in a watched namespace triggers the creation of a new PubSub+ Event Broker deployment that meets the properties specified in the CR manifest (also referred to as "broker spec").
* Deviation of the deployment from the desired state or a change in the CR spec also triggers the operator to reconcile, that is to adjust the deployment towards the desired state.
* The operator runs reconcile in loops, making one adjustment at a time, until the desired state has been reached.
* Note that RBAC settings are required to permit the operator create Kubernetes objects, especially in other namespaces. Refer to the [Security]() section for further details.

The activity of the Operator can be followed from its Pod logs as described in the [troubleshooting]() section.

### Event Broker Deployment

The diagram illustrates a [Highly Available (HA)](https://docs.solace.com/Features/HA-Redundancy/SW-Broker-Redundancy-and-Fault-Tolerance.htm) PubSub+ Event Broker deployment in Kubernetes. HA deployment requires three brokers in designated roles of Primary, Backup and Monitor in an HA group.

![alt text](/docs/images/BrokerDeployment.png "HA broker deployment")

* At the core, there are the Pods running the broker containers and the associated PVC storage elements, directly managed by dedicated StatefulSets.
* Secrets are mounted on the containers feeding into the security configuration.
* There are also a set of shell scripts in a ConfigMap mounted on each broker container. They take care of configuring the broker at startup and conveying internal broker state to Kubernetes by reporting readiness and signalling which Pod is active and ready for service traffic. Active status is signalled by setting an `active=true` Pod label.
* A Service exposes the active broker Pod's services at service ports to clients.
* An additional Discovery Service enables internal access between brokers.
* Signalling active broker state requires permissions for a Pod to update its own label so this needs to be configured through RBAC settings for the deployment.

The Operator ensures that all above objects are in place with the exeception of the Pods and storage managed by the StatefulSets. This enables that even if the Operator is temporarily out of service, the broker will stay functional and resilient (noting that introducing changes will not be possible during that time) because the StatefulSets control the Pods directly.

A non-HA deployment differs from HA in that (1) there is only one StatefulSet managing one Pod that hosts the single broker, (2) there is no Discovery Service for internal communication, and (3) there is no PreShared AuthenticationKey to secure internal communication.

### Prometheus Monitoring

Support can be enabled for [Prometheus Monitoring](https://prometheus.io/docs/introduction/overview/).
time series collection happens via a pull model over HTTP
Prometheus requires an exporter running that pulls requested metrics from the monitored application - the broker in this case. 


**To be added: diagram and components description.**

## Deployment Planning

This section describes options that should be considered when planning a PubSub+ Event Broker deployment, especially for Production. 

### Deployment Topology

####	High Availability

The Operator supports deploying a single non-HA broker and also HA deployment for fault tolerance. This can be enabled by setting `spec.redundancy` to `true` in the broker deployment manifest.

#### Node Assignment

No single point of failure is important for HA deployments. Kubernetes by default tries to spread broker pods of an HA redundancy group across Availability Zones. For more deterministic deployments, specific control is enabled through the `spec.nodeAssignment` section of the broker spec for the Primary, Backup and Monitor brokers where Kubernetes standard [Affinity and NodeSelector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) definitions can be provided.

#### Enabling Pod Disruption Budget

In an HA deployment with Primary, Backup and Monitor nodes, a minimum of two nodes need to be available to reach quorum. Specifying a [Pod Disruption Budget](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) is recommended to limit situations where quorum may be lost.

This can be enabled setting the `spec.podDisruptionBudgetForHA` parameter to `true`. This will create a PodDisruptionBudget resource adapted to the broker deployment's needs, that is the number of minimum available pods set to two. Note that the parameter is ignored for a non-HA deployment.

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
[Refer to github examples]()
```yaml
  image:
    repository: solace/solace-pubsub-standard
    tag: latest
    pullPolicy: IfNotPresent
    pullSecrets:
      - pull-secret
```

#### Using a public registry

For the broker image, default values are `solace/solace-pubsub-standard/` and `latest`, which is the free PubSub+ Software Event Broker Standard Edition from the [public Solace Docker Hub repo](//hub.docker.com/r/solace/solace-pubsub-standard/). It is generally recommended to set the image tag to a specific build for traceability purposes.

Similarly, the default exporter image values are `solace/solace-pubsub-prometheus-exporter` and `latest`.

#### Using a private registry

Follow the general steps below to load an image into a private container registry (e.g.: GCR, ECR or Harbor). For specifics, consult the documentation of the registry you are using.

* Prerequisite: local installation of [Podman](https://podman.io/) or [Docker](https://www.docker.com/get-started/)
* Login to the private registry:
```sh
podman login <private-registry> ...
```
* First, load the image to the local registry:
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
Note that additional steps may be required if using signed images.

#### Pulling images from a private registry

An ImagePullSecret may be required if pulling images from a private registry, e.g.: Harbor. 

Here is an example of creating an ImagePullSecret. Refer to your registry's documentation for the specific details of use.

```sh
kubectl create secret docker-registry <pull-secret-name> --dockerserver=<private-registry-server> \
  --docker-username=<registry-user-name> --docker-password=<registry-user-password> \
  --docker-email=<registry-user-email>
```

Then add `<pull-secret-name>` to the list under the `image.pullSecrets` parameter.

### Broker Scaling

The Solace PubSub+ Event Mesh can be scaled vertically and horizontally.

Horizontal scaling is possible through [connecting multiple broker deployments](https://docs.solace.com/Features/DMR/DMR-Overview.htm#Horizontal_Scaling). This is out of scope for this document.

#### Vertical Scaling

Vertical scaling sets the maximum capacity of a given broker deployment using [system scaling parameters](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm).

Following scaling parameters can be specified:
* [Maximum Number of Client Connections](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm#max-client-connections), in `spec.systemScaling.maxConnections` parameter
* [Maximum Number of Queue Messages](https://docs.solace.com/Software-Broker/System-Scaling-Parameters.htm#max-queue-messages), in `spec.systemScaling.maxQueueMessages` parameter
* [Maximum Spool Usage](https://docs.solace.com/Messaging/Guaranteed-Msg/Message-Spooling.htm#max-spool-usage), in `spec.systemScaling.maxSpoolUsage` parameter

Additionally, for a given set of scaling, broker container CPU and Memory must be calculated  and provided in `spec.systemScaling.cpu` and `spec.systemScaling.memory` parameters. Use the [Solace online System Resource Calculator](https://docs.solace.com/Admin-Ref/Resource-Calculator/pubsubplus-resource-calculator.html) to determine CPU and memory requirements for the selected scaling parameters.

Example:
```yaml
spec:
  systemScaling:
    maxConnections: 100
    maxQueueMessages: 100
    maxSpoolUsage: 1000
    messagingNodeCpu: 2
    messagingNodeMemory: 4025Mi
```

>Note: beyond CPU and memory requirements, broker storage size (see [Storage](#storage) section) must also support the provided scaling. The calculator can be used to determine that as well.

Also note, that specifying maxConnections, maxQueueMessages and maxSpoolUsage on initial deployment will overwrite the broker’s default values. On the other hand, doing the same using upgrade on an existing deployment will not overwrite these values on brokers configuration, but it can be used to prepare (first step) for a manual scale up through CLI where these parameter changes would actually become effective (second step).

##### Minimum footprint deployment for Developers

A minimum footprint deployment option is available for development purposes but with no guaranteed performance. The minimum available resources requirements are 1 CPU, 3.4 GiB memory and 7Gi of disk storage additional to the Kubernetes environment requirements.

To activate, set `spec.developer` to `true`.

>Important: If set to `true`, `spec.developer` has precedence over any `spec.systemScaling` vertical scaling settings.

### Storage

The [PubSub+ deployment uses disk storage](https://docs.solace.com/Software-Broker/Configuring-Storage.htm) for logging, configuration, guaranteed messaging, and storing diagnostic and other information, allocated from Kubernetes volumes.

For a given set of [scaling](#vertical-scaling), use the [Solace online System Resource Calculator](https://docs.solace.com/Admin-Ref/Resource-Calculator/pubsubplus-resource-calculator.html) to determine the required storage size.

The broker pods can use following storage options:
* Dynamically allocated storage from a Kubernetes Storage Class (default)
* Static storage through a Persistent Volume Claim linked to a Persistent Volume
* Ephemeral storage

>Note: Ephemeral storage is generally not recommended. It may be acceptable for temporary deployments understanding that all configuration and messages will be lost with the loss of the broker pod.

#### Dynamically allocated storage from a Storage Class

The recommended default allocation is using Kubernetes [Dynamic Volume Provisioning](https://kubernetes.io/docs/concepts/storage/dynamic-provisioning/) utilizing [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/). 

The StatefulSet controlling a broker pod will create a Persistent Volume Claim (PVC) specifying the requested size and the Storage Class of the volume, and a Persistent Volume (PV) will be allocated from the storage class pool that meets the requirements. Both the PVC and PV names will be linked to the broker pod's name. When deleting the event broker pod(s) or even the entire deployment, the PVC and the allocated PV will not be deleted, so potentially complex configuration is preserved. They will be re-mounted and reused with the existing configuration when a new pod starts (controlled by the StatefulSet, automatically matched to the old pod even in an HA deployment) or when a deployment with the same as the old name is started. Explicitly delete a PVC if no longer needed, which will delete the corresponding PV - refer to [Deleting a Deployment]().

Example:
```yaml
spec:
  storage:
    messagingNodeStorageSize: 30Gi
    monitorNodeStorageSize: 3Gi
    # dynamic allocation
    useStorageClass: standard
```

For message processing brokers (this includes the single broker in non-HA deployment), the requested storage size is set using the `spec.storage.messagingNodeStorageSize` parameter. If not specified then the default value of `30Gi` is used. If the storage size is set to `0` then `useStorageClass` will be disregarded and pod-local ephemeral storage will be used.

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

If using NFS, or generally if allocating from a defined Kubernetes [Persistent Volume](//kubernetes.io/docs/concepts/storage/persistent-volumes/#persistent-volumes), specify a `storageClassName` in the PV manifest as in this NFS example, then set the `spec.storage.useStorageClass` parameter to the same:
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

You can to use an existing PVC with its associated PV for storage, but it must be taken into account that the deployment will try to use any existing, potentially incompatible, configuration data on that volume. The PV size must also meet the broker scaling requirements.

PVCs need to be assigned individually to the brokers in an HA deployment. Assign a PVC to the Primary in case of non-HA.
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

#### Storage solutions and providers

The PubSub+ Software Event Broker has been tested to work with Portworx, Ceph, Cinder (Openstack) and vSphere storage for Kubernetes as documented [here](https://docs.solace.com/Cloud/Deployment-Considerations/resource-requirements-k8s.htm#supported-storage-solutions).

Regarding providers, note that for [EKS](https://docs.solace.com/Cloud/Deployment-Considerations/installing-ps-cloud-k8s-eks-specific-req.htm) and [GKE](https://docs.solace.com/Cloud/Deployment-Considerations/installing-ps-cloud-k8s-gke-specific-req.htm#storage-class), `xfs` produced the best results during tests. [AKS](https://docs.solace.com/Cloud/Deployment-Considerations/installing-ps-cloud-k8s-aks-specific-req.htm) users can opt for `Local Redundant Storage (LRS)` redundancy which produced the best results when compared with other types available on Azure.

### Accessing Broker Services

Broker services (messaging, management) are available through the service ports of the [Service object](#event-broker-deployment) created as part of the deployment.

Clients may access the service ports directly through a configured [standard Kubernetes service type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types). Alternatively, services can be mapped to Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress). These options are discussed in details in the upcoming [Using Service Type](#using-a-service-type) and [Using Ingress](#using-ingress) sections.
>Note: an OpenShift-specific alternative of exposing services through Routes is described in the [PubSub+ Openshift Deployment Guide](https://github.com/SolaceProducts/pubsubplus-openshift-quickstart/blob/master/docs/PubSubPlusOpenShiftDeployment.md).

Enabling TLS for services is recommended and will also be [described](#configuring-tls-for-services).

Regardless the way to access services, the Service object is always used and it determines when and which broker Pod provides the actual service as explained in the next section.

#### Serving Pod Selection

The first criteria for a broker Pod to be selected for service is its readiness - if readiness  starts to fail Kubernetes will stop sending traffic to the pod until it passes again.

The second, additional criteria is the pod label set to `active=true`.

Both pod readiness and label are updated periodically (every 5 seconds) triggered by the pod readiness probe which invokes the `readiness_check.sh` script which is mounted on the broker container.

The requirements for a broker pod to satisfy both criteria are:
* The broker must be in Guaranteed Active service state, that is providing [Guaranteed Messaging Quality-of-Service (QoS) level of event messages persistence](https://docs.solace.com/PubSub-Basics/Guaranteed-Messages.htm). If service level is degraded even to [Direct Messages QoS](//docs.solace.com/PubSub-Basics/Direct-Messages.htm) this is no longer sufficient.
* Management service must be up at the broker container level at port 8080.
* In an HA deployment, networking must enable the broker pods to communicate with each-other at the internal ports using the Service-Discovery service.
* The Kubernetes service account associated with the deployment must have sufficient rights to patch the pod's label when the active event broker is service ready
* The broker pods must be able to communicate with the Kubernetes API at kubernetes.default.svc.cluster.local at port $KUBERNETES_SERVICE_PORT. You can find out the address and port by SSH into the pod.

In summary, a deployment is ready for service requests when there is a broker pod that is running, `1/1` ready, and the pod's label is `active=true`. An exposed service port will forward traffic to that active event broker node. Pod readiness and labels can be checked with the command:
```
kubectl get pods --show-labels
```

#### Using a Service Type

[PubSub+ services](//docs.solace.com/Configuring-and-Managing/Default-Port-Numbers.htm#Software) can be exposed through one of the following [Kubernetes service types](//kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types) by specifying the `spec.service.type` parameter:

* LoadBalancer (default) - a load balancer, typically externally accessible depending on the K8s provider.
* NodePort - maps PubSub+ services to a port on a Kubernetes node; external access depends on access to the Kubernetes node.
* ClusterIP - internal access only from within K8s.

To support [Internal load balancers](//kubernetes.io/docs/concepts/services-networking/service/#internal-load-balancer), provider-specific service annotation may be added through defining the `spec.service.annotations` parameter.

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

It is assumed that a provider out of scope of this document will be used to create a server key and certificate for the event broker, that meet the [requirements described in the Solace Documentation](https://docs.solace.com/Configuring-and-Managing/Managing-Server-Certs.htm). If the server key is password protected it shall be transformed to an unencrypted key, e.g.:  `openssl rsa -in encryedprivate.key -out unencryed.key`.

The server key and certificate must be packaged in a Kubernetes secret, for example by [creating a TLS secret](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets). Example:
```yaml
kubectl create secret tls <my-tls-secret> --key="<my-server-key-file>" --cert="<my-certificate-file>"
```

This secret name and related parameters shall be specified in the broker spec:
```
spec:
  tls:
    enabled: true
    serverTlsConfigSecret: test-tls
    certFilename:    # optional, default if not provided: tls.crt 
    certKeyFilename: # optional, default if not provided: tls.key
```

> Note: ensure filenames are matching the files reported from running `kubectl describe secret <my-tls-secret>`.

Important: it is not possible to update an existing deployment to enable TLS that has been created without TLS enabled, by a simply using the [update deployment]() procedure. In this case, for the first time, certificates need to be [manually loaded and set up](//docs.solace.com/Configuring-and-Managing/Managing-Server-Certs.htm) on each broker node. After that it is possible to use update with a secret specified.

##### Rotating the TLS certificate

In the event the server key or certificate need to be rotated the TLS Config Secret shall be updated or recreated with the new contents. Alternatively a new secret can be created and the broker spec can be updated with that secret's name.

If reusing an existing TLS secret, the new contents will be automatically mounted on the broker containers. The Operator is already watching the configured secret for any changes and will automatically initiate a rolling pod restart to take effect. Deleting the existing TLS secret will not result in immediate action but broker pods will not start if the specified TLS secret does not exist.

> Note: a pod restart will result in provisioning the server certificate from the secret again so it will revert back from any other server certificate that may have been provisioned on the broker through other mechanism.

#### Using Ingress

The `LoadBalancer` or `NodePort` service types can be used to expose all services from one PubSub+ broker (one-to-one relationship). [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress) may be used to enable efficient external access from a single external IP address to multiple PubSub+ services, potentially provided by multiple brokers.

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

This is the IP (or the IP address the FQDN resolves to) of the ingress where external clients shall target their request and any additional DNS-resolvable hostnames, used for name-based virtual host routing, must also be configured to resolve to this IP address. If using TLS then the host certificate Common Name (CN) and/or Subject Alternative Name (SAN) must be configured to match the respective FQDN.

For options to expose multiple services from potentially multiple brokers, review the [Types of Ingress from the Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#types-of-ingress).
 
The next examples provide Ingress manifests that can be applied using `kubectl apply -f <manifest-yaml>`. Then check that an external IP address (ingress controller external IP) has been assigned to the rule/service and also that the host/external IP is ready for use as it could take a some time for the address to be populated.

```
kubectl get ingress
NAME                              CLASS   HOSTS
ADDRESS         PORTS   AGE
example.address                   nginx   frontend.host
20.120.69.200   80      43m
```

##### HTTP, no TLS

The following example configures ingress to [access PubSub+ REST service](https://docs.solace.com/RESTMessagingPrtl/Solace-REST-Example.htm#cURL). Replace `<my-pubsubplus-service>` with the name of the service of your deployment (hint: the service name is similar to your pod names). The port name must match the `service.ports` name in the PubSub+ `values.yaml` file.

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

External requests shall be targeted to the ingress External-IP at the HTTP port (80) and the specified path.

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

External requests shall be targeted to the ingress External-IP through the defined hostname (here `https-example.foo.com`) at the TLS port (443) and the specified path.


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

In this case the ingress does not terminate TLS, only provides routing to the broker Pod based on the hostname provided in the SNI extension of the Client Hello at TLS connection setup. Since it will pass through TLS traffic directly to the broker as opaque data, this enables the use of ingress for any TCP-based protocol using TLS as transport.

The TLS passthrough capability must be explicitly enabled on the NGINX ingress controller, as it is off by default. This can be done by editing the `ingress-nginx-controller` "Deployment" in the `ingress-nginx` namespace.
1. Open the controller for editing: `kubectl edit deployment ingress-nginx-controller --namespace ingress-nginx`
2. Search where the `nginx-ingress-controller` arguments are provided, insert `--enable-ssl-passthrough` to the list and save. For more information refer to the [NGINX User Guide](https://kubernetes.github.io/ingress-nginx/user-guide/tls/#ssl-passthrough). Also note the potential performance impact of using SSL Passthrough mentioned here.

The Ingress manifest specifies "passthrough" by adding the `nginx.ingress.kubernetes.io/ssl-passthrough: "true"` annotation.

The deployed PubSub+ broker(s) must have TLS configured with a certificate that includes DNS names in CN and/or SAN, that match the host used. In the example the broker server certificate may specify the host `*.broker1.bar.com`, so multiple services can be exposed from `broker1`, distinguished by the host FQDN.

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
External requests shall be targeted to the ingress External-IP through the defined hostname (here `smf.broker1.bar.com`) at the TLS port (443) with no path required.

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
Additional Pod labels and annotations can be specified through `spec.podLabels` and `spec.podAnnotations`.

Additional environment variables can be passed to the broker container in the form of
* a name-value list of variables, using `spec.extraEnvVars`
* providing the name of a ConfigMap that contains env variable names and values, using `spec.extraEnvVarsCM`
* providing the name of a secret that contains env variable names and values, using `spec.extraEnvVarsSecret`

One of the primary use of environment variables is to define [configuration keys](https://docs.solace.com/Software-Broker/Configuration-Keys-Reference.htm) that are consumed and applied at the broker initial deployment. It shall be noted that configuration keys are ignored thereafter so they won't take effect even if updated later.

Finally, the timezone can be passed to the the event broker container.

### Security Considerations

The default installation of the Operator is optimized for easy deployment and getting started in a wide range of Kubernetes environments even by developers. Production use requires more tightened security. This section provides considerations for that.

#### Operator controlled namespaces

The Operator can be configured with which namespaces to watch, so it will pick up all broker specs created in the watched namespaces and create deployments there. However all other namespaces will be ignored.

Watched namespaces can be configured by providing the comma-separated list of namespaces in the `WATCH_NAMESPACE` environment variable defined in the container spec of the [Deployment](#operator) which controls the Operator pod. Assingning an empty string (default) means watching all namespaces.

It is recommended to restrict the watched namespaces for Production use. It is generally also recommended to not include the Operator's own namespace in the list because it is easier to separate RBAC settings for the operator from the broker's deployment - see next section.

#### Operator RBAC

The Operator requires CRUD permissions to manage all broker deployment resource types (e.g.: secrets) and the broker spec itself. This is defined in a ClusterRole which is bound to the Operator's service account using a ClusterRoleBinding if using the default Operator deployment. This enables the Operator to manage any of those resource types in all namespaces even if they don't belong to a broker deployment.

This needs to be restricted in a Production environment by creating a service account for the Operator in each watched namespace and use RoleBinding to bind the defined ClusterRole in each.

#### Broker deployment RBAC

A broker deployment only needs permission to update pod labels. This is defined in a Role, and a RoleBinding is created to the ServiceAccount used for the deployment. Note that without this permission the deployment will not work.

#### Operator image from private registry

The default deployment of the Operator will pull the Operator image from a public registry. If a Production deployment needs to pull the Operator image from a private registry then the [Deployment](#operator) which controls the Operator pod requires `imagePullSecrets` added for that repo:

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

#### Broker secrets

Using secrets to store passwords, TLS server keys and certificates in the broker deployment namespace follows Kubernetes recommendations.

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

Following additional settings are configurable through broker spec parameters:
```
spec:
  securityContext:
    runAsUser: 1000001
    fsGroup: 1000002
```
Above are generally the defaults if not provided. It shall be noted that the Operator will detect if the current Kubernetes environment is OpenShift and in that case, if not provided, the default `runAsUser` and `fsGroup` will be set to unspecified because otherwise they would conflict with the OpenShift "restricted" Security Context Constraint settings for a project.

#### Using Network Policies

In a controlled environment it may be necessary to configure a [NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/ ) to enable [required communication](#serving-pod-selection) between the broker nodes as well as between the broker container and the API server to set the Pod label.

##	Exposing health and performance metrics


## Deployment Guide

### Deployment pre-requisites

#### Platform and tools

##### Install the `kubectl` command-line tool

Refer to [these instructions](//kubernetes.io/docs/tasks/tools/install-kubectl/) to install `kubectl` if your environment does not already provide this tool or equivalent (like `oc` in OpenShift).

##### Perform any necessary Kubernetes platform-specific setup

This refers to getting your platform ready either by creating a new one or getting access to an existing one. Supported platforms include but are not restricted to:
* Amazon EKS
* Azure AKS
* Google GCP
* OpenShift
* MiniKube
* VMWare PKS

Check your platform running the `kubectl get nodes` command from your command-line client.

Also ensure Kubernetes CPU, Memory and Disk resources available to how the intended [scale of deployment](#broker-scaling).

###	Install Operator
####	OLM

OLM has evolved as the standard way to discover, acquire and manage Kubernetes operators. This is the also preferred way to install the PubSub+ Event Broker Operator.

Follow this [link to the Operator on OperatorHub]().

##### Standard OperatorHub install
##### Manual install with options, from GitHub repo Deploy

####	kubectl

* Install script from GitHub repo Deploy



###	Deploy Broker



####	Samples
####	Validating the deployment
Pod status
####	How to connect, etc.
how to obtain the service addresses and ports specific to your deployment
List of services
Expose services
###	Operate broker
###	Update / upgrade broker
7.6.1	Enable or disable for an existing deployment is manual only

8.4.1	Rolling vs. Manual update
8.4.2	Mechanics of picking up changes
8.4.3	AutoReconfiguration-enabled parameters
8.4.4	Maintenance mode
###	Undeploy Broker
###	Re-Install Broker
###	Troubleshooting

CRD in place
Check operator running
Check operator settings
- namespace
Check operator logs
Check pod logs


9.1	Common K8s issues
9.2	Status, Logs, Events, Conditions
9.3	Broker stuck in bad state
9.4	Using of Metrics
##	Migration from Helm-based deployment
10.1	Possible IP address change
10.a    PVC, admin password
