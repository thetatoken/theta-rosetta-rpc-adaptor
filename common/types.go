package common

import (
	"math/big"

	"github.com/thetatoken/theta/ledger/types"
)

const (
	// Theta Symbol
	Theta = "THETA"
	// TFuel Symbol
	TFuel = "TFUEL"
	// Decimals
	CoinDecimals = 18
)

const (
	// HashLength is the expected length of the hash
	HashLength = 32
	// AddressLength is the expected length of the adddress
	AddressLength = 20
)

type Bytes []byte
type Hash [HashLength]byte
type JSONUint64 uint64
type Address [AddressLength]byte

type Coins struct {
	ThetaWei *big.Int
	TFuelWei *big.Int
}

type Signature struct {
	data Bytes
}

type TxInput struct {
	Address   Address // Hash of the PubKey
	Coins     Coins
	Sequence  uint64     // Must be 1 greater than the last committed TxInput
	Signature *Signature // Depends on the PubKey type and the whole Tx
}

type ServicePaymentTx struct {
	Fee             Coins   // Fee
	Source          TxInput // source account
	Target          TxInput // target account
	PaymentSequence uint64  // each on-chain settlement needs to increase the payment sequence by 1
	ReserveSequence uint64  // ReserveSequence to locate the ReservedFund
	ResourceID      string  // The corresponding resourceID
}

type TransferRecord struct {
	ServicePayment ServicePaymentTx `json:"service_payment"`
}

type ReservedFund struct {
	Collateral      Coins
	InitialFund     Coins
	UsedFund        Coins
	ResourceIDs     []string // List of resource ID
	EndBlockHeight  uint64
	ReserveSequence uint64           // sequence number of the corresponding ReserveFundTx transaction
	TransferRecords []TransferRecord // signed ServerPaymentTransactions
}

type Account struct {
	Address                Address
	Sequence               uint64
	Balance                Coins
	ReservedFunds          []ReservedFund // TODO: replace the slice with map
	LastUpdatedBlockHeight uint64

	// Smart contract
	Root     Hash `json:"root"`      // merkle root of the storage trie
	CodeHash Hash `json:"code_hash"` // hash of the smart contract code
}

type GetAccountArgs struct {
	Address string `json:"address"`
	// Height  JSONUint64 `json:"height"`
	Preview bool `json:"preview"` // preview the account balance from the ScreenedView
}

type GetAccountResult struct {
	*types.Account
	Address string `json:"address"`
}

type GetBlockByHeightArgs struct {
	Height JSONUint64 `json:"height"`
}
