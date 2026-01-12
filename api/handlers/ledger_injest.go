package handlers

import (
	"database/sql"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/YoshiTheExplorer/TipMNEE/api/middleware"
	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"
	util "github.com/YoshiTheExplorer/TipMNEE/util"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

const tipEscrowABIJSON = `[
  {"anonymous":false,"inputs":[
    {"indexed":true,"internalType":"bytes32","name":"channelIdHash","type":"bytes32"},
    {"indexed":true,"internalType":"address","name":"from","type":"address"},
    {"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},
    {"indexed":false,"internalType":"string","name":"message","type":"string"}
  ],"name":"Tipped","type":"event"},
  {"anonymous":false,"inputs":[
    {"indexed":true,"internalType":"bytes32","name":"channelIdHash","type":"bytes32"},
    {"indexed":true,"internalType":"address","name":"payoutAddress","type":"address"},
    {"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}
  ],"name":"Withdrawn","type":"event"}
]`

type LedgerIngestHandler struct {
	store       *db.Queries
	client      *ethclient.Client
	chainID     int64
	escrow      common.Address
	escrowABI   abi.ABI
	tippedID    common.Hash
	withdrawnID common.Hash
}

// isValidHexHash checks if a string is a valid Ethereum transaction hash.
func isValidHexHash(hash string) bool {
	return len(hash) == 66 && strings.HasPrefix(hash, "0x")
}

func NewLedgerIngestHandler(store *db.Queries) (*LedgerIngestHandler, error) {
	chainIDStr := strings.TrimSpace(os.Getenv("CHAIN_ID"))
	if chainIDStr == "" {
		return nil, errEnv("CHAIN_ID")
	}
	chainID, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		return nil, err
	}

	escrowStr := strings.TrimSpace(os.Getenv("ESCROW_CONTRACT"))
	if !common.IsHexAddress(escrowStr) {
		return nil, errEnv("ESCROW_CONTRACT (must be 0x...)")
	}
	escrow := common.HexToAddress(escrowStr)

	rpcURL := strings.TrimSpace(os.Getenv("SEPOLIA_RPC_URL"))
	if rpcURL == "" {
		rpcURL = strings.TrimSpace(os.Getenv("RPC_URL"))
	}
	if rpcURL == "" {
		return nil, errEnv("SEPOLIA_RPC_URL (or RPC_URL)")
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(tipEscrowABIJSON))
	if err != nil {
		return nil, err
	}

	return &LedgerIngestHandler{
		store:       store,
		client:      client,
		chainID:     chainID,
		escrow:      escrow,
		escrowABI:   parsedABI,
		tippedID:    parsedABI.Events["Tipped"].ID,
		withdrawnID: parsedABI.Events["Withdrawn"].ID,
	}, nil
}

type ingestReq struct {
	TxHash   string `json:"tx_hash" binding:"required"`
	ChannelID string `json:"channel_id" binding:"required"`
	ChainID  *int64  `json:"chain_id,omitempty"`
}

// PUBLIC: anyone can tip (no JWT)
func (h *LedgerIngestHandler) RecordDeposit(c *gin.Context) {
	var req ingestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ChainID != nil && *req.ChainID != h.chainID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong chain_id"})
		return
	}

	channelID := strings.TrimSpace(req.ChannelID)
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id required"})
		return
	}
	expectedHash := util.ChannelHash(channelID)

	txHashStr := strings.TrimSpace(req.TxHash)
	if !isValidHexHash(txHashStr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tx_hash"})
		return
	}
	txHash := common.HexToHash(txHashStr)

	ctx := c.Request.Context()

	// Ensure tx is to your escrow contract (extra safety)
	tx, _, err := h.client.TransactionByHash(ctx, txHash)
	if err != nil {
		if err == ethereum.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "tx not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tx"})
		return
	}
	if tx.To() == nil || *tx.To() != h.escrow {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tx not sent to escrow contract"})
		return
	}

	receipt, err := h.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		if err == ethereum.NotFound {
			c.JSON(http.StatusConflict, gin.H{"error": "tx not mined yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch receipt"})
		return
	}
	if receipt.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tx failed"})
		return
	}

	block, err := h.client.BlockByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch block"})
		return
	}
	blockTime := time.Unix(int64(block.Time()), 0).UTC()

	inserted := 0
	duplicates := 0

	// Determine user_id if channel is already verified/linked (optional)
	var userID sql.NullInt64
	if sl, err := h.store.GetSocialLinkByPlatformUser(ctx, db.GetSocialLinkByPlatformUserParams{
		Platform: "youtube", PlatformUserID: channelID,
	}); err == nil && sl.VerifiedAt.Valid {
		userID = sql.NullInt64{Int64: sl.UserID, Valid: true}
	}

	for _, lg := range receipt.Logs {
		if lg.Address != h.escrow || len(lg.Topics) == 0 {
			continue
		}
		if lg.Topics[0] != h.tippedID {
			continue
		}
		// topics[1] = channelIdHash
		if len(lg.Topics) < 2 || lg.Topics[1] != expectedHash {
			continue
		}

		// Decode non-indexed fields: amount, message
		var decoded struct {
			Amount  *big.Int
			Message string
		}
		if err := h.escrowABI.UnpackIntoInterface(&decoded, "Tipped", lg.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode Tipped log"})
			return
		}

		msg := sql.NullString{Valid: false}
		if strings.TrimSpace(decoded.Message) != "" {
			msg = sql.NullString{String: decoded.Message, Valid: true}
		}

		_, err := h.store.InsertLedgerEvent(ctx, db.InsertLedgerEventParams{
			Platform:       "youtube",
			PlatformUserID: channelID,
			UserID:         userID,
			EventType:      "TIP_ESCROW",
			AmountRaw:      decoded.Amount.String(),
			Message:        msg,
			TxHash:         lg.TxHash.Hex(),
			LogIndex:       int32(lg.Index),
			BlockTime:      blockTime,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				duplicates++
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "failed to insert ledger event",
				"detail": err.Error(),
			})

			return
		}
		inserted++
	}

	if inserted == 0 && duplicates == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no matching Tipped event found for channel_id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":         true,
		"inserted":   inserted,
		"duplicates": duplicates,
	})
}

// PROTECTED: only logged-in creator should record their withdrawal
func (h *LedgerIngestHandler) RecordWithdrawal(c *gin.Context) {
	user := middleware.MustUserID(c)
	if user == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req ingestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ChainID != nil && *req.ChainID != h.chainID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong chain_id"})
		return
	}

	channelID := strings.TrimSpace(req.ChannelID)
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id required"})
		return
	}
	expectedHash := util.ChannelHash(channelID)

	txHashStr := strings.TrimSpace(req.TxHash)
	if !isValidHexHash(txHashStr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tx_hash"})
		return
	}
	txHash := common.HexToHash(txHashStr)

	ctx := c.Request.Context()

	// Must own + be verified for this channel
	sl, err := h.store.GetSocialLinkByPlatformUser(ctx, db.GetSocialLinkByPlatformUserParams{
		Platform: "youtube", PlatformUserID: channelID,
	})
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "channel not linked"})
		return
	}
	if sl.UserID != user {
		c.JSON(http.StatusForbidden, gin.H{"error": "channel linked to another user"})
		return
	}
	if !sl.VerifiedAt.Valid {
		c.JSON(http.StatusForbidden, gin.H{"error": "channel not verified"})
		return
	}

	// Ensure tx is to escrow
	tx, _, err := h.client.TransactionByHash(ctx, txHash)
	if err != nil {
		if err == ethereum.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "tx not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tx"})
		return
	}
	if tx.To() == nil || *tx.To() != h.escrow {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tx not sent to escrow contract"})
		return
	}

	receipt, err := h.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		if err == ethereum.NotFound {
			c.JSON(http.StatusConflict, gin.H{"error": "tx not mined yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch receipt"})
		return
	}
	if receipt.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tx failed"})
		return
	}

	block, err := h.client.BlockByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch block"})
		return
	}
	blockTime := time.Unix(int64(block.Time()), 0).UTC()

	inserted := 0
	duplicates := 0

	for _, lg := range receipt.Logs {
		if lg.Address != h.escrow || len(lg.Topics) == 0 {
			continue
		}
		if lg.Topics[0] != h.withdrawnID {
			continue
		}
		// topics[1] = channelIdHash
		if len(lg.Topics) < 2 || lg.Topics[1] != expectedHash {
			continue
		}

		var decoded struct {
			Amount *big.Int
		}
		if err := h.escrowABI.UnpackIntoInterface(&decoded, "Withdrawn", lg.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode Withdrawn log"})
			return
		}

		_, err := h.store.InsertLedgerEvent(ctx, db.InsertLedgerEventParams{
			Platform:       "youtube",
			PlatformUserID: channelID,
			UserID:         sql.NullInt64{Int64: user, Valid: true},
			EventType:      "WITHDRAW",
			AmountRaw:      decoded.Amount.String(),
			Message:        sql.NullString{Valid: false},
			TxHash:         lg.TxHash.Hex(),
			LogIndex:       int32(lg.Index),
			BlockTime:      blockTime,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				duplicates++
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert ledger event"})
			return
		}
		inserted++
	}

	if inserted == 0 && duplicates == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no matching Withdrawn event found for channel_id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":         true,
		"inserted":   inserted,
		"duplicates": duplicates,
	})
}
