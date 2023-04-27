#!/bin/bash
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
fi