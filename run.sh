#!/bin/bash

export THETA_NETWORK=${THETA_NETWORK:-testnet}
export THETA_MODE=${THETA_MODE:-online}
export THETA_PW=${THETA_PW:-qwertyuiop}

if [ $THETA_NETWORK == "mainnet" ]; then
    echo "Downloading config for ${THETA_NETWORK}"
    apt-get -y install curl
    curl -k --output /go/src/github.com/thetatoken/mainnet/walletnode/config.yaml `curl -k 'https://mainnet-data.thetatoken.org/config?is_guardian=true'`
    
    MAINNET_SNAPSHOT=/go/src/github.com/thetatoken/mainnet/walletnode/snapshot

    [ -f "$MAINNET_SNAPSHOT" ]
    result=$?

    if (( result != 0 )); then
        echo "Downloading snapshot for ${THETA_NETWORK}"
        apt-get -y install wget
        wget -O /go/src/github.com/thetatoken/mainnet/walletnode/snapshot `curl -k https://mainnet-data.thetatoken.org/snapshot`
    fi

    /app/theta start --config=/go/src/github.com/thetatoken/mainnet/walletnode --password=$THETA_PW &
    
    if (( result != 0 )); then
        sleep 200
    fi

elif [ $THETA_NETWORK == "testnet" ]; then
    mkdir -p /go/src/github.com/thetatoken/testnet/walletnode
    cp /go/src/github.com/thetatoken/theta/integration/testnet/walletnode/config.yaml /go/src/github.com/thetatoken/testnet/walletnode/

    TESTNET_SNAPSHOT=/go/src/github.com/thetatoken/testnet/walletnode/snapshot

    [ -f "$TESTNET_SNAPSHOT" ]
    result=$?

    if (( result != 0 )); then
        echo "downloading snapshot for ${THETA_NETWORK}"
        apt-get -y install wget
        wget -O /go/src/github.com/thetatoken/testnet/walletnode/snapshot https://theta-testnet-backup.s3.amazonaws.com/snapshot/snapshot
    fi 

    /app/theta start --config=/go/src/github.com/thetatoken/testnet/walletnode --password=$THETA_PW &

    if (( result != 0 )); then
        sleep 200
    fi 
    
else
    cp -r /go/src/github.com/thetatoken/theta/integration/privatenet /go/src/github.com/thetatoken/privatenet
    mkdir ~/.thetacli
    cp -r /go/src/github.com/thetatoken/theta/integration/privatenet/thetacli/* ~/.thetacli/
    chmod 700 ~/.thetacli/keys/encrypted

    /app/theta start --config=/go/src/github.com/thetatoken/privatenet/node --password=$THETA_PW &
fi

sleep 30

/app/theta-rosetta-rpc-adaptor start --mode=$THETA_MODE