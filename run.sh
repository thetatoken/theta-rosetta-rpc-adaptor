#!/bin/bash

/go/bin/theta start --config=../privatenet/node --password="qwertyuiop"
sleep 15
/go/bin/theta-rosetta-rpc-adaptor start