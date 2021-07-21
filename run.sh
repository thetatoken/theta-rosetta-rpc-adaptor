#!/bin/bash

/go/bin/theta start --config=../privatenet/node --password="qwertyuiop" &

cd ~/.thetacli/keys/encrypted/
ls
pwd
whoami
# cp /go/src/github.com/thetatoken/privatenet/node/key/encrypted/* ~/.thetacli/keys/encrypted/
sleep 10
/go/bin/thetacli query status
/go/bin/theta-rosetta-rpc-adaptor start