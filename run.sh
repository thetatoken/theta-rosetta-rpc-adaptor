#!/bin/bash

export THETA_NETWORK=${THETA_NETWORK:-testnet}
export THETA_MODE=${THETA_MODE:-online}
export THETA_PW=${THETA_PW:-qwertyuiop}

echo $THETA_NETWORK
echo $THETA_MODE
echo $THETA_PW

# echo "consensus:
#   minProposalWait: 2" >> ../privatenet/node/config.yaml

if [ $THETA_NETWORK == "mainnet" ]
then
    /app/theta start --config=../mainnet/walletnode --password=$THETA_PW &
elif [ $THETA_NETWORK == "testnet" ]
then
    /app/theta start --config=../testnet/walletnode --password=$THETA_PW &
else
    /app/theta start --config=../privatenet/node --password=$THETA_PW &
fi

sleep 60

/app/theta-rosetta-rpc-adaptor start --mode=$THETA_MODE