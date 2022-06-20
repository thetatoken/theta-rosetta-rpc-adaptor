package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/coinbase/rosetta-sdk-go/parser"
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

	if request.PublicKey == nil {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "public key is not provided"
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

	matches, e := getOperationDescriptions(request.Operations)
	if e != nil {
		return nil, e
	}

	if len(matches) == 2 {
		options["type"] = cmn.SmartContractTx
	} else if len(matches) == 5 { // SendTx
		fromThetaOp, _ := matches[0].First()
		fromTFuelOp, _ := matches[1].First()
		toThetaOp, _ := matches[2].First()
		toTFuelOp, _ := matches[3].First()
		feeOp, feeWei := matches[4].First()

		if fromThetaOp.Account.Address != fromTFuelOp.Account.Address || fromTFuelOp.Account.Address != feeOp.Account.Address {
			terr := cmn.ErrServiceInternal
			terr.Message += "from address not matching"
			return nil, terr
		}

		if toThetaOp.Account.Address != toTFuelOp.Account.Address {
			terr := cmn.ErrServiceInternal
			terr.Message += "to address not matching"
			return nil, terr
		}

		if fromTFuelOp.Account.Address == toTFuelOp.Account.Address {
			terr := cmn.ErrServiceInternal
			terr.Message += "from and to accounts are the same"
			return nil, terr
		}

		options["type"] = cmn.SendTx
		options["fee"] = feeWei
	} else {
		err := cmn.ErrServiceInternal
		err.Message += "invalid number of operations"
		return nil, err
	}

	fromOp, _ := matches[0].First()
	options["signer"] = fromOp.Account.Address

	if gasLimit, ok := request.Metadata["gas_limit"]; ok {
		options["gas_limit"] = gasLimit
	}
	if gasPrice, ok := request.Metadata["gas_price"]; ok {
		options["gas_price"] = gasPrice
	}
	if data, ok := request.Metadata["data"]; ok {
		options["data"] = data
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
	var signer interface{}
	if signer, ok = request.Options["signer"]; !ok {
		terr := cmn.ErrInvalidInputParam
		terr.Message += "empty signer address"
		return nil, terr
	}

	rpcRes, rpcErr := s.client.Call("theta.GetAccount", GetAccountArgs{
		Address: signer.(string),
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

	// meta["type"] = txType

	var status *cmn.GetStatusResult
	suggestedFee := big.NewInt(0)

	if cmn.TxType(txType.(float64)) == cmn.SendTx {
		if fee, ok := request.Options["fee"]; ok {
			suggestedFee = new(big.Int).SetUint64(uint64(fee.(float64)))
		}
		if suggestedFee.Cmp(big.NewInt(0)) == 0 {
			status, err = cmn.GetStatus(s.client)
			if err != nil {
				terr := cmn.ErrInvalidInputParam
				terr.Message += "can't get blockchain status"
				return nil, terr
			}
			height := uint64(status.CurrentHeight)
			suggestedFee = ttypes.GetSendTxMinimumTransactionFeeTFuelWei(2, height) // only allow 1-to-1 transfer
		}
		meta["fee"] = suggestedFee
	} else if cmn.TxType(txType.(float64)) == cmn.SmartContractTx {
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
			meta["gas_limit"] = gasLimit
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
			meta["gas_price"] = gasPrice
		}

		if data, ok := request.Options["data"]; ok {
			meta["data"] = data
		}
	}

	return &types.ConstructionMetadataResponse{
		Metadata: meta,
		SuggestedFee: []*types.Amount{
			{
				Value:    suggestedFee.String(),
				Currency: cmn.GetTFuelCurrency(),
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

	var ok bool
	var err error
	var seq interface{}
	if seq, ok = request.Metadata["sequence"]; !ok {
		terr := cmn.ErrServiceInternal
		terr.Message += "missing tx sequence"
		return nil, terr
	}
	sequence := uint64(seq.(float64))

	var tx ttypes.Tx

	matches, e := getOperationDescriptions(request.Operations)
	if e != nil {
		return nil, e
	}

	if len(request.Operations) == 2 { // SmartContractTx
		fromOp, fromTFuelWei := matches[0].First()
		from := ttypes.TxInput{
			Address: common.HexToAddress(fromOp.Account.Address),
			Coins: ttypes.Coins{
				ThetaWei: new(big.Int).SetUint64(0),
				TFuelWei: fromTFuelWei,
			},
			Sequence: sequence,
		}

		toOp, _ := matches[1].First()
		to := ttypes.TxOutput{
			Address: common.HexToAddress(toOp.Account.Address),
		}

		gasPriceStr := "0wei"
		if gasPrice, ok := request.Metadata["gas_price"]; ok {
			gasPriceStr = gasPrice.(string) + "wei"
		}
		gasPrice, ok := ttypes.ParseCoinAmount(gasPriceStr)
		if !ok {
			terr := cmn.ErrServiceInternal
			terr.Message += "failed to parse gas price"
			return nil, terr
		}

		gasLimit := uint64(10000000) //TODO: 10000000 or 0?
		if gasLim, ok := request.Metadata["gas_limit"]; ok {
			gasLimit, err = strconv.ParseUint(gasLim.(string), 10, 64)
			if err != nil {
				terr := cmn.ErrServiceInternal
				terr.Message += "failed to parse gas limit"
				return nil, terr
			}
		}

		var dataStr string
		if datum, ok := request.Metadata["data"]; ok {
			dataStr = datum.(string)
		}
		data, err := hex.DecodeString(dataStr)
		if err != nil {
			terr := cmn.ErrServiceInternal
			terr.Message += "failed to parse data"
			return nil, terr
		}

		tx = &ttypes.SmartContractTx{
			From:     from,
			To:       to,
			GasLimit: gasLimit, //uint64(meta["gas_limit"].(float64)),
			GasPrice: gasPrice, //big.NewInt(int64(meta["gas_price"].(float64))),
			Data:     data,     //request.Metadata["data"].([]byte),
		}

	} else if len(request.Operations) == 5 { // SendTx
		fromThetaOp, fromThetaWei := matches[0].First()
		_, fromTFuelWei := matches[1].First()
		toThetaOp, toThetaWei := matches[2].First()
		_, toTFuelWei := matches[3].First()
		_, feeWei := matches[4].First() // TODO: overwritten with request.Metadata["fee"] ?

		fromThetaWei = new(big.Int).Mul(fromThetaWei, big.NewInt(-1))
		fromTFuelWei = new(big.Int).Mul(fromTFuelWei, big.NewInt(-1))
		feeWei = new(big.Int).Mul(feeWei, big.NewInt(-1))

		inputs := []ttypes.TxInput{
			{
				Address: common.HexToAddress(fromThetaOp.Account.Address),
				Coins: ttypes.Coins{
					ThetaWei: fromThetaWei,
					TFuelWei: new(big.Int).Add(fromTFuelWei, feeWei),
				},
				Sequence: uint64(seq.(float64)),
			},
		}

		outputs := []ttypes.TxOutput{
			{
				Address: common.HexToAddress(toThetaOp.Account.Address),
				Coins: ttypes.Coins{
					ThetaWei: toThetaWei,
					TFuelWei: toTFuelWei,
				},
			},
		}

		tx = &ttypes.SendTx{
			Fee: ttypes.Coins{
				ThetaWei: new(big.Int).SetUint64(0),
				TFuelWei: feeWei,
			},
			Inputs:  inputs,
			Outputs: outputs,
		}
	} else {
		err := cmn.ErrServiceInternal
		err.Message += "invalid number of operations"
		return nil, err
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
			{
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

	var signer string
	var meta map[string]interface{}
	var ops []*types.Operation

	switch tx.(type) {
	case *ttypes.SendTx:
		tran := *tx.(*ttypes.SendTx)
		signer = tran.Inputs[0].Address.String()
		meta, ops = cmn.ParseSendTx(tran, nil, cmn.SendTx)
	case *ttypes.SmartContractTx:
		tran := *tx.(*ttypes.SmartContractTx)
		signer = tran.From.Address.String()
		meta, ops = cmn.ParseSmartContractTx(tran, nil, cmn.SmartContractTx, uint64(0), nil) //TODO: gas used = 0?
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
			{
				Address: signer,
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
	case *ttypes.SendTx:
		tran := *tx.(*ttypes.SendTx)
		tran.SetSignature(signer, sig)
	case *ttypes.SmartContractTx:
		tran := *tx.(*ttypes.SmartContractTx)
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

func getOperationDescriptions(operations []*types.Operation) (matches []*parser.Match, err *types.Error) {
	var e error
	if len(operations) == 2 { // SmartContractTx
		descriptions := &parser.Descriptions{
			OperationDescriptions: []*parser.OperationDescription{
				{
					Type: cmn.SmartContractTxFrom.String(),
					Account: &parser.AccountDescription{
						Exists: true,
					},
					Amount: &parser.AmountDescription{
						Exists:   true,
						Sign:     parser.NegativeOrZeroAmountSign,
						Currency: cmn.GetTFuelCurrency(),
					},
				},
				{
					Type: cmn.SmartContractTxTo.String(),
					Account: &parser.AccountDescription{
						Exists: true,
					},
					Amount: &parser.AmountDescription{
						Exists:   true,
						Sign:     parser.PositiveOrZeroAmountSign,
						Currency: cmn.GetTFuelCurrency(),
					},
				},
			},
			ErrUnmatched: true,
		}

		matches, e = parser.MatchOperations(descriptions, operations)
		if e != nil {
			err = cmn.ErrServiceInternal
			err.Message += e.Error()
		}
	} else if len(operations) == 5 { // SendTx
		descriptions := &parser.Descriptions{
			OperationDescriptions: []*parser.OperationDescription{
				{
					Type: cmn.SendTxInput.String(),
					Account: &parser.AccountDescription{
						Exists: true,
					},
					Amount: &parser.AmountDescription{
						Exists:   true,
						Sign:     parser.NegativeOrZeroAmountSign,
						Currency: cmn.GetThetaCurrency(),
					},
				},
				{
					Type: cmn.SendTxInput.String(),
					Account: &parser.AccountDescription{
						Exists: true,
					},
					Amount: &parser.AmountDescription{
						Exists:   true,
						Sign:     parser.NegativeOrZeroAmountSign,
						Currency: cmn.GetTFuelCurrency(),
					},
				},
				{
					Type: cmn.SendTxOutput.String(),
					Account: &parser.AccountDescription{
						Exists: true,
					},
					Amount: &parser.AmountDescription{
						Exists:   true,
						Sign:     parser.PositiveOrZeroAmountSign,
						Currency: cmn.GetThetaCurrency(),
					},
				},
				{
					Type: cmn.SendTxOutput.String(),
					Account: &parser.AccountDescription{
						Exists: true,
					},
					Amount: &parser.AmountDescription{
						Exists:   true,
						Sign:     parser.PositiveOrZeroAmountSign,
						Currency: cmn.GetTFuelCurrency(),
					},
				},
				{
					Type: cmn.TxFee.String(),
					Account: &parser.AccountDescription{
						Exists: true,
					},
					Amount: &parser.AmountDescription{
						Exists:   true,
						Sign:     parser.NegativeOrZeroAmountSign,
						Currency: cmn.GetTFuelCurrency(),
					},
				},
			},
			ErrUnmatched: true,
		}

		matches, e = parser.MatchOperations(descriptions, operations)
		if e != nil {
			err = cmn.ErrServiceInternal
			err.Message += e.Error()
		}
	} else {
		err = cmn.ErrServiceInternal
		err.Message += "invalid number of operations"
	}

	return
}
