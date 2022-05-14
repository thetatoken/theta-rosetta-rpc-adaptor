#!/bin/bash

export THETA_NETWORK=${THETA_NETWORK:-testnet}
export THETA_MODE=${THETA_MODE:-online}
export THETA_PW=${THETA_PW:-qwertyuiop}

if [ $THETA_NETWORK == "mainnet" ]; then
    echo "Downloading config for ${THETA_NETWORK}"
    curl -k --output /app/mainnet/walletnode/config.yaml `curl -k 'https://mainnet-data.thetatoken.org/config?is_guardian=true'`

    echo "config:
    path: /app/mainnet/walletnode" >> /app/mainnet/walletnode/config.yaml
    echo "data:
    path: /data" >> /app/mainnet/walletnode/config.yaml
    
    MAINNET_SNAPSHOT=/app/mainnet/walletnode/snapshot

    if [ ! -f "$MAINNET_SNAPSHOT" ]; then
        echo "Downloading snapshot for ${THETA_NETWORK}"
        wget -O /app/mainnet/walletnode/snapshot `curl -k https://mainnet-data.thetatoken.org/snapshot`
    fi

    /app/theta start --config=/app/mainnet/walletnode --password=$THETA_PW &

elif [ $THETA_NETWORK == "testnet" ]; then
    mkdir -p /app/testnet/walletnode
    cp /app/integration/testnet/walletnode/config.yaml /app/testnet/walletnode/

    echo "config:
    path: /app/testnet/walletnode" >> /app/testnet/walletnode/config.yaml
    echo "data:
    path: /data" >> /app/testnet/walletnode/config.yaml

    TESTNET_SNAPSHOT=/app/testnet/walletnode/snapshot

    if [ ! -f "$TESTNET_SNAPSHOT" ]; then
        echo "downloading snapshot for ${THETA_NETWORK}"
        wget -O /app/testnet/walletnode/snapshot https://theta-testnet-backup.s3.amazonaws.com/snapshot/snapshot
    fi 

    /app/theta start --config=/app/testnet/walletnode --password=$THETA_PW &

else
    cp -r /app/integration/privatenet /app/privatenet
    mkdir ~/.thetacli
    cp -r /app/integration/privatenet/thetacli/* ~/.thetacli/
    chmod 700 ~/.thetacli/keys/encrypted

    echo "config:
    path: /app/privatenet/node" >> /app/privatenet/node/config.yaml
    echo "data:
    path: /data" >> /app/privatenet/node/config.yaml

    /app/theta start --config=/app/privatenet/node --password=$THETA_PW &
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