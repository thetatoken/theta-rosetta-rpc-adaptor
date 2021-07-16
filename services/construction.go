package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	ttypes "github.com/thetatoken/theta/ledger/types"

	jrpc "github.com/ybbus/jsonrpc"
)

const (
	CurveType     = "secp256k1"
	SignatureType = "ecdsa_recovery"
)

type BroadcastRawTransactionAsyncArgs struct {
	TxBytes string `json:"tx_bytes"`
}

type BroadcastRawTransactionAsyncResult struct {
	TxHash string `json:"hash"`
}

type constructionAPIService struct {
	client jrpc.RPCClient
}

// NewConstructionAPIService creates a new instance of an ConstructionAPIService.
func NewConstructionAPIService(client jrpc.RPCClient) server.ConstructionAPIServicer {
	return &constructionAPIService{
		client: client,
	}
}

// ConstructionCombine implements the /construction/combine endpoint.
func (s *constructionAPIService) ConstructionCombine(
	ctx context.Context,
	request *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	// if terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier); terr != nil {
	// 	return nil, terr
	// }

	rawTx, err := hex.DecodeString(request.UnsignedTransaction)
	if err != nil {
		terr := cmn.ErrUnableToParseTx
		terr.Message += err.Error()
		return nil, terr
	}

	tx, err := ttypes.TxFromBytes(rawTx)
	if err != nil {
		terr := cmn.ErrUnableToParseTx
		terr.Message += "invalid transaction format: " + err.Error()
		return nil, terr
	}

	signBytes := tx.SignBytes(cmn.GetChainId())

	if len(request.Signatures) != 1 {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "need exact 1 signature"
		return nil, terr
	}
	sig := &crypto.Signature{}
	sig.UnmarshalJSON(request.Signatures[0].Bytes)

	signer := common.HexToAddress(request.Signatures[0].SigningPayload.AccountIdentifier.Address)
	var in ttypes.TxInput

	switch tx.(type) {
	case *ttypes.CoinbaseTx:
		tran := *tx.(*ttypes.CoinbaseTx)
		tran.SetSignature(signer, sig)
		in = tran.Proposer
	case *ttypes.SlashTx:
		tran := *tx.(*ttypes.SlashTx)
		tran.SetSignature(signer, sig)
		in = tran.Proposer
	case *ttypes.SendTx:
		tran := *tx.(*ttypes.SendTx)
		tran.SetSignature(signer, sig)
		in = tran.Inputs[0]
	case *ttypes.ReserveFundTx:
		tran := *tx.(*ttypes.ReserveFundTx)
		tran.SetSignature(signer, sig)
		in = tran.Source
	case *ttypes.ReleaseFundTx:
		tran := *tx.(*ttypes.ReleaseFundTx)
		tran.SetSignature(signer, sig)
		in = tran.Source
	case *ttypes.ServicePaymentTx:
		tran := *tx.(*ttypes.ServicePaymentTx)
		// tran.SetSignature(signer, sig)
		in = tran.Source
	case *ttypes.SplitRuleTx:
		tran := *tx.(*ttypes.SplitRuleTx)
		tran.SetSignature(signer, sig)
		in = tran.Initiator
	case *ttypes.SmartContractTx:
		tran := *tx.(*ttypes.SmartContractTx)
		tran.SetSignature(signer, sig)
		in = tran.From
	case *ttypes.DepositStakeTx, *ttypes.DepositStakeTxV2:
		tran := *tx.(*ttypes.DepositStakeTx)
		tran.SetSignature(signer, sig)
		in = tran.Source
	case *ttypes.WithdrawStakeTx:
		tran := *tx.(*ttypes.WithdrawStakeTx)
		tran.SetSignature(signer, sig)
		in = tran.Source
	case *ttypes.StakeRewardDistributionTx:
		tran := *tx.(*ttypes.StakeRewardDistributionTx)
		tran.SetSignature(signer, sig)
		in = tran.Holder
	default:
		terr := cmn.ErrUnableToParseTx
		terr.Message += "unsupported tx type"
		return nil, terr
	}

	// Check signatures
	if !in.Signature.Verify(signBytes, signer) {
		terr := cmn.ErrInvalidInputParam
		terr.Message += fmt.Sprintf("Signature verification failed, SignBytes: %v", hex.EncodeToString(signBytes))
		return nil, terr
	}

	raw, err := ttypes.TxToBytes(tx)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "Failed to encode transaction"
		return nil, terr
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: hex.EncodeToString(raw),
	}, nil
}

// ConstructionDerive implements the /construction/derive endpoint.
func (s *constructionAPIService) ConstructionDerive(
	ctx context.Context,
	request *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	// if terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier); terr != nil {
	// 	return nil, terr
	// }

	if len(request.PublicKey.Bytes) == 0 || request.PublicKey.CurveType != CurveType {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "unsupported public key curve type"
		return nil, terr
	}

	rawPub := request.PublicKey.Bytes
	pk, err := crypto.PublicKeyFromBytes(rawPub)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "invalid public key: " + err.Error()
		return nil, terr
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: pk.Address().String(),
		},
	}, nil
}

// ConstructionHash implements the /construction/hash endpoint.
func (s *constructionAPIService) ConstructionHash(
	ctx context.Context,
	request *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	// if terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier); terr != nil {
	// 	return nil, terr
	// }

	rawTx, err := hex.DecodeString(request.SignedTransaction)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "invalid signed transaction format: " + err.Error()
		return nil, terr
	}
	tx, err := ttypes.TxFromBytes(rawTx)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "invalid signed transaction format: " + err.Error()
		return nil, terr
	}

	hash := ttypes.TxID(cmn.GetChainId(), tx)
	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: hash.String(),
		},
	}, nil
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (s *constructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	// if terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier); terr != nil {
	// 	return nil, terr
	// }

	meta := make(map[string]interface{})

	var ok bool
	var sender interface{}
	if sender, ok = request.Options["sender"]; !ok {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "empty sender address"
		return nil, terr
	}

	rpcRes, rpcErr := s.client.Call("theta.GetAccount", GetAccountArgs{
		Address: sender.(string),
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		account := GetAccountResult{}.Account
		json.Unmarshal(jsonBytes, &account)
		return account.Sequence, nil
	}

	seq, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, cmn.ErrUnableToGetAccount
	}

	meta["sequence"] = seq.(uint64)

	var txType interface{}

	if txType, ok = request.Options["type"]; !ok {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "tx type missing in metadata"
		return nil, terr
	}

	switch txType.(string) {
	case cmn.CoinbaseTx.String():
		// if blockHeight, ok := request.Options["block_height"]; ok {
		// 	meta["block_height"] = blockHeight
		// }

	case cmn.SlashTx.String():
		// if slashedAddr, ok := meta["slashed_address"]; ok {
		// 	meta["slashed_address"] = slashedAddr
		// }
		// if reserveSeq, ok := meta["reserve_sequence"]; ok {
		// 	meta["reserve_sequence"] = reserveSeq
		// }
		// if slashProof, ok := meta["slash_proof"]; ok {
		// 	meta["slash_proof"] = slashProof
		// }

	case cmn.SendTx.String():
		if fee, ok := meta["fee"]; ok {
			meta["fee"] = fee
		} else { //TODO
			// if request.feeMultiplier != nil {
			// 	terr := cmn.ErrInvalidInputParam
			// 	terr.Message += "missing fee multiplier for send tx"
			// 	return nil, terr
			// }
			// gasPrice = ttypes.GetSendTxMinimumTransactionFeeTFuelWei(uint64(*opts.feeMultiplier), height).Uint64()
		}

	case cmn.ReserveFundTx.String():
		if collateral, ok := meta["collateral"]; ok {
			meta["collateral"] = collateral
		}
		if resourceIds, ok := meta["resource_ids"]; ok {
			meta["resource_ids"] = resourceIds
		}
		if duration, ok := meta["duration"]; ok {
			meta["duration"] = duration
		}

	case cmn.ReleaseFundTx.String():
		if fee, ok := meta["fee"]; ok {
			meta["fee"] = fee
		}
		if reserveSeq, ok := meta["reserve_seq"]; ok {
			meta["reserve_seq"] = reserveSeq
		}

	case cmn.ServicePaymentTx.String():
		if fee, ok := meta["fee"]; ok {
			meta["fee"] = fee
		}
		if resourceId, ok := meta["resource_id"]; ok {
			meta["resource_id"] = resourceId
		}
		if paymentSequence, ok := meta["payment_sequence"]; ok {
			meta["payment_sequence"] = paymentSequence
		}
		if reserveSequence, ok := meta["reserve_sequence"]; ok {
			meta["reserve_sequence"] = reserveSequence
		}

	case cmn.SplitRuleTx.String():
		if fee, ok := meta["fee"]; ok {
			meta["fee"] = fee
		}
		if resourceId, ok := meta["resource_id"]; ok {
			meta["resource_id"] = resourceId
		}
		if splits, ok := meta["splits"]; ok {
			meta["splits"] = splits
		}
		if duration, ok := meta["duration"]; ok {
			meta["duration"] = duration
		}

	case cmn.SmartContractTx.String():
		var height uint64
		if gasLimit, ok := meta["gas_limit"]; ok {
			meta["gas_limit"] = gasLimit
		} else {
			status, _ := cmn.GetStatus(s.client)
			height = uint64(status.CurrentHeight)
			gasLimit = ttypes.GetMaxGasLimit(height).Uint64()
		}
		if gasPrice, ok := meta["gas_price"]; ok {
			meta["gas_price"] = gasPrice
		} else {
			if height == 0 {
				status, _ := cmn.GetStatus(s.client)
				height = uint64(status.CurrentHeight)
			}
			gasPrice = ttypes.GetMinimumGasPrice(height).Uint64()
		}
		if data, ok := meta["data"]; ok {
			meta["data"] = data
		}

	case cmn.DepositStakeTx.String():
		if fee, ok := meta["fee"]; ok {
			meta["fee"] = fee
		}
		if purpose, ok := meta["purpose"]; ok {
			meta["purpose"] = purpose
		}

	case cmn.WithdrawStakeTx.String():
		if fee, ok := meta["fee"]; ok {
			meta["fee"] = fee
		}
		if purpose, ok := meta["purpose"]; ok {
			meta["purpose"] = purpose
		}

	case cmn.StakeRewardDistributionTx.String():
		if fee, ok := meta["fee"]; ok {
			meta["fee"] = fee
		}
		if splitBasisPoint, ok := meta["split_basis_point"]; ok {
			meta["split_basis_point"] = splitBasisPoint
		}

	default:
	}

	// var gasLimit, gasPrice uint64
	// var height uint64

	// if opts.gasPrice == nil {
	// 	if height == 0 {
	// 		status, _ := cmn.GetStatus(s.client)
	// 		height = uint64(status.CurrentHeight)
	// 	}
	// 	switch opts.typ {
	// 	case cmn.SendTxInput:
	// 		if opts.feeMultiplier != nil {
	// 			terr := cmn.ErrInvalidInputParam
	// 			terr.Message += "missing fee multiplier for send tx"
	// 			return nil, terr
	// 		}
	// 		gasPrice = ttypes.GetSendTxMinimumTransactionFeeTFuelWei(uint64(*opts.feeMultiplier), height).Uint64()
	// 	case cmn.SmartContractTxFrom:
	// 		gasPrice = ttypes.GetMinimumGasPrice(height).Uint64()
	// 	default:
	// 		gasPrice = ttypes.GetMinimumTransactionFeeTFuelWei(height).Uint64()
	// 	}
	// } else {
	// 	gasPrice = *opts.gasPrice
	// }
	// meta["gasLimit"] = gasLimit
	// meta["gasPrice"] = gasPrice
	// suggestedFee := new(big.Int).Mul(
	// 	new(big.Int).SetUint64(gasPrice),
	// 	new(big.Int).SetUint64(gasLimit))

	// // check if maxFee >= fee
	// if opts.maxFee != nil {
	// 	if opts.maxFee.Cmp(suggestedFee) < 0 {
	// 		return nil, cmn.ErrExceededFee
	// 	}
	// }

	return &types.ConstructionMetadataResponse{
		Metadata: meta,
		SuggestedFee: []*types.Amount{
			&types.Amount{
				// Value: suggestedFee.String(), //TODO
				Currency: &types.Currency{
					Symbol:   ttypes.DenomTFuelWei,
					Decimals: cmn.CoinDecimals,
				},
			},
		},
	}, nil
}

// ConstructionParse implements the /construction/parse endpoint.
func (s *constructionAPIService) ConstructionParse(
	ctx context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	// if terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier); terr != nil {
	// 	return nil, terr
	// }

	rawTx, err := hex.DecodeString(request.Transaction)
	if err != nil {
		return nil, cmn.ErrUnableToParseTx
	}

	tx, err := ttypes.TxFromBytes(rawTx)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "invalid transaction format: " + err.Error()
		return nil, terr
	}

	var sender string
	var meta map[string]interface{}
	var ops []*types.Operation

	switch tx.(type) {
	case *ttypes.CoinbaseTx:
		tran := *tx.(*ttypes.CoinbaseTx)
		sender = tran.Proposer.Address.String()
		meta, ops = cmn.ParseCoinbaseTx(tran, nil, cmn.CoinbaseTx)
	case *ttypes.SlashTx:
		tran := *tx.(*ttypes.SlashTx)
		sender = tran.Proposer.Address.String()
		meta, ops = cmn.ParseSlashTx(tran, nil, cmn.SlashTx)
	case *ttypes.SendTx:
		tran := *tx.(*ttypes.SendTx)
		sender = tran.Inputs[0].Address.String()
		meta, ops = cmn.ParseSendTx(tran, nil, cmn.SendTx)
	case *ttypes.ReserveFundTx:
		tran := *tx.(*ttypes.ReserveFundTx)
		sender = tran.Source.Address.String()
		meta, ops = cmn.ParseReserveFundTx(tran, nil, cmn.ReserveFundTx)
	case *ttypes.ReleaseFundTx:
		tran := *tx.(*ttypes.ReleaseFundTx)
		sender = tran.Source.Address.String()
		meta, ops = cmn.ParseReleaseFundTx(tran, nil, cmn.ReleaseFundTx)
	case *ttypes.ServicePaymentTx:
		tran := *tx.(*ttypes.ServicePaymentTx)
		sender = tran.Source.Address.String()
		meta, ops = cmn.ParseServicePaymentTx(tran, nil, cmn.ServicePaymentTx)
	case *ttypes.SplitRuleTx:
		tran := *tx.(*ttypes.SplitRuleTx)
		sender = tran.Initiator.Address.String()
		meta, ops = cmn.ParseSplitRuleTx(tran, nil, cmn.SplitRuleTx)
	case *ttypes.SmartContractTx:
		tran := *tx.(*ttypes.SmartContractTx)
		sender = tran.From.Address.String()
		meta, ops = cmn.ParseSmartContractTx(tran, nil, cmn.SmartContractTx)
	case *ttypes.DepositStakeTx:
		tran := *tx.(*ttypes.DepositStakeTx)
		sender = tran.Source.Address.String()
		meta, ops = cmn.ParseDepositStakeTx(tran, nil, cmn.DepositStakeTx)
	case *ttypes.WithdrawStakeTx, *ttypes.DepositStakeTxV2:
		tran := *tx.(*ttypes.WithdrawStakeTx)
		sender = tran.Source.Address.String()
		meta, ops = cmn.ParseWithdrawStakeTx(tran, nil, cmn.WithdrawStakeTx)
	case *ttypes.StakeRewardDistributionTx:
		tran := *tx.(*ttypes.StakeRewardDistributionTx)
		sender = tran.Holder.Address.String()
		meta, ops = cmn.ParseStakeRewardDistributionTx(tran, nil, cmn.StakeRewardDistributionTx)
	default:
		terr := cmn.ErrUnableToParseTx
		terr.Message += "unsupported tx type"
		return nil, terr
	}

	resp := &types.ConstructionParseResponse{
		Operations: ops,
		Metadata:   meta,
	}
	if request.Signed {
		resp.AccountIdentifierSigners = []*types.AccountIdentifier{
			&types.AccountIdentifier{
				Address: sender,
			},
		}
	}
	return resp, nil
}

// ConstructionPayloads implements the /construction/payloads endpoint.
func (s *constructionAPIService) ConstructionPayloads(
	ctx context.Context,
	request *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	// if err := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier); err != nil {
	// 	return nil, err
	// }

	tx, err := cmn.AssembleTx(request.Operations, request.Metadata)
	if err != nil {
		terr := cmn.ErrServiceInternal
		terr.Message += err.Error()
		return nil, terr
	}

	raw, err := ttypes.TxToBytes(tx)
	if err != nil {
		terr := cmn.ErrServiceInternal
		terr.Message += err.Error()
		return nil, terr
	}
	unsignedTx := hex.EncodeToString(raw)

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: unsignedTx,
		Payloads: []*types.SigningPayload{
			&types.SigningPayload{
				AccountIdentifier: &types.AccountIdentifier{
					Address: request.Operations[0].Account.Address,
				},
				Bytes:         raw[:],
				SignatureType: SignatureType,
			},
		},
	}, nil
}

// ConstructionPreprocess implements the /construction/preprocess endpoint.
func (s *constructionAPIService) ConstructionPreprocess(
	ctx context.Context,
	request *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	// if err := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier); err != nil {
	// 	return nil, err
	// }

	options := make(map[string]interface{})
	options["sender"] = request.Operations[0].Account.Address

	// if request.Metadata["type"] != nil {	// tx type
	// 	switch request.Metadata["type"].(string) {
	// 	case cmn.CoinbaseTx.String():
	// 	case cmn.SlashTx.String():
	// 	case cmn.SendTx.String():
	// 	case cmn.ReserveFundTx.String():
	// 	case cmn.ReleaseFundTx.String():
	// 	case cmn.ServicePaymentTx.String():
	// 	case cmn.SplitRuleTx.String():
	// 	case cmn.SmartContractTx.String():
	// 	case cmn.DepositStakeTx.String():
	// 	case cmn.WithdrawStakeTx.String():
	// 	case cmn.StakeRewardDistributionTx.String():
	// 	default:
	// 	}
	// } else {								// tx op type
	// 	switch request.Operations[0].Type {
	// 	case cmn.CoinbaseTxProposer.String():
	// 	case cmn.SlashTxProposer.String():
	// 	case cmn.SendTxInput.String():
	// 	case cmn.ReserveFundTxSource.String():
	// 	case cmn.ReleaseFundTxSource.String():
	// 	case cmn.ServicePaymentTxSource.String():
	// 	case cmn.SplitRuleTxInitiator.String():
	// 	case cmn.SmartContractTxFrom.String():
	// 	case cmn.DepositStakeTxSource.String():
	// 	case cmn.WithdrawStakeTxSource.String():
	// 	case cmn.StakeRewardDistributionTxHolder.String():
	// 	default:
	// 	}
	// }

	if request.Metadata["type"] != nil {
		options["type"] = request.Metadata["type"]
	} else {
		switch request.Operations[0].Type {
		case cmn.CoinbaseTxProposer.String():
			options["type"] = cmn.CoinbaseTx.String()
		case cmn.SlashTxProposer.String():
			options["type"] = cmn.SlashTx.String()
		case cmn.SendTxInput.String():
			options["type"] = cmn.SendTx.String()
		case cmn.ReserveFundTxSource.String():
			options["type"] = cmn.ReserveFundTx.String()
		case cmn.ReleaseFundTxSource.String():
			options["type"] = cmn.ReleaseFundTx.String()
		case cmn.ServicePaymentTxSource.String():
			options["type"] = cmn.ServicePaymentTx.String()
		case cmn.SplitRuleTxInitiator.String():
			options["type"] = cmn.SplitRuleTx.String()
		case cmn.SmartContractTxFrom.String():
			options["type"] = cmn.SmartContractTx.String()
		case cmn.DepositStakeTxSource.String():
			options["type"] = cmn.DepositStakeTx.String()
		case cmn.WithdrawStakeTxSource.String():
			options["type"] = cmn.WithdrawStakeTx.String()
		case cmn.StakeRewardDistributionTxHolder.String():
			options["type"] = cmn.StakeRewardDistributionTx.String()
		default:
		}
	}

	// if request.Metadata["block_height"] != nil {
	// 	options["block_height"] = request.Metadata["block_height"]
	// }
	if request.Metadata["slashed_address"] != nil {
		options["slashed_address"] = request.Metadata["slashed_address"]
	}
	if request.Metadata["reserve_sequence"] != nil {
		options["reserve_sequence"] = request.Metadata["reserve_sequence"]
	}
	if request.Metadata["slash_proof"] != nil {
		options["slash_proof"] = request.Metadata["slash_proof"]
	}
	if request.Metadata["fee"] != nil {
		options["fee"] = request.Metadata["fee"]
	}
	if request.Metadata["collateral"] != nil {
		options["collateral"] = request.Metadata["collateral"]
	}
	if request.Metadata["resource_ids"] != nil {
		options["resource_ids"] = request.Metadata["resource_ids"]
	}
	if request.Metadata["duration"] != nil {
		options["duration"] = request.Metadata["duration"]
	}
	if request.Metadata["reserve_sequence"] != nil {
		options["reserve_sequence"] = request.Metadata["reserve_sequence"]
	}
	if request.Metadata["payment_sequence"] != nil {
		options["payment_sequence"] = request.Metadata["payment_sequence"]
	}
	if request.Metadata["resource_id"] != nil {
		options["resource_id"] = request.Metadata["resource_id"]
	}
	if request.Metadata["splits"] != nil {
		options["splits"] = request.Metadata["splits"]
	}
	if request.Metadata["gas_limit"] != nil {
		options["gas_limit"] = request.Metadata["gas_limit"]
	}
	if request.Metadata["gas_price"] != nil {
		options["gas_price"] = request.Metadata["gas_price"]
	}
	if request.Metadata["data"] != nil {
		options["data"] = request.Metadata["data"]
	}
	if request.Metadata["purpose"] != nil {
		options["purpose"] = request.Metadata["purpose"]
	}
	if request.Metadata["split_basis_point"] != nil {
		options["split_basis_point"] = request.Metadata["split_basis_point"]
	}

	// check and set max fee and fee multiplier
	if len(request.MaxFee) != 0 {
		maxFee := request.MaxFee[0]
		if maxFee.Currency.Symbol != ttypes.DenomTFuelWei || maxFee.Currency.Decimals != cmn.CoinDecimals {
			terr := cmn.ErrConstructionCheck
			terr.Message += "invalid fee currency"
			return nil, terr
		}
		options["max_fee"] = maxFee.Value
	}
	if request.SuggestedFeeMultiplier != nil {
		options["fee_multiplier"] = *request.SuggestedFeeMultiplier
	}

	return &types.ConstructionPreprocessResponse{
		Options: options,
	}, nil
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *constructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	// terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier)
	// if terr != nil {
	// 	return nil, terr
	// }

	rpcRes, rpcErr := s.client.Call("theta.BroadcastRawTransactionAsync", BroadcastRawTransactionAsyncArgs{
		TxBytes: request.SignedTransaction,
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		broadcastResult := BroadcastRawTransactionAsyncResult{}
		json.Unmarshal(jsonBytes, &broadcastResult)

		resp := types.TransactionIdentifierResponse{}
		resp.TransactionIdentifier = &types.TransactionIdentifier{
			Hash: broadcastResult.TxHash,
		}
		return resp, nil
	}

	res, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		terr := cmn.ErrUnableToSubmitTx
		terr.Message += err.Error()
		return nil, terr
	}

	ret, _ := res.(types.TransactionIdentifierResponse)
	return &ret, nil
}

func validateInputAdvanced(acc *ttypes.Account, signBytes []byte, in ttypes.TxInput) error {
	// Check sequence/coins
	seq, balance := acc.Sequence, acc.Balance
	if seq+1 != in.Sequence {
		return fmt.Errorf("ValidateInputAdvanced: Got %v, expected %v. (acc.seq=%v)", in.Sequence, seq+1, acc.Sequence)
	}

	// Check amount
	if !balance.IsGTE(in.Coins) {
		return fmt.Errorf("Insufficient fund: balance is %v, tried to send %v", balance, in.Coins)
	}

	// Check signatures
	if !in.Signature.Verify(signBytes, acc.Address) {
		return fmt.Errorf("Signature verification failed, SignBytes: %v", hex.EncodeToString(signBytes))
	}

	return nil
}
