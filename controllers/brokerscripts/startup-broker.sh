#!/bin/bash
APP=$(basename "$0")
IFS='-' read -ra host_array <<< $(hostname)
is_monitor=$([ ${host_array[-2]} = "m" ] && echo 1 || echo 0)
is_backup=$([ ${host_array[-2]} = "b" ] && echo 1 || echo 0)
echo "$(date) INFO: ${APP}-PubSub+ broker node starting. HA flags: HA_configured=${BROKER_REDUNDANCY}, Backup=${is_backup}, Monitor=${is_monitor}"
echo "$(date) INFO: ${APP}-Waiting for management API to become available"
password=$(cat /mnt/disks/secrets/username_admin_password)
INITIAL_STARTUP_FILE=/var/lib/solace/var/k8s_initial_startup_marker
loop_guard=120
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
  # Future improvement: enable CA configuration from secret ca.crt
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
exit 0
