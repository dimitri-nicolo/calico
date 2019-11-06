#!/bin/bash -e

# when set to 1, restore plane later after break plane
RESTORE=${RESTORE:=1}

# Set config variables needed for kubectl and calicoctl.
export KUBECONFIG=~/.kube/kind-config-kind

function break_plane () {
    plane=$1
    docker exec bird-a${plane} ip link set dev eth1 down
}

function restore_plane () {
    plane=$1
    docker exec bird-a${plane} ip link set dev eth1 up
}

function break_client_dev () {
    docker exec kind-worker3 ip link set dev ${client_dev} down
}

function restore_client_dev () {
    docker exec kind-worker3 ip link set dev ${client_dev} up 
}

function get_pod_ip () {
  local namespace=$1
  local key=$2    
  local value=$3    

  while ! kubectl wait pod --for=condition=Ready -l ${key}=${value} -n ${namespace} --timeout=30s  >&2 ; do
    sleep 3
  done

  pod_info=$(kubectl get pods --selector="${key}=${value}" -o json -n ${namespace} 2> /dev/null)
  echo $pod_info | jq -r '.items[] | "\(.metadata.name) \(.status.podIP)"' \
  | while read pod_name pod_ip; do
    echo ${pod_ip}
  done
}

# Setup client and server pod on different rack
function setup_client_server {
	cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  labels:
    pod-name: client
  name: client
spec:
  containers:
  - command: ["/bin/sh", "-c", "sleep 360000"]
    image: busybox
    name: client
  restartPolicy: Never
  nodeName: kind-worker

---
apiVersion: v1
kind: Pod
metadata:
  labels:
    pod-name: server
  name: server
spec:
  containers:
  - command: ["/bin/sh", "-c", "sleep 360000"]
    image: itsthenetwork/alpine-ncat 
    name: server
  restartPolicy: Never
  nodeName: kind-worker3

EOF

}

function get_dev () {
  local plane=$1
  if [ "$plane" == "1" ];then
    echo "eth0"
    return
  fi
  if [ "$plane" == "2" ];then
    echo "eth1"
    return
  fi

  echo "invalid device"
  exit 1
}  

# Get the route from client pod to server pod. 
# Extract the plane id and interface for the route.
function get_plane_dev() {
  client_plane=$(kubectl exec -t client -- traceroute -n ${server_ip} | grep "2  172" | sed -n "s/^.*172.31.1\(\S*\).1.*$/\1/p")
  server_plane=$(kubectl exec -t server -- traceroute -n ${client_ip} | grep "2  172" | sed -n "s/^.*172.31.2\(\S*\).1.*$/\1/p")

  client_dev=$(get_dev ${client_plane})
  server_dev=$(get_dev ${server_plane})
}

# The client sends packets to server. We count each second there should be
# more than 80 packets recieved by server. If not, this is counted as a link break.
# During the test, if more than 10 link break has reached, the test failed.
function test_connection() {
  # start server listening
  server_ip=$(get_pod_ip default pod-name server)
  client_ip=$(get_pod_ip default pod-name client)
  get_plane_dev
  echo "client is using plane ${client_plane}, via ${client_dev}"
  echo "server is using plane ${server_plane}, via ${server_dev}"
  kubectl exec server -- ncat -l  -k -m 10 -p 8080 > rcvd.txt &
  sleep 1

  # start client to send packets 
  local command='for i in `seq 1 6000`; do echo "$i -- dual tor test"; sleep 0.01; done |'" nc -w 1 ${server_ip} 8080"
  echo "start test connection from client to server..."
  echo "   $command"
  kubectl exec -t client -- /bin/sh -c "${command}" &

  count=0
  error=0
  previous_seq=0
  seq_string="invalid"
  while [ "${seq_string}" != "6000" ] && [ "${count}" -lt 300 ]; do
    sleep 1
    let "count +=1"
    seq_string=$(tail -n 1 rcvd.txt | cut -d " " -f1)

    seq=$seq_string
    diff=$(($seq-$previous_seq))
    echo "$count second -- seq $seq_string    packets recieved <$diff>"

    if [ "$RESTORE" -eq 1 ]; then
      #check if packets recieved is more than 80 except for first and last few iterations.
      if [ "$count" -gt 1 ] && [ "$count" -lt 60 ] && [ "$diff" -lt 80 ]; then
        let "error +=1"
        echo "Getting $diff packets since last second, link break count $error."
      fi
    else
      #count error when packets recieved is less than 80 packet except for first iteration.
      if [ "$count" -gt 1 ] && [ "$diff" -lt 80 ]; then
	failover=0      
        let "error +=1"
      fi
      #print out if packets flow again. 
      if [ "$error" -gt 0 ] && [ "$diff" -gt 80 ] && [ "$failover" -eq 0 ]; then
	failover=1      
	timeout="$((count-5))"
        echo "Getting $diff packets since link break. time taken for failover $timeout seconds"
      fi
    fi	    

    # break plane 1 after 5 seconds
    if [ "$count" -eq 5 ]; then
      break_plane ${client_plane} 
      echo "plane ${client_plane} has been brought down."
    fi

    # restore plane 1 after 25 seconds
    if [ "$RESTORE" -eq 1 ] && [ "$count" -eq 25 ]; then
      restore_plane ${client_plane}
      echo "plane ${client_plane} has been restored."
    fi

    previous_seq=$seq
  done

  # restore plane at the end of test if it has not been done earlier.
  if [ "$RESTORE" -eq 0 ]; then
    restore_plane ${client_plane}
    echo "plane ${client_plane} has been restored."
  fi

  # Test failed if more than 10 times the link break.
  if [ "$error" -gt 10 ]; then
    echo "resilience test failed."
    exit 1
  fi

  echo "resilience test passed."
}

setup_client_server
test_connection
