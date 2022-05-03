#!/bin/bash

export THETA_NETWORK=${THETA_NETWORK:-testnet}
export THETA_MODE=${THETA_MODE:-online}
export THETA_PW=${THETA_PW:-qwertyuiop}

if [ $THETA_NETWORK == "mainnet" ]; then
    echo "Downloading config for ${THETA_NETWORK}"
    curl -k --output /go/src/github.com/thetatoken/mainnet/walletnode/config.yaml `curl -k 'https://mainnet-data.thetatoken.org/config?is_guardian=true'`
    
    MAINNET_SNAPSHOT=/go/src/github.com/thetatoken/mainnet/walletnode/snapshot

    if [ ! -f "$MAINNET_SNAPSHOT" ]; then
        echo "Downloading snapshot for ${THETA_NETWORK}"
        wget -O /go/src/github.com/thetatoken/mainnet/walletnode/snapshot `curl -k https://mainnet-data.thetatoken.org/snapshot`
    fi

    /app/theta start --config=/go/src/github.com/thetatoken/mainnet/walletnode --password=$THETA_PW &

elif [ $THETA_NETWORK == "testnet" ]; then
    mkdir -p /go/src/github.com/thetatoken/testnet/walletnode
    cp /go/src/github.com/thetatoken/theta/integration/testnet/walletnode/config.yaml /go/src/github.com/thetatoken/testnet/walletnode/

    TESTNET_SNAPSHOT=/go/src/github.com/thetatoken/testnet/walletnode/snapshot

    if [ ! -f "$TESTNET_SNAPSHOT" ]; then
        echo "downloading snapshot for ${THETA_NETWORK}"
        wget -O /go/src/github.com/thetatoken/testnet/walletnode/snapshot https://theta-testnet-backup.s3.amazonaws.com/snapshot/snapshot
    fi 

    /app/theta start --config=/go/src/github.com/thetatoken/testnet/walletnode --password=$THETA_PW &

else
    cp -r /go/src/github.com/thetatoken/theta/integration/privatenet /go/src/github.com/thetatoken/privatenet
    mkdir ~/.thetacli
    cp -r /go/src/github.com/thetatoken/theta/integration/privatenet/thetacli/* ~/.thetacli/
    chmod 700 ~/.thetacli/keys/encrypted

    /app/theta start --config=/go/src/github.com/thetatoken/privatenet/node --password=$THETA_PW &
fi

STATUS=`/app/thetacli query status`

if [[ $STATUS == Failed* ]]; then
    echo "waiting for Theta node to finish startup"
fi
    
while [[ $STATUS == Failed* ]]
do
    sleep 5
    STATUS=`/app/thetacli query status`
done

/app/theta-rosetta-rpc-adaptor start --mode=$THETA_MODE