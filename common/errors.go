package common

import "github.com/coinbase/rosetta-sdk-go/types"

var (
	ErrUnableToGetChainID = &types.Error{
		Code:      1,
		Message:   "unable to get chain ID",
		Retriable: true,
	}

	ErrInvalidBlockchain = &types.Error{
		Code:      2,
		Message:   "invalid blockchain specified in network identifier",
		Retriable: false,
	}

	ErrInvalidSubnetwork = &types.Error{
		Code:      3,
		Message:   "invalid sub-network identifier",
		Retriable: false,
	}

	ErrInvalidNetwork = &types.Error{
		Code:      4,
		Message:   "invalid network specified in network identifier",
		Retriable: false,
	}

	ErrMissingNID = &types.Error{
		Code:      5,
		Message:   "network identifier is missing",
		Retriable: false,
	}

	ErrUnableToGetLatestBlk = &types.Error{
		Code:      6,
		Message:   "unable to get latest block",
		Retriable: true,
	}

	ErrUnableToGetGenesisBlk = &types.Error{
		Code:      7,
		Message:   "unable to get genesis block",
		Retriable: true,
	}

	ErrUnableToGetAccount = &types.Error{
		Code:      8,
		Message:   "unable to get account",
		Retriable: true,
	}

	ErrMissingBlockHashOrHeight = &types.Error{
		Code:      9,
		Message:   "missing block hash or height for querying block",
		Retriable: false,
	}

	ErrInvalidAccountAddress = &types.Error{
		Code:      10,
		Message:   "invalid account address",
		Retriable: false,
	}

	ErrMustSpecifySubAccount = &types.Error{
		Code:      11,
		Message:   "a valid subaccount must be specified ('general' or 'escrow')",
		Retriable: false,
	}

	ErrUnableToGetBlk = &types.Error{
		Code:      12,
		Message:   "unable to get block",
		Retriable: false,
	}

	ErrNotImplemented = &types.Error{
		Code:      13,
		Message:   "operation not implemented",
		Retriable: false,
	}

	ErrUnableToGetTxns = &types.Error{
		Code:      14,
		Message:   "unable to get transactions",
		Retriable: false,
	}

	ErrUnableToSubmitTx = &types.Error{
		Code:      15,
		Message:   "unable to submit transaction",
		Retriable: false,
	}

	ErrUnableToGetNextNonce = &types.Error{
		Code:      16,
		Message:   "unable to get next nonce",
		Retriable: true,
	}

	ErrMalformedValue = &types.Error{
		Code:      17,
		Message:   "malformed value",
		Retriable: false,
	}

	ErrUnableToGetNodeStatus = &types.Error{
		Code:      18,
		Message:   "unable to get node status",
		Retriable: true,
	}

	ErrInvalidInputParam = &types.Error{
		Code:      19,
		Message:   "Invalid input param: ",
		Retriable: false,
	}

	ErrUnsupportedPublicKeyType = &types.Error{
		Code:      20,
		Message:   "unsupported public key type",
		Retriable: false,
	}

	ErrUnableToParseTx = &types.Error{
		Code:      21,
		Message:   "unable to parse transaction",
		Retriable: false,
	}

	ErrInvalidGasPrice = &types.Error{
		Code:      22,
		Message:   "invalid gas price",
		Retriable: false,
	}

	ErrUnmarshal = &types.Error{
		Code:      23,
		Message:   "proto unmarshal error",
		Retriable: false,
	}

	ErrConstructionCheck = &types.Error{
		Code:      24,
		Message:   "operation construction check error: ",
		Retriable: true,
	}

	ErrServiceInternal = &types.Error{
		Code:      25,
		Message:   "Internal error: ",
		Retriable: false,
	}

	ErrExceededFee = &types.Error{
		Code:      26,
		Message:   "exceeded max fee",
		Retriable: true,
	}

	ErrUnableToEstimateGas = &types.Error{
		Code:      27,
		Message:   "unable to estimate gas: ",
		Retriable: true,
	}

	ErrUnableToGetSuggestGas = &types.Error{
		Code:      28,
		Message:   "unable to get suggest gas: ",
		Retriable: true,
	}

	ErrUnableToGetBlkTx = &types.Error{
		Code:      29,
		Message:   "unable to get block transaction",
		Retriable: true,
	}

	ErrUnableToGetMemPool = &types.Error{
		Code:      30,
		Message:   "unable to get mempool",
		Retriable: true,
	}

	ErrUnableToGetMemPoolTx = &types.Error{
		Code:      31,
		Message:   "unable to get mempool transaction",
		Retriable: true,
	}

	ErrUnavailableOffline = &types.Error{
		Code:    32,
		Message: "Endpoint unavailable offline",
	}

	ErrDBKeyNotFound = &types.Error{
		Code:    33,
		Message: "db key not found",
	}

	ErrorList = []*types.Error{
		ErrUnableToGetChainID,
		ErrInvalidBlockchain,
		ErrInvalidSubnetwork,
		ErrInvalidNetwork,
		ErrMissingNID,
		ErrUnableToGetLatestBlk,
		ErrUnableToGetGenesisBlk,
		ErrUnableToGetAccount,
		ErrMissingBlockHashOrHeight,
		ErrInvalidAccountAddress,
		ErrMustSpecifySubAccount,
		ErrUnableToGetBlk,
		ErrNotImplemented,
		ErrUnableToGetTxns,
		ErrUnableToSubmitTx,
		ErrUnableToGetNextNonce,
		ErrMalformedValue,
		ErrUnableToGetNodeStatus,
		ErrInvalidInputParam,
		ErrUnsupportedPublicKeyType,
		ErrUnableToParseTx,
		ErrInvalidGasPrice,
		ErrUnmarshal,
		ErrConstructionCheck,
		ErrServiceInternal,
		ErrExceededFee,
		ErrUnableToEstimateGas,
		ErrUnableToGetSuggestGas,
		ErrUnableToGetBlkTx,
		ErrUnableToGetMemPool,
		ErrUnableToGetMemPoolTx,
		ErrUnavailableOffline,
		ErrDBKeyNotFound,
	}
)
