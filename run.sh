#!/bin/bash

echo "consensus:
  minProposalWait: 2" >> ../privatenet/node/config.yaml

/go/bin/theta start --config=../privatenet/node --password="qwertyuiop" &

sleep 30

/go/bin/theta-rosetta-rpc-adaptor start