#!/bin/bash

export THETA_NETWORK=${THETA_NETWORK:-testnet}
export THETA_MODE=${THETA_MODE:-online}
export THETA_PW=${THETA_PW:-qwertyuiop}

if [ $THETA_NETWORK == "mainnet" ]; then
    echo "Downloading config for ${THETA_NETWORK}"
    apt-get -y install curl
    curl -k --output /go/src/github.com/thetatoken/mainnet/walletnode/config.yaml `curl -k 'https://mainnet-data.thetatoken.org/config?is_guardian=true'`
    
    MAINNET_SNAPSHOT=/go/src/github.com/thetatoken/mainnet/walletnode/snapshot
    if [ ! -f "$MAINNET_SNAPSHOT" ]; then
        echo "Downloading snapshot for ${THETA_NETWORK}"
        apt-get -y install wget
        wget -O /go/src/github.com/thetatoken/mainnet/walletnode/snapshot `curl -k https://mainnet-data.thetatoken.org/snapshot`
    fi

    /app/theta start --config=/go/src/github.com/thetatoken/mainnet/walletnode --password=$THETA_PW &

elif [ $THETA_NETWORK == "testnet" ]; then
    mkdir /go/src/github.com/thetatoken/testnet
    cp -r /go/src/github.com/thetatoken/theta/integration/testnet/walletnode /go/src/github.com/thetatoken/testnet

    TESTNET_SNAPSHOT=/go/src/github.com/thetatoken/theta/testnet/walletnode/snapshot
    if [ ! -f "$TESTNET_SNAPSHOT" ]; then
        echo "downloading snapshot for ${THETA_NETWORK}"
        apt-get -y install curl
        apt-get -y install wget
        # wget -O /go/src/github.com/thetatoken/theta/testnet/walletnode/snapshot `curl -k https://mainnet-data.thetatoken.org/snapshot`
    fi 

    /app/theta start --config=/go/src/github.com/thetatoken/testnet/walletnode --password=$THETA_PW &

else
    cp -r /go/src/github.com/thetatoken/theta/integration/privatenet /go/src/github.com/thetatoken/privatenet
    mkdir ~/.thetacli
    cp -r /go/src/github.com/thetatoken/theta/integration/privatenet/thetacli/* ~/.thetacli/
    chmod 700 ~/.thetacli/keys/encrypted

    /app/theta start --config=/go/src/github.com/thetatoken/privatenet/node --password=$THETA_PW &
fi

sleep 60

/app/theta-rosetta-rpc-adaptor start --mode=$THETA_MODE