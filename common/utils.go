package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	rpcc "github.com/ybbus/jsonrpc"

	"github.com/coinbase/rosetta-sdk-go/types"
	log "github.com/sirupsen/logrus"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "utils"})

var chainId string

func GetChainId() string {
	return chainId
}

func SetChainId(chid string) {
	chainId = chid
}

func GetThetaRPCEndpoint() string {
	thetaRPCEndpoint := viper.GetString(CfgThetaRPCEndpoint)
	return thetaRPCEndpoint
}

func HandleThetaRPCResponse(rpcRes *rpcc.RPCResponse, rpcErr error, parse func(jsonBytes []byte) (interface{}, error)) (result interface{}, err error) {
	if rpcErr != nil {
		return nil, fmt.Errorf("failed to get theta RPC response: %v", rpcErr)
	}
	if rpcRes.Error != nil {
		return nil, fmt.Errorf("theta RPC returns an error: %v", rpcRes.Error)
	}

	var jsonBytes []byte
	jsonBytes, err = json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}

	result, err = parse(jsonBytes)
	return
}

func ValidateNetworkIdentifier(ctx context.Context, ni *types.NetworkIdentifier) *types.Error {
	if ni != nil {
		logger.Debugf("==============chain id: %v", GetChainId())
		if !strings.EqualFold(ni.Blockchain, Theta) {
			return ErrInvalidBlockchain
		}
		if ni.SubNetworkIdentifier != nil {
			return ErrInvalidSubnetwork
		}
		if !strings.EqualFold(ni.Network, GetChainId()) {
			return ErrInvalidNetwork
		}
	} else {
		return ErrMissingNID
	}
	return nil
}
