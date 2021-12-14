#!/bin/bash

ensure_can_reach_repo_branch(){
	REPO=$1
	BRANCH=$2
	MESSAGE="$3"
	git ls-remote --exit-code git@github.com:$REPO $BRANCH >/dev/null 2>&1;
	exit_code=$?
	if [[ $exit_code -ne 0 ]];  then 
		echo "ERROR: git@github.com:$REPO/$BRANCH is not reachable."; 
		echo " NOTE: $MESSAGE"
	fi
	exit $exit_code
}




#argument passing
"$@"