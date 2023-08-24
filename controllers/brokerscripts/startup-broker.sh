#!/bin/bash
APP=$(basename "$0")
IFS='-' read -ra host_array <<< $(hostname)
is_monitor=$([ ${host_array[-2]} = "m" ] && echo 1 || echo 0)
is_backup=$([ ${host_array[-2]} = "b" ] && echo 1 || echo 0)
echo "$(date) INFO: ${APP}-PubSub+ broker node starting. HA flags: HA_configured=${BROKER_REDUNDANCY}, Backup=${is_backup}, Monitor=${is_monitor}"
echo "$(date) INFO: ${APP}-Waiting for management API to become available"
password=$(cat /mnt/disks/secrets/admin/username_admin_password)
INITIAL_STARTUP_FILE=/var/lib/solace/var/k8s_initial_startup_marker
loop_guard=60
pause=10
count=0
# Wait for Solace Management API
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
if [ "${BROKER_TLS_ENABLED}" = "true" ]; then
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
  # Function to get remote sync state
  get_router_remote_config_state() {
    # Params: $1 is property of config to return for router
    routerresults=$(/mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
              -q "<rpc><show><config-sync><database/><router/><remote/></config-sync></show></rpc>" \
              -v "/rpc-reply/rpc/show/config-sync/database/remote/tables/table[1]/source-router/${1}")
    routerremotesync_result=$(echo ${routerresults} | xmllint -xpath "string(returnInfo/valueSearchResult)" -)
    echo $routerremotesync_result
  }
  # for non-monitor nodes setup redundancy and config-sync
  if [ "${is_monitor}" = "0" ]; then
    resync_step_required=""
    role=""
    count=0
    # Determine node's primary or backup role
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
      echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, got ${role_results} for this node's primary or backup role"
      sleep ${pause}
    done
    if [ ${count} -eq ${loop_guard} ]; then
      echo "$(date) ERROR: ${APP}-Could not determine this node's primary or backup role"  >&2
      exit 1
    fi
    echo "$(date) INFO: ${APP}-Management API is up, determined that this node's role is: ${role}"
    # Determine activity (local or mate active)
    count=0
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
          echo "$(date) WARN: ${APP}-Unexpected state: this is not an initial startup of the broker and this node reports Local Active. Possibly a redeploy?"
        fi
        break
        ;;
      "Mate Active")
        echo "$(date) INFO: ${APP}-Node activity status is Mate Active, after ${run_time} seconds"
        break
        ;;
      esac
      ((count++))
      echo "$(date) INFO: ${APP}-Waited ${run_time} seconds, node activity state is: ${local_activity}"
      sleep ${pause}
    done
    if [ ${count} -eq ${loop_guard} ]; then
      echo "$(date) ERROR: ${APP}-Node activity state never become Local Active or Mate Active"  >&2
      exit 1
    fi
    # If we need to assert leader, then first wait for mate to report Standby state
    if [ "${resync_step_required}" = "true" ]; then
      # This branch is AD-active only
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
    fi # if assert-leader
    # Ensure Config-sync connection state is Connected for both primary and backup before proceeding
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
    # Now can issue assert-leader command
    if [ "${resync_step_required}" = "true" ]; then
      # This branch is AD-active only
      echo "$(date) INFO: ${APP}-Initiating assert-leader"
      /mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
        -q "<rpc><admin><config-sync><assert-leader><router/></assert-leader></config-sync></admin></rpc>"
      /mnt/disks/solace/semp_query.sh -n admin -p ${password} -u http://localhost:8080 \
        -q "<rpc><admin><config-sync><assert-leader><vpn-name>*</vpn-name></assert-leader></config-sync></admin></rpc>"
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
      # Additional checks to confirm config-sync (even if reported gloabally as not Up, it may be still up between local primary and backup in a DR setup)
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
        sleep ${pause}
        continue
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
        sleep ${pause}
        continue
      fi
      break
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