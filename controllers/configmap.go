/*
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
*/

package controllers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	eventbrokerv1alpha1 "github.com/SolaceProducts/pubsubplus-operator/api/v1alpha1"
)

func (r *PubSubPlusEventBrokerReconciler) configmapForEventBroker(cmName string, m *eventbrokerv1alpha1.PubSubPlusEventBroker) *corev1.ConfigMap {
	dep := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: m.Namespace,
			Labels:    getObjectLabels(m.Name),
		},
		Data: map[string]string{
			"init.sh":            InitSh,
			"startup-broker.sh":  StartupBrokerSh,
			"readiness_check.sh": ReadinessCheckSh,
			"semp_query.sh":      SempQuerySh,
		},
	}
	// Set PubSubPlusEventBroker instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// TODO: following assignment will result in unstructured YAML code
// when retreiving the resulting ConfigMap using `kubectl get cm ...`
// Consider typed "sigs.k8s.io/structured-merge-diff/v4/typed"
// 					var schemaYAML = typed.YAMLObject(`
// for YAML
// or https://github.com/kubernetes/client-go/issues/193
// or some other solution

var InitSh = `export username_admin_passwordfilepath="/mnt/disks/secrets/username_admin_password"
export username_admin_globalaccesslevel=admin
export service_ssh_port='2222'
export service_webtransport_port='8008'
export service_webtransport_tlsport='1443'
export service_semp_tlsport='1943'
export logging_debug_output=all
export system_scaling_maxconnectioncount=${BROKER_MAXCONNECTIONCOUNT}
if [ "${BROKER_TLS_ENEBLED}" = "true" ]; then
  cat /mnt/disks/certs/server/${BROKER_CERT_FILENAME} /mnt/disks/certs/server/${BROKER_CERTKEY_FILENAME} > /dev/shm/server.cert
  export tls_servercertificate_filepath="/dev/shm/server.cert"
fi
if [ "${BROKER_REDUNDANCY}" = "true" ]; then
  IFS='-' read -ra host_array <<< $(hostname)
  is_monitor=$([ ${host_array[-2]} = "m" ] && echo 1 || echo 0)
  is_backup=$([ ${host_array[-2]} = "b" ] && echo 1 || echo 0)
  namespace=$(echo $STATEFULSET_NAMESPACE)
  service=${BROKERSERVICES_NAME}
  # Deal with the fact we cannot accept "-" in broker names
  service_name=$(echo ${service} | sed 's/-//g')
  export routername=$(echo $(hostname) | sed 's/-//g')
  export redundancy_enable=yes
  export configsync_enable=yes
  export redundancy_authentication_presharedkey_key=$(cat /mnt/disks/secrets/username_admin_password | awk '{x=$0;for(i=length;i<51;i++)x=x "0";}END{print x}' | base64) # Right-pad with 0s to 50 length
  export service_redundancy_firstlistenport='8300'
  export redundancy_group_node_${service_name}p0_nodetype=message_routing
  export redundancy_group_node_${service_name}p0_connectvia=${service}-p-0.${service}-discovery.${namespace}.svc:${service_redundancy_firstlistenport}
  export redundancy_group_node_${service_name}b0_nodetype=message_routing
  export redundancy_group_node_${service_name}b0_connectvia=${service}-b-0.${service}-discovery.${namespace}.svc:${service_redundancy_firstlistenport}
  export redundancy_group_node_${service_name}m0_nodetype=monitoring
  export redundancy_group_node_${service_name}m0_connectvia=${service}-m-0.${service}-discovery.${namespace}.svc:${service_redundancy_firstlistenport}

  # Non Monitor Nodes
  if [ "${is_monitor}" = "0" ]; then
	case ${is_backup} in
	0)
	  export nodetype=message_routing
	  export redundancy_matelink_connectvia=${service}-b-0.${service}-discovery.${namespace}.svc
	  export redundancy_activestandbyrole=primary
	  ;;
	1)
	  export nodetype=message_routing
	  export redundancy_matelink_connectvia=${service}-p-0.${service}-discovery.${namespace}.svc
	  export redundancy_activestandbyrole=backup
	  ;;
	esac
  else
	export nodetype=monitoring
  fi
fi`

var StartupBrokerSh = `#!/bin/bash
APP=$(basename "$0")
IFS='-' read -ra host_array <<< $(hostname)
is_monitor=$([ ${host_array[-2]} = "m" ] && echo 1 || echo 0)
is_backup=$([ ${host_array[-2]} = "b" ] && echo 1 || echo 0)
echo "$(date) INFO: ${APP}-PubSub+ broker node starting. HA flags: HA_configured=${BROKER_REDUNDANCY}, Backup=${is_backup}, Monitor=${is_monitor}"
echo "$(date) INFO: ${APP}-Waiting for management API to become available"
password=$(cat /mnt/disks/secrets/username_admin_password)
INITIAL_STARTUP_FILE=/var/lib/solace/var/k8s_initial_startup_marker
loop_guard=60
pause=10
count=0
while [ ${count} -lt ${loop_guard} ]; do 
  if /mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 -t ; then
	break
  fi
  run_time=$((${count} * ${pause}))
  ((count++))
  echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, Management API not yet accessible"
  sleep ${pause}
done
if [ ${count} -eq ${loop_guard} ]; then
  echo "$(date) ERROR: ${APP}-Solace Management API never came up"  >&2
  exit 1 
fi
if [ "${BROKER_TLS_ENEBLED}" = "true" ]; then
  rm /dev/shm/server.cert # remove as soon as possible
  cert_results=$(curl --write-out '%{http_code}' --silent --output /dev/null -k -X PATCH -u admin:${password} https://localhost:1943/SEMP/v2/config/ \
	-H "content-type: application/json" \
	-d "{\"tlsServerCertContent\":\"$(cat /mnt/disks/certs/server/${BROKER_CERT_FILENAME} /mnt/disks/certs/server/${BROKER_CERTKEY_FILENAME} | awk '{printf "%s\\n", $0}')\"}")
  if [ "${cert_results}" != "200" ]; then
	echo "$(date) ERROR: ${APP}-Unable to set the server certificate, exiting"  >&2
	exit 1 
  fi
  echo "$(date) INFO: ${APP}-Server certificate has been configured"
fi
if [ "${BROKER_REDUNDANCY}" = "true" ]; then
  # for non-monitor nodes setup redundancy and config-sync
  if [ "${is_monitor}" = "0" ]; then
	resync_step_required=""
	role=""
	count=0
	while [ ${count} -lt ${loop_guard} ]; do 
	  role_results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc><show><redundancy><detail/></redundancy></show></rpc>" \
			-v "/rpc-reply/rpc/show/redundancy/active-standby-role[text()]")
	  run_time=$((${count} * ${pause}))
	  case "$(echo ${role_results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)" in
		"Primary")
		role="primary"
		break
		;;
		"Backup")
		role="backup"
		break
		;;
	  esac
	  ((count++))
	  echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, got ${role_results} for this node's active-standby role"
	  sleep ${pause}
	done
	if [ ${count} -eq ${loop_guard} ]; then
	  echo "$(date) ERROR: ${APP}-Could not determine this node's active-standby role"  >&2
	  exit 1 
	fi
	# Determine local activity
	count=0
	echo "$(date) INFO: ${APP}-Management API is up, determined that this node's active-standby role is: ${role}"
	while [ ${count} -lt ${loop_guard} ]; do 
	  online_results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
		  -q "<rpc><show><redundancy><detail/></redundancy></show></rpc>" \
		  -v "/rpc-reply/rpc/show/redundancy/virtual-routers/${role}/status/activity[text()]")
	  local_activity=$(echo ${online_results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
	  run_time=$((${count} * ${pause}))
	  case "${local_activity}" in
		"Local Active")
		  echo "$(date) INFO: ${APP}-Node activity status is Local Active, after ${run_time} seconds"
		  # We should only be here on new cluster create, if not this is an indication of unexpected HA procedures
		  if [[ ! -e ${INITIAL_STARTUP_FILE} ]]; then
			  # Need to issue assert master to get back into sync only one time when the PubSub+ Event Broker starts the first time
			  echo "$(date) INFO: ${APP}-Broker initial startup detected. This node will assert config-sync configuration over its mate"
			  resync_step_required="true"
		  else
			echo "$(date) WARN: ${APP}-Unexpected state: this is not an initial startup of the broker and this node reports Local Active. Normally expected nodes are Mate Active after restart"
		  fi
		  break
		  ;;
		"Mate Active")
		  echo "$(date) INFO: ${APP}-Node activity status is Mate Active, after ${run_time} seconds"
		  break
		  ;;
	  esac
	  ((count++))
	  echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, Local activity state is: ${local_activity}"
	  sleep ${pause}
	done
	if [ ${count} -eq ${loop_guard} ]; then
	  echo "$(date) ERROR: ${APP}-Local activity state never become Local Active or Mate Active"  >&2
	  exit 1 
	fi
	# If we need to assert master, then we need to wait for mate to reconcile
	if [ "${resync_step_required}" = "true" ]; then
	  count=0
	  echo "$(date) INFO: ${APP}-Waiting for mate activity state to be 'Standby'"
	  while [ ${count} -lt ${loop_guard} ]; do 
		online_results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc><show><redundancy><detail/></redundancy></show></rpc>" \
			-v "/rpc-reply/rpc/show/redundancy/virtual-routers/${role}/status/detail/priority-reported-by-mate/summary[text()]")
		mate_activity=$(echo ${online_results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
		run_time=$((${count} * ${pause}))
		case "${mate_activity}" in
		  "Standby")
			echo "$(date) INFO: ${APP}-Activity state reported by mate is Standby, after ${run_time} seconds"
			break
			;;
		esac
		((count++))
		echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, Mate activity state is: ${mate_activity}, not yet in sync"
		sleep ${pause}
	  done
	  if [ ${count} -eq ${loop_guard} ]; then
		echo "$(date) ERROR: ${APP}-Mate not in sync, never reached Standby" >&2
		exit 1 
	  fi
	fi # if assert-master
	# Ensure Config-sync connection state is Connected before proceeding
	count=0
	echo "$(date) INFO: ${APP}-Waiting for config-sync connected"
	while [ ${count} -lt ${loop_guard} ]; do
	  online_results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
		  -q "<rpc><show><config-sync></config-sync></show></rpc>" \
		  -v "/rpc-reply/rpc/show/config-sync/status/client/connection-state")
	  connection_state=$(echo ${online_results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
	  run_time=$((${count} * ${pause}))
	  case "${connection_state}" in
		"Connected")
		  echo "$(date) INFO: ${APP}-Config-sync connection state is Connected, after ${run_time} seconds"
		  break
		  ;;
	  esac
	  ((count++))
	  echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, Config-sync connection state is: ${connection_state}, not yet in Connected"
	  sleep ${pause}
	done
	if [ ${count} -eq ${loop_guard} ]; then
	  echo "$(date) ERROR: ${APP}-Config-sync connection state never reached Connected" >&2
	  exit 1
	fi
	# Now can issue assert-master command
	if [ "${resync_step_required}" = "true" ]; then
		echo "$(date) INFO: ${APP}-Initiating assert-master"
		/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc semp-version=\"soltr/9_8VMR\"><admin><config-sync><assert-master><router/></assert-master></config-sync></admin></rpc>"
		/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc semp-version=\"soltr/9_8VMR\"><admin><config-sync><assert-master><vpn-name>*</vpn-name></assert-master></config-sync></admin></rpc>"
	fi
	# Wait for config-sync results
	count=0
	echo "$(date) INFO: ${APP}-Waiting for config-sync results"
	while [ ${count} -lt ${loop_guard} ]; do
	  online_results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			  -q "<rpc><show><config-sync></config-sync></show></rpc>" \
			  -v "/rpc-reply/rpc/show/config-sync/status/oper-status")
	  confsyncstatus_results=$(echo ${online_results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
	  run_time=$((${count} * ${pause}))
	  case "${confsyncstatus_results}" in
		"Up")
		  echo "$(date) INFO: ${APP}-Config-sync is Up, after ${run_time} seconds"
		  break
		  ;;
	  esac
	  ((count++))
	  echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, Config-sync is: ${confsyncstatus_results}, not yet Up"
	  sleep ${pause}
	done
	if [ ${count} -eq ${loop_guard} ]; then
	  echo "$(date) ERROR: ${APP}-Config-sync never reached state \"Up\"" >&2
	  exit 1
	fi
  fi # if not monitor
fi
echo "$(date) INFO: ${APP}-PubSub+ Event Broker bringup is complete for this node."
# create startup file after PubSub+ Event Broker is up and running.  Create only if it does not exist
if [[ ! -e ${INITIAL_STARTUP_FILE} ]]; then
	echo "PubSub+ Event Broker initial startup completed on $(date)" > ${INITIAL_STARTUP_FILE}
fi
exit 0`

var ReadinessCheckSh = `#!/bin/bash
APP=$(basename "$0")
LOG_FILE=/usr/sw/var/k8s_readiness_check.log # STDOUT/STDERR goes to k8s event logs but gets cleaned out eventually. This will also persist it.
if [ -f ${LOG_FILE} ] ; then
	tail -n 1000 ${LOG_FILE} > ${LOG_FILE}.tmp; mv -f ${LOG_FILE}.tmp ${LOG_FILE} || :  # Limit logs size
fi
exec > >(tee -a ${LOG_FILE}) 2>&1 # Setup logging
FINAL_ACTIVITY_LOGGED_TRACKING_FILE=/tmp/final_activity_state_logged

# Function to read Kubernetes metadata labels
get_label () {
  # Params: $1 label name
  echo $(cat /etc/podinfo/labels | awk -F= '$1=="'${1}'"{print $2}' | xargs);
}

# Function to set Kubernetes metadata labels
set_label () {
  # Params: $1 label name, $2 label set value
  #Prevent overdriving Kubernetes infra, don't set activity state to same as previous state
  previous_state=$(get_label "active")
  if [ "${2}" = "${previous_state}" ]; then
	#echo "$(date) INFO: ${APP}-Current and Previous state match (${2}), not updating pod label"
	:
  else
	echo "$(date) INFO: ${APP}-Updating pod label using K8s API from ${previous_state} to ${2}"
	echo "[{\"op\": \"add\", \"path\": \"/metadata/labels/${1}\", \"value\": \"${2}\" }]" > /tmp/patch_label.json
	K8S=https://kubernetes.default.svc.cluster.local:$KUBERNETES_SERVICE_PORT
	KUBE_TOKEN=$(</var/run/secrets/kubernetes.io/serviceaccount/token)
	CACERT=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
	NAMESPACE=$(</var/run/secrets/kubernetes.io/serviceaccount/namespace)
	if ! curl -sS --output /dev/null --cacert $CACERT --connect-timeout 5 \
		--request PATCH --data "$(cat /tmp/patch_label.json)" \
		-H "Authorization: Bearer $KUBE_TOKEN" -H "Content-Type:application/json-patch+json" \
		$K8S/api/v1/namespaces/$NAMESPACE/pods/$HOSTNAME ; then
	  # Label update didn't work this way, fall back to alternative legacy method to update label
	  if ! curl -sSk --output /dev/null -H "Authorization: Bearer $KUBE_TOKEN" --request PATCH --data "$(cat /tmp/patch_label.json)" \
		-H "Content-Type:application/json-patch+json" \
		https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_PORT_443_TCP_PORT/api/v1/namespaces/$STATEFULSET_NAMESPACE/pods/$HOSTNAME ; then
		echo "$(date) ERROR: ${APP}-Unable to update pod label, check access from pod to K8s API or RBAC authorization" >&2
		rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
	  fi
	fi
  fi
}

# Main logic: note that there are no re-tries here, if check fails then return not ready.
if [ "${BROKER_REDUNDANCY}" = "true" ]; then
  # HA config
  IFS='-' read -ra host_array <<< $(hostname)
  is_monitor=$([ ${host_array[-2]} = "m" ] && echo 1 || echo 0)
  is_backup=$([ ${host_array[-2]} = "b" ] && echo 1 || echo 0)
  password=$(cat /mnt/disks/secrets/username_admin_password)
  # For update (includes SolOS upgrade) purposes, additional checks are required for readiness state when the pod has been started
  # This is an update if the LASTVERSION_FILE with K8s controller-revision-hash exists and contents differ from current value
  LASTVERSION_FILE=/var/lib/solace/var/lastConfigRevisionBeforeReboot
  if [ ! -f ${LASTVERSION_FILE} ] || [[ $(cat ${LASTVERSION_FILE}) != $(get_label "controller-revision-hash") ]] ; then
	echo "$(date) INFO: ${APP}-Initial startup or Upgrade detected, running additional checks..."
	# Check redundancy
	echo "$(date) INFO: ${APP}-Running checks. Redundancy state check started..."
	results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc><show><redundancy/></show></rpc>" \
			-v "/rpc-reply/rpc/show/redundancy/redundancy-status")
	redundancystatus_results=$(echo ${results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
	if [ "${redundancystatus_results}" != "Up" ]; then
	  echo "$(date) INFO: ${APP}-Redundancy state is not yet up."
	  rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
	fi
	# Additionally check config-sync status for non-monitoring nodes
	echo "$(date) INFO: ${APP}-Running checks. Config-sync state check started..."
	if [ "${is_monitor}" = "0" ]; then
	  results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			  -q "<rpc><show><config-sync></config-sync></show></rpc>" \
			  -v "/rpc-reply/rpc/show/config-sync/status/oper-status")
	  confsyncstatus_results=$(echo ${results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
	  if [ "${confsyncstatus_results}" != "Up" ]; then
		echo "$(date) INFO: ${APP}-Config-sync state is not yet up."
		rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
	  fi
	fi
  fi
  # Record current version in LASTVERSION_FILE
  echo $(get_label "controller-revision-hash") > ${LASTVERSION_FILE}
  # For monitor node just check for redundancy; active label will never be set
  if [ "${is_monitor}" = "1" ]; then
	# Check redundancy
	echo "$(date) INFO: ${APP}-Running checks. Redundancy state check started..."
	results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc><show><redundancy/></show></rpc>" \
			-v "/rpc-reply/rpc/show/redundancy/redundancy-status")
	redundancystatus_results=$(echo ${results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
	if [ "${redundancystatus_results}" != "Up" ]; then
		echo "$(date) INFO: ${APP}-Redundancy state is not yet up."
		rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
	fi
	if [ ! -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE} ]; then
		echo "$(date) INFO: ${APP}-All nodes online, monitor node is redundancy ready"
		touch ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}
		echo "$(date) INFO: ${APP}-Server status check complete for this broker node"
		exit 1
	fi
	exit 0
  fi # End Monitor Node
  # For Primary or Backup nodes set both service readiness (active label) and k8s readiness (exit return value)
  health_result=$(curl -s -o /dev/null -w "%{http_code}"  http://localhost:5550/health-check/guaranteed-active)
  case "${health_result}" in
	"200")
	  if [ ! -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE} ]; then
		echo "$(date) INFO: ${APP}-HA Event Broker health check reported 200, message spool is up"
		touch ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}
		echo "$(date) INFO: ${APP}-Server status check complete for this broker node"
		echo "$(date) INFO: ${APP}-Changing pod label to active"
		#exit 1
	  fi
	  set_label "active" "true"
	  exit 0
	  ;;
	"503")
	  if [[ $(get_label "active") = "true" ]]; then echo "$(date) INFO: ${APP}-HA Event Broker health check reported 503"; fi
	  set_label "active" "false"
	  # Further check is required to determine readiness
	  ;;
	*)
	  echo "$(date) WARN: ${APP}-HA Event Broker health check reported unexpected ${health_result}"
	  set_label "active" "false"
	  echo "$(date) INFO: ${APP}-Changing pod label to inactive"
	  rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
  esac
  # At this point analyzing readiness after health check returned 503 - checking if Event Broker is Standby
  case "${is_backup}" in
	"0")
	  config_role="primary"
	  ;;
	"1")
	  config_role="backup"
	  ;;
  esac
  online_results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
		  -q "<rpc><show><redundancy><detail/></redundancy></show></rpc>" \
		  -v "/rpc-reply/rpc/show/redundancy/virtual-routers/${config_role}/status/activity[text()]")
  local_activity=$(echo ${online_results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
  case "${local_activity}" in
	"Mate Active")
	  # Check redundancy
	  results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc><show><redundancy/></show></rpc>" \
			-v "/rpc-reply/rpc/show/redundancy/redundancy-status")
	  redundancystatus_results=$(echo ${results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
	  if [ "${redundancystatus_results}" != "Up" ]; then
		echo "$(date) INFO: ${APP}-Running checks.Redundancy state is not yet up."
		rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
	  fi
	  # Additionally check config-sync status for non-monitoring nodes
	  if [ "${node_ordinal}" != "2" ]; then
		results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
			-q "<rpc><show><config-sync></config-sync></show></rpc>" \
			-v "/rpc-reply/rpc/show/config-sync/status/oper-status")
		confsyncstatus_results=$(echo ${results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
		if [ "${confsyncstatus_results}" != "Up" ]; then
			echo "$(date) INFO: ${APP}-Running checks. Config-sync state is not yet up."
			rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
		fi
	  fi
	  # Pass readiness check
	  if [ ! -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE} ]; then
		echo "$(date) INFO: ${APP}-Redundancy is up and node is mate Active"
		touch ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}
		echo "$(date) INFO: ${APP}-Server status check complete for this broker node"
		exit 1
	  fi
	  exit 0
	  ;;
	*)
	  echo "$(date) WARN: ${APP}-Health check returned 503 and local activity state is: ${local_activity}, failing readiness check."
	  rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
	  ;;
  esac
else
  # nonHA config
  health_result=$(curl -s -o /dev/null -w "%{http_code}"  http://localhost:5550/health-check/guaranteed-active)
  case "${health_result}" in
	"200")
	  if [ ! -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE} ]; then
		echo "$(date) INFO: ${APP}-nonHA Event Broker health check reported 200, message spool is up"
		touch ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}
		echo "$(date) INFO: ${APP}-Server status check complete for this broker node"
		echo "$(date) INFO: ${APP}-Changing pod label to active"
		exit 1
	  fi
	  set_label "active" "true"
	  exit 0
	  ;;
	"503")
	  if [[ $(get_label "active") = "true" ]]; then echo "$(date) INFO: ${APP}-nonHA Event Broker health check reported 503, message spool is down"; fi
	  set_label "active" "false"
	  echo "$(date) INFO: ${APP}-Changing pod label to inactive"
	  # Fail readiness check
	  rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
	  ;;
	*)
	  echo "$(date) WARN: ${APP}-nonHA Event Broker health check reported ${health_result}"
	  set_label "active" "false"
	  echo "$(date) INFO: ${APP}-Changing pod label to inactive"
	  # Fail readiness check
	  rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
  esac
fi`

var SempQuerySh = `#!/bin/bash
APP=$(basename "$0")
OPTIND=1         # Reset in case getopts has been used previously in the shell.
# Initialize our own variables:
count_search=""
name=""
password=""
query=""
url=""
value_search=""
test_connection_only=false
script_name=$0
verbose=0
while getopts "c:n:p:q:u:v:t" opt; do
	case "$opt" in
	c)  count_search=$OPTARG
		;;
	n)  username=$OPTARG
		;;
	p)  password=$OPTARG
		;;
	q)  query=$OPTARG
		;;
	u)  url=$OPTARG
		;;
	v)  value_search=$OPTARG
		;;
	t)  test_connection_only=true
		;;
	esac
done
shift $((OPTIND-1))
[ "$1" = "--" ] && shift
verbose=1
#echo "$(date) INFO: ${APP}-${script_name}: count_search=${count_search} ,username=${username} ,password=xxx query=${query} \
#            ,url=${url} ,value_search=${value_search} ,Leftovers: $@" >&2
if [[ ${url} = "" || ${username} = "" || ${password} = "" ]]; then
  echo "$(date) ERROR: ${APP}-${script_name}: url, username, password are madatory fields" >&2
  echo  '<returnInfo><errorInfo>missing parameter</errorInfo></returnInfo>'
  exit 1
fi
if [ "$(curl --write-out '%{http_code}' --silent --output /dev/null -u ${username}:${password} ${url}/SEMP)" != "200" ] ; then
  echo  "<returnInfo><errorInfo>management host is not responding</errorInfo></returnInfo>"
  exit 1
fi
if [ "$test_connection_only" = true ] ; then
  exit 0      # done here, connection is up
fi
query_response=$(curl -sS -u ${username}:${password} ${url}/SEMP -d "${query}")
# Validate first char of response is "<", otherwise no hope of being valid xml
if [[ ${query_response:0:1} != "<" ]] ; then 
  echo  "<returnInfo><errorInfo>no valid xml returned</errorInfo></returnInfo>"
  exit 1
fi
query_response_code=$(echo $query_response | xmllint -xpath 'string(/rpc-reply/execute-result/@code)' -)

if [[ -z ${query_response_code} && ${query_response_code} != "ok" ]]; then
	echo  "<returnInfo><errorInfo>query failed -${query_response_code}-</errorInfo></returnInfo>"
	exit 1
fi
#echo "$(date) INFO: ${APP}-${script_name}: query passed ${query_response_code}" >&2
if [[ ! -z $value_search ]]; then
	value_result=$(echo $query_response | xmllint -xpath "string($value_search)" -)
	echo  "<returnInfo><errorInfo></errorInfo><valueSearchResult>${value_result}</valueSearchResult></returnInfo>"
	exit 0
fi
if [[ ! -z $count_search ]]; then
	count_line=$(echo $query_response | xmllint -xpath "$count_search" -)
	count_string=$(echo $count_search | cut -d '"' -f 2)
	count_result=$(echo ${count_line} | tr "><" "\n" | grep -c ${count_string})
	echo  "<returnInfo><errorInfo></errorInfo><countSearchResult>${count_result}</countSearchResult></returnInfo>"
	exit 0
fi`
