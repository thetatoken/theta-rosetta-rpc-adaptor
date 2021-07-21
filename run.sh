#!/bin/bash

/go/bin/theta start --config=../privatenet/node --password="qwertyuiop" &
sleep 1
cp /go/src/github.com/thetatoken/privatenet/node/key/encrypted/* ~/.thetacli/keys/encrypted/
sleep 15
/go/bin/thetacli query status
/go/bin/theta-rosetta-rpc-adaptor start