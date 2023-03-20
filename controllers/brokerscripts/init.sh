#!/bin/bash
export username_admin_passwordfilepath="/mnt/disks/secrets/admin/username_admin_password"
export username_admin_globalaccesslevel=admin
export username_monitor_passwordfilepath="/mnt/disks/secrets/monitoring/username_monitor_password"
export username_monitor_globalaccesslevel=read-only
export service_ssh_port='2222'
export service_webtransport_port='8008'
export service_webtransport_tlsport='1443'
export service_semp_tlsport='1943'
export logging_debug_output=all
export system_scaling_maxconnectioncount=${BROKER_MAXCONNECTIONCOUNT}
export system_scaling_maxqueuemessagecount=${BROKER_MAXQUEUEMESSAGECOUNT}
export messagespool_maxspoolusage=${BROKER_MAXSPOOLUSAGE}
if [ "${BROKER_TLS_ENABLED}" = "true" ]; then
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
  export redundancy_authentication_presharedkey_key=$(cat /mnt/disks/secrets/presharedauthkey/preshared_auth_key)
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
fi