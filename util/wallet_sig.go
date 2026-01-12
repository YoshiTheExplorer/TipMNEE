package util

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func RecoverAddressFromPersonalSign(message string, signatureHex string) (string, error) {
  sig := strings.TrimPrefix(signatureHex, "0x")
  sigBytes, err := hex.DecodeString(sig)
  if err != nil {
    return "", err
  }
  if len(sigBytes) != 65 {
    return "", errors.New("invalid signature length")
  }

  // MetaMask sometimes returns v=27/28; go-ethereum expects 0/1
  if sigBytes[64] >= 27 {
    sigBytes[64] -= 27
  }
  if sigBytes[64] != 0 && sigBytes[64] != 1 {
    return "", errors.New("invalid signature recovery id (v)")
  }

  prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
  hash := crypto.Keccak256Hash([]byte(prefixed))

  pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
  if err != nil {
    return "", err
  }

  addr := crypto.PubkeyToAddress(*pubKey).Hex()
  return strings.ToLower(addr), nil
}
