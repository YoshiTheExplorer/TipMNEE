package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	gethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type ClaimSigResult struct {
	ChannelIDHash string `json:"channel_id_hash"`
	Expiry        int64  `json:"expiry"`
	Nonce         string `json:"nonce"`
	Signature     string `json:"signature"`
}

func ChannelHash(channelID string) common.Hash {
	// matches solidity: keccak256(abi.encodePacked(channelId))
	return gethCrypto.Keccak256Hash([]byte(channelID))
}

func randBytes32Hex() (common.Hash, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(b), nil
}

// Build + sign EIP-712 Claim(...) to match your Solidity EIP712("TipMNEE","1") + CLAIM_TYPEHASH.
func BuildClaimPayload(
	verifierPrivHex string,
	chainID int64,
	escrow common.Address,
	channelID string,
	payout common.Address,
	ttl time.Duration,
) (*ClaimSigResult, error) {

	channelHash := ChannelHash(channelID)

	nonce, err := randBytes32Hex()
	if err != nil {
		return nil, err
	}

	expiry := time.Now().Add(ttl).Unix()

	sig, err := SignClaimEIP712(verifierPrivHex, chainID, escrow, channelHash, payout, expiry, nonce)
	if err != nil {
		return nil, err
	}

	return &ClaimSigResult{
		ChannelIDHash: channelHash.Hex(),
		Expiry:        expiry,
		Nonce:         nonce.Hex(),
		Signature:     sig,
	}, nil
}

func SignClaimEIP712(
	verifierPrivHex string,
	chainID int64,
	verifyingContract common.Address,
	channelIDHash common.Hash,
	payout common.Address,
	expiry int64,
	nonce common.Hash,
) (string, error) {

	priv, err := gethCrypto.HexToECDSA(strings.TrimPrefix(verifierPrivHex, "0x"))
	if err != nil {
		return "", err
	}

	// EIP-712 typed data (must match Solidity domain + type)
	td := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Claim": {
				{Name: "channelIdHash", Type: "bytes32"},
				{Name: "payoutAddress", Type: "address"},
				{Name: "expiry", Type: "uint256"},
				{Name: "nonce", Type: "bytes32"},
			},
		},
		PrimaryType: "Claim",
		Domain: apitypes.TypedDataDomain{
			Name:              "TipMNEE",
			Version:           "1",
			ChainId: math.NewHexOrDecimal256(chainID),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"channelIdHash": channelIDHash.Hex(),
			"payoutAddress": payout.Hex(),
			"expiry":        fmt.Sprintf("%d", expiry),
			"nonce":         nonce.Hex(),
		},
	}

	structHash, err := td.HashStruct(td.PrimaryType, td.Message)
	if err != nil {
		return "", err
	}
	domainSep, err := td.HashStruct("EIP712Domain", td.Domain.Map())
	if err != nil {
		return "", err
	}

	// digest = keccak256("\x19\x01" || domainSep || structHash)
	digest := gethCrypto.Keccak256Hash(
		[]byte{0x19, 0x01},
		domainSep,
		structHash,
	)

	sig, err := gethCrypto.Sign(digest.Bytes(), priv)
	if err != nil {
		return "", err
	}

	// go-ethereum returns v as 0/1; Solidity ECDSA.recover expects 27/28
	sig[64] += 27

	return "0x" + hex.EncodeToString(sig), nil
}
