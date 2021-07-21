#!/bin/bash

/go/bin/theta start --config=../privatenet/node --password="qwertyuiop" &
sleep 10
/go/bin/theta-rosetta-rpc-adaptor start