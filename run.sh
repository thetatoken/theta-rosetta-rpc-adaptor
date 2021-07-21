#!/bin/bash

echo "consensus:
  minProposalWait: 2" >> ../privatenet/node/config.yaml

cat ../privatenet/node/config.yaml
/go/bin/theta start --config=../privatenet/node --password="qwertyuiop" &

# cp /go/src/github.com/thetatoken/privatenet/node/key/encrypted/* ~/.thetacli/keys/encrypted/
sleep 30
/go/bin/thetacli query status
/go/bin/theta-rosetta-rpc-adaptor start