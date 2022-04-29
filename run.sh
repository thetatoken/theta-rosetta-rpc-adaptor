#!/bin/bash

export THETA_NETWORK=${THETA_NETWORK:-testnet}
export THETA_MODE=${THETA_MODE:-online}
export THETA_PW=${THETA_PW:-qwertyuiop}

# echo "consensus:
#   minProposalWait: 2" >> ../privatenet/node/config.yaml

# /go/bin/theta start --config=../privatenet/node --password="qwertyuiop" &
/app/theta start --config=../privatenet/node --password=$THETA_PW &

sleep 30

/app/theta-rosetta-rpc-adaptor start --mode=$THETA_MODE