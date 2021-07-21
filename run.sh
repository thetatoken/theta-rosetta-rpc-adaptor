#!/bin/bash

/go/bin/theta start --config=../privatenet/node --password="qwertyuiop" &
# ls /go/src/github.com/thetatoken/privatenet/node/key/encrypted
ls ~/.thetacli/keys/encrypted/
ls /go/src/github.com/thetatoken/privatenet/node/config.yaml
# cp /go/src/github.com/thetatoken/privatenet/node/key/encrypted/* ~/.thetacli/keys/encrypted/
sleep 10
/go/bin/thetacli query status
/go/bin/theta-rosetta-rpc-adaptor start