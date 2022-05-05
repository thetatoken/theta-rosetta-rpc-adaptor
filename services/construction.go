package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/spf13/viper"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/crypto/secp256k1"
	"github.com/thetatoken/theta/crypto/sha3"
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

// ConstructionDerive implements the /construction/derive endpoint.
func (s *constructionAPIService) ConstructionDerive(
	ctx context.Context,
	request *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	if terr := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); terr != nil {
		return nil, terr
	}

	if len(request.PublicKey.Bytes) == 0 || request.PublicKey.CurveType != CurveType {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "unsupported public key curve type"
		return nil, terr
	}

	pubkey, err := decompressPubkey(request.PublicKey.Bytes)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "Unable to decompress public key: " + err.Error()
		return nil, terr
	}

	addr := pubkeyToAddress(*pubkey)
	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: addr.Hex(),
		},
	}, nil
}

// ConstructionPreprocess implements the /construction/preprocess endpoint.
func (s *constructionAPIService) ConstructionPreprocess(
	ctx context.Context,
	request *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	options := make(map[string]interface{})
	options["sender"] = request.Operations[0].Account.Address

	if request.Metadata["type"] != nil {
		options["type"] = request.Metadata["type"]
	} else {
		switch request.Operations[0].Type {
		case cmn.CoinbaseTxProposer.String():
			options["type"] = cmn.CoinbaseTx
		case cmn.SlashTxProposer.String():
			options["type"] = cmn.SlashTx
		case cmn.SendTxInput.String():
			options["type"] = cmn.SendTx
			options["fee_multiplier"] = len(request.Operations) //TODO: what if len(inputs) != len(outputs) ?
		case cmn.ReserveFundTxSource.String():
			options["type"] = cmn.ReserveFundTx
		case cmn.ReleaseFundTxSource.String():
			options["type"] = cmn.ReleaseFundTx
		case cmn.ServicePaymentTxSource.String():
			options["type"] = cmn.ServicePaymentTx
		case cmn.SplitRuleTxInitiator.String():
			options["type"] = cmn.SplitRuleTx
		case cmn.SmartContractTxFrom.String():
			options["type"] = cmn.SmartContractTx
		case cmn.DepositStakeTxSource.String():
			options["type"] = cmn.DepositStakeTx
		case cmn.WithdrawStakeTxSource.String():
			options["type"] = cmn.WithdrawStakeTx
		case cmn.StakeRewardDistributionTxHolder.String():
			options["type"] = cmn.StakeRewardDistributionTx
		default:
			terr := cmn.ErrUnableToParseTx
			terr.Message += "unsupported tx type"
			return nil, terr
		}
	}

	// if request.Metadata["block_height"] != nil {
	// 	options["block_height"] = request.Metadata["block_height"]
	// }
	// if request.Metadata["slashed_address"] != nil {
	// 	options["slashed_address"] = request.Metadata["slashed_address"]
	// }
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

	return &types.ConstructionPreprocessResponse{
		Options: options,
	}, nil
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (s *constructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	if !strings.EqualFold(cmn.CfgRosettaModeOnline, viper.GetString(cmn.CfgRosettaMode)) {
		return nil, cmn.ErrUnavailableOffline
	}

	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

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
		err := json.Unmarshal(jsonBytes, &account)
		if err != nil {
			return nil, err
		}
		return account.Sequence, nil
	}

	seq, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, cmn.ErrUnableToGetAccount
	}

	meta["sequence"] = seq.(uint64) + 1

	var txType interface{}

	if txType, ok = request.Options["type"]; !ok {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "tx type missing in metadata"
		return nil, terr
	}

	meta["type"] = txType

	var status *cmn.GetStatusResult
	var suggestedFee *big.Int

	if fee, ok := meta["fee"]; ok {
		suggestedFee = new(big.Int)
		suggestedFee.SetUint64(fee.(uint64))
	} else {
		status, err = cmn.GetStatus(s.client)
		if err != nil {
			terr := cmn.ErrInvalidInputParam
			terr.Message += "can't get blockchain status"
			return nil, terr
		}
		height := uint64(status.CurrentHeight)

		switch cmn.TxType(txType.(float64)) {
		case cmn.SendTx:
			if feeMultiplier, ok := request.Options["fee_multiplier"]; ok {
				suggestedFee = ttypes.GetSendTxMinimumTransactionFeeTFuelWei(uint64(feeMultiplier.(float64)), height)
			} else {
				terr := cmn.ErrInvalidInputParam
				terr.Message += "missing fee multiplier for send tx"
				return nil, terr
			}
		case cmn.SmartContractTx:
			suggestedFee = ttypes.GetMinimumGasPrice(height)
		default:
			suggestedFee = ttypes.GetMinimumTransactionFeeTFuelWei(height)
		}
	}
	meta["fee"] = suggestedFee

	switch cmn.TxType(txType.(float64)) {
	case cmn.CoinbaseTx:
		// if blockHeight, ok := request.Options["block_height"]; ok {
		// 	meta["block_height"] = blockHeight
		// }

	case cmn.SlashTx:
		// if slashedAddr, ok := request.Options["slashed_address"]; ok {
		// 	meta["slashed_address"] = slashedAddr
		// }
		// if reserveSeq, ok := request.Options["reserve_sequence"]; ok {
		// 	meta["reserve_sequence"] = reserveSeq
		// }
		// if slashProof, ok := request.Options["slash_proof"]; ok {
		// 	meta["slash_proof"] = slashProof
		// }

	case cmn.SendTx:

	case cmn.ReserveFundTx:
		if collateral, ok := request.Options["collateral"]; ok {
			meta["collateral"] = collateral
		}
		if resourceIds, ok := request.Options["resource_ids"]; ok {
			meta["resource_ids"] = resourceIds
		}
		if duration, ok := request.Options["duration"]; ok {
			meta["duration"] = duration
		}

	case cmn.ReleaseFundTx:
		if reserveSeq, ok := request.Options["reserve_seq"]; ok {
			meta["reserve_seq"] = reserveSeq
		}

	case cmn.ServicePaymentTx:
		if resourceId, ok := request.Options["resource_id"]; ok {
			meta["resource_id"] = resourceId
		}
		if paymentSequence, ok := request.Options["payment_sequence"]; ok {
			meta["payment_sequence"] = paymentSequence
		}
		if reserveSequence, ok := request.Options["reserve_sequence"]; ok {
			meta["reserve_sequence"] = reserveSequence
		}

	case cmn.SplitRuleTx:
		if resourceId, ok := request.Options["resource_id"]; ok {
			meta["resource_id"] = resourceId
		}
		if splits, ok := request.Options["splits"]; ok {
			meta["splits"] = splits
		}
		if duration, ok := request.Options["duration"]; ok {
			meta["duration"] = duration
		}

	case cmn.SmartContractTx:
		if gasLimit, ok := request.Options["gas_limit"]; ok {
			meta["gas_limit"] = gasLimit
		} else {
			if status == nil {
				status, err = cmn.GetStatus(s.client)
				if err != nil {
					terr := cmn.ErrInvalidInputParam
					terr.Message += "can't get blockchain status"
					return nil, terr
				}
			}

			height := uint64(status.CurrentHeight)
			gasLimit = ttypes.GetMaxGasLimit(height).Uint64()
		}
		if gasPrice, ok := request.Options["gas_price"]; ok {
			meta["gas_price"] = gasPrice
		} else {
			if status == nil {
				status, err = cmn.GetStatus(s.client)
				if err != nil {
					terr := cmn.ErrInvalidInputParam
					terr.Message += "can't get blockchain status"
					return nil, terr
				}
			}

			height := uint64(status.CurrentHeight)
			gasPrice = ttypes.GetMinimumGasPrice(height).Uint64()
		}
		if data, ok := request.Options["data"]; ok {
			meta["data"] = data
		}

	case cmn.DepositStakeTx:
		if purpose, ok := request.Options["purpose"]; ok {
			meta["purpose"] = purpose
		}

	case cmn.WithdrawStakeTx:
		if purpose, ok := request.Options["purpose"]; ok {
			meta["purpose"] = purpose
		}

	case cmn.StakeRewardDistributionTx:
		if splitBasisPoint, ok := request.Options["split_basis_point"]; ok {
			meta["split_basis_point"] = splitBasisPoint
		}

	default:
		terr := cmn.ErrUnableToParseTx
		terr.Message += "unsupported tx type"
		return nil, terr
	}

	return &types.ConstructionMetadataResponse{
		Metadata: meta,
		SuggestedFee: []*types.Amount{
			&types.Amount{
				Value: suggestedFee.String(),
				Currency: &types.Currency{
					Symbol:   ttypes.DenomTFuelWei,
					Decimals: cmn.CoinDecimals,
				},
			},
		},
	}, nil
}

// ConstructionPayloads implements the /construction/payloads endpoint.
func (s *constructionAPIService) ConstructionPayloads(
	ctx context.Context,
	request *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

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
	signBytes := tx.SignBytes(cmn.GetChainId())

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: unsignedTx,
		Payloads: []*types.SigningPayload{
			&types.SigningPayload{
				AccountIdentifier: &types.AccountIdentifier{
					Address: request.Operations[0].Account.Address,
				},
				Bytes:         crypto.Keccak256Hash(signBytes).Bytes(),
				SignatureType: SignatureType,
			},
		},
	}, nil
}

// ConstructionParse implements the /construction/parse endpoint.
func (s *constructionAPIService) ConstructionParse(
	ctx context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	rawTx, err := hex.DecodeString(request.Transaction)
	if err != nil {
		terr := cmn.ErrUnableToParseTx
		terr.Message += err.Error()
		return nil, terr
	}

	tx, err := ttypes.TxFromBytes(rawTx)
	if err != nil {
		terr := cmn.ErrUnableToParseTx
		terr.Message += err.Error()
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
		meta, ops = cmn.ParseSmartContractTx(tran, nil, cmn.SmartContractTx, uint64(0), nil) //TODO: gas used = 0?
	case *ttypes.DepositStakeTx, *ttypes.DepositStakeTxV2:
		tran := *tx.(*ttypes.DepositStakeTxV2)
		sender = tran.Source.Address.String()
		meta, ops = cmn.ParseDepositStakeTx(tran, nil, cmn.DepositStakeTx)
	case *ttypes.WithdrawStakeTx:
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

// ConstructionCombine implements the /construction/combine endpoint.
func (s *constructionAPIService) ConstructionCombine(
	ctx context.Context,
	request *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	rawTx, err := hex.DecodeString(request.UnsignedTransaction)
	if err != nil {
		terr := cmn.ErrUnableToParseTx
		terr.Message += err.Error()
		return nil, terr
	}

	tx, err := ttypes.TxFromBytes(rawTx)
	if err != nil {
		terr := cmn.ErrUnableToParseTx
		terr.Message += err.Error()
		return nil, terr
	}

	if len(request.Signatures) != 1 {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "need exact 1 signature"
		return nil, terr
	}

	sig, err := crypto.SignatureFromBytes(request.Signatures[0].Bytes)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += fmt.Sprintf("Cannot convert signature from payload bytes")
		return nil, terr
	}

	signer := common.HexToAddress(request.Signatures[0].SigningPayload.AccountIdentifier.Address)

	// Check signatures
	signBytes := tx.SignBytes(cmn.GetChainId())

	if !sig.Verify(signBytes, signer) {
		terr := cmn.ErrInvalidInputParam
		terr.Message += fmt.Sprintf("Signature verification failed, SignBytes: %v", hex.EncodeToString(signBytes))
		return nil, terr
	}

	switch tx.(type) {
	case *ttypes.CoinbaseTx:
		tran := *tx.(*ttypes.CoinbaseTx)
		tran.SetSignature(signer, sig)
	case *ttypes.SlashTx:
		tran := *tx.(*ttypes.SlashTx)
		tran.SetSignature(signer, sig)
	case *ttypes.SendTx:
		tran := *tx.(*ttypes.SendTx)
		tran.SetSignature(signer, sig)
	case *ttypes.ReserveFundTx:
		tran := *tx.(*ttypes.ReserveFundTx)
		tran.SetSignature(signer, sig)
	case *ttypes.ReleaseFundTx:
		tran := *tx.(*ttypes.ReleaseFundTx)
		tran.SetSignature(signer, sig)
	case *ttypes.ServicePaymentTx:
		// tran := *tx.(*ttypes.ServicePaymentTx)
		// tran.SetSignature(signer, sig)
	case *ttypes.SplitRuleTx:
		tran := *tx.(*ttypes.SplitRuleTx)
		tran.SetSignature(signer, sig)
	case *ttypes.SmartContractTx:
		tran := *tx.(*ttypes.SmartContractTx)
		tran.SetSignature(signer, sig)
	case *ttypes.DepositStakeTx, *ttypes.DepositStakeTxV2:
		tran := *tx.(*ttypes.DepositStakeTx)
		tran.SetSignature(signer, sig)
	case *ttypes.WithdrawStakeTx:
		tran := *tx.(*ttypes.WithdrawStakeTx)
		tran.SetSignature(signer, sig)
	case *ttypes.StakeRewardDistributionTx:
		tran := *tx.(*ttypes.StakeRewardDistributionTx)
		tran.SetSignature(signer, sig)
	default:
		terr := cmn.ErrUnableToParseTx
		terr.Message += "unsupported tx type"
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

// ConstructionHash implements the /construction/hash endpoint.
func (s *constructionAPIService) ConstructionHash(
	ctx context.Context,
	request *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	rawTx, err := hex.DecodeString(request.SignedTransaction)
	if err != nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "invalid signed transaction format: " + err.Error()
		return nil, terr
	}

	hash := crypto.Keccak256Hash(rawTx)

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: hash.String(),
		},
	}, nil
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *constructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if !strings.EqualFold(cmn.CfgRosettaModeOnline, viper.GetString(cmn.CfgRosettaMode)) {
		return nil, cmn.ErrUnavailableOffline
	}

	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	rpcRes, rpcErr := s.client.Call("theta.BroadcastRawTransactionAsync", BroadcastRawTransactionAsyncArgs{
		TxBytes: request.SignedTransaction,
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		broadcastResult := BroadcastRawTransactionAsyncResult{}
		err := json.Unmarshal(jsonBytes, &broadcastResult)
		if err != nil {
			return nil, err
		}

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

// decompressPubkey parses a public key in the 33-byte compressed format.
func decompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {
	x, y := secp256k1.DecompressPubkey(pubkey)
	if x == nil {
		return nil, fmt.Errorf("invalid public key")
	}
	return &ecdsa.PublicKey{X: x, Y: y, Curve: s256()}, nil
}

// s256 returns an instance of the secp256k1 curve.
func s256() elliptic.Curve {
	return secp256k1.S256()
}

func pubkeyToAddress(p ecdsa.PublicKey) common.Address {
	pubBytes := fromECDSAPub(&p)
	return common.BytesToAddress(keccak256(pubBytes[1:])[12:])
}

func fromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(s256(), pub.X, pub.Y)
}

// keccak256 calculates and returns the Keccak256 hash of the input data.
func keccak256(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
