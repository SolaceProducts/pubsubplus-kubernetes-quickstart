#!/bin/bash
# Uninstalls the PubSub+ Helm chart to easily upgrade to the PubSub+ Operator.
#   It ensures the PVC is not uninstalled during the process
#   It migrates secrets from PubSub+ Helm chart deployment to PubSub+ PubSub+
# Params:
#   $1: the chart name
#   $2: namespace of deployment
# Assumes being run on a Kubernetes environment with enough resources for HA dev deployment
#   - kubectl configured


kubectl create secret generic broker-secret --from-literal=username_admin_password=admin
kubectl annotate pvc --all "helm.sh/resource-policy=keep" -n $2
kubectl annotate pv --all "helm.sh/resource-policy=keep" -n $2
helm uninstall $1