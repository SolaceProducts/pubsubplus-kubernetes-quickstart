#!/bin/bash
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


# Function to get remote sync state
get_router_remote_config_state() {
  # Params: $1 is property of config to return for router
  routerresults=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
            -q "<rpc><show><config-sync><database/><router/><remote/></config-sync></show></rpc>" \
            -v "/rpc-reply/rpc/show/config-sync/database/remote/tables/table[1]/source-router/${1}")
  routerremotesync_result=$(echo ${routerresults} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
  echo $routerremotesync_result
}

# Main logic: note that there are no re-tries here, if check fails then return not ready.
if [ "${BROKER_REDUNDANCY}" = "true" ]; then
  # HA config
  IFS='-' read -ra host_array <<< $(hostname)
  is_monitor=$([ ${host_array[-2]} = "m" ] && echo 1 || echo 0)
  is_backup=$([ ${host_array[-2]} = "b" ] && echo 1 || echo 0)
  password=$(cat /mnt/disks/secrets/admin/username_admin_password)
  # For monitor node just check for redundancy; active label will never be set
  if [ "${is_monitor}" = "1" ]; then
    # Check redundancy
    results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
            -q "<rpc><show><redundancy/></show></rpc>" \
            -v "/rpc-reply/rpc/show/redundancy/redundancy-status")
    redundancystatus_results=$(echo ${results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
    if [ "${redundancystatus_results}" != "Up" ]; then
      echo "$(date) INFO: ${APP}-Waiting for redundancy up, redundancy state is not yet up."
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
  # From here only message routing nodes.
  # For Primary or Backup nodes set both service readiness (active label) and k8s readiness (exit return value)
  health_result=$(curl -s -o /dev/null -w "%{http_code}"  http://localhost:5550/health-check/guaranteed-active)
  case "${health_result}" in
    "200")
      if [ ! -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE} ]; then
        echo "$(date) INFO: ${APP}-HA Event Broker health check reported 200, message spool is up"
        touch ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}
        echo "$(date) INFO: ${APP}-Server status check complete for this broker node"
        echo "$(date) INFO: ${APP}-Changing pod label to active"
        #exit 1 Removing as this may delay activity switch by 5 seconds
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
      # Check config-sync status
      results=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
              -q "<rpc><show><config-sync></config-sync></show></rpc>" \
              -v "/rpc-reply/rpc/show/config-sync/status/oper-status")
      confsyncstatus_results=$(echo ${results} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
      if [ "${confsyncstatus_results}" != "Up" ]; then

        # Additional check to confirm config-sync
        echo "$(date) INFO: ${APP}-Checking Config-sync Setup. Starting additional checks to confirm config-sync locally..."

        messagevpn_result=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
              -q "<rpc><show><config-sync><database/><detail/></config-sync></show></rpc>" \
              -v "count(/rpc-reply/rpc/show/config-sync/database/local/tables/table)")
        messagevpn_total=$(echo ${messagevpn_result} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)

        # Count message_vpns in-sync and compare with total
        localmessagevpn_result=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
              -q "<rpc><show><config-sync><database/></config-sync></show></rpc>" \
              -v "count(//table[sync-state='In-Sync'])")
        local_messagevpn_total_insync=$(echo ${localmessagevpn_result} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
        if [ "$messagevpn_total" -ne "$local_messagevpn_total_insync" ]; then
          echo "$(date) INFO: ${APP}-Config-sync state is not in-sync locally."
          rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
        fi

        echo "$(date) INFO: ${APP}-Checking Config-sync Setup. Remote config-sync state check starting..."
        vpnremotehamate_result=$(get_router_remote_config_state "name")

        remote_messagevpn_result=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
              -q "<rpc><show><config-sync><database/><remote/></config-sync></show></rpc>" \
              -v "count(//table/source-router[name='$vpnremotehamate_result'])")
        remote_messagevpn_total=$(echo ${remote_messagevpn_result} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)

        #Count message_vpns in-sync, not stale and compare with total
        remotemessagevpn_result=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
              -q "<rpc><show><config-sync><database/><remote/></config-sync></show></rpc>" \
              -v "count(//table/source-router[name='$vpnremotehamate_result' and sync-state='In-Sync' and stale='No'])")
        remote_messagevpn_total_insync=$(echo ${remotemessagevpn_result} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
        if [ "$remote_messagevpn_total" -ne "$remote_messagevpn_total_insync" ]; then
          echo "$(date) INFO: ${APP}-Config-sync state is not in-sync for remote."
          rm -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE}; exit 1
        fi
      fi
      # Pass readiness check
      if [ ! -f ${FINAL_ACTIVITY_LOGGED_TRACKING_FILE} ]; then
        echo "$(date) INFO: ${APP}-Redundancy is up and node is Mate Active"
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
fi