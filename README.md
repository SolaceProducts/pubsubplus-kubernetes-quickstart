# pubsubplus-v1alpha1
// TODO(user): Add simple overview of use/purpose

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Run locally outside the cluster

Additional Prerequisites:

* go version 1.18
* docker version 17.03+.
* kubectl

Before running locally you need to configure you local environment to pull images from the Github Container Registry. For work in progress, 
docker images are released internally and only available in the private Container Registry. This means you will need to configure your local environment to pull
images. 

1. Generate a [Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token). 
2. Ensure that you have granted SSO access to the token generated.  
3. Run the commands to set and login into Github Container Registry to download the private images.
   1. `export GHCR_PAT=YOUR_TOKEN`
   2. `echo GHCR_PAT | docker login ghcr.io -u USERNAME --password-stdin`
   
4. Clone this git repo, checkout to this branch
5. Change to the project root
```sh
cd pubsubplus-kubernetes-operator/
```
6. Create Custom Resource and start the operator
```sh
make install run
```
7. Use the sample `EventBroker` resource to create an HA cluster
```sh
kubectl apply -f config/samples/pubsubplus_v1alpha1_eventbroker.yaml
```
8. Wait for the pods to come up
```sh
kubectl get po -w --show-labels
```
9. Forward services at port 8080 and 8008 to localhost to use WebAdmin and Try-me. May also forward port 55555 for SMF messaging. Example:
```sh
kubectl port-forward svc/<service-name> 8080:8080 &
```

### Running on the cluster
1. Install Custom Resources and Operator deployment:
```sh
make deploy
```
2. Use the sample `EventBroker` resource to create an HA cluster
```sh
kubectl apply -f config/samples/pubsubplus_v1alpha1_eventbroker.yaml
```

Note: For internal docker images only available in Github Container Registry, you have to either configure your cluster to download this images.

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## Default Parameter Configuration
// TODO(user): An in-depth details of default parameters and why we use them. Including but not limited to why we use `latest` as default parameter for the images for the broker and monitoring exporter.


## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

