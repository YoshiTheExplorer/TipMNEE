package handlers

import (
	"net/http"
	"strings"

	"github.com/YoshiTheExplorer/TipMNEE/api/middleware"
	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/gin-gonic/gin"
)

type PayoutsHandler struct {
	store *db.Queries
}

func NewPayoutsHandler(store *db.Queries) *PayoutsHandler {
	return &PayoutsHandler{store: store}
}

type upsertPayoutReq struct {
	Chain   string `json:"chain" binding:"required"`   // "ethereum"
	Address string `json:"address" binding:"required"` // 0x...
}

func (h *PayoutsHandler) UpsertPayout(c *gin.Context) {
	userID := middleware.MustUserID(c)

	var req upsertPayoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cleanAddr := strings.ToLower(strings.TrimSpace(req.Address))
	if !common.IsHexAddress(cleanAddr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ethereum address format"})
		return
	}

	p, err := h.store.UpsertPayout(c.Request.Context(), db.UpsertPayoutParams{
		UserID:  userID,
		Chain:   req.Chain,
		Address: cleanAddr,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to set payout"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// Public: resolve channel payout for extension.
func (h *PayoutsHandler) ResolveYouTubeChannelPayout(c *gin.Context) {
	channelID := strings.TrimSpace(c.Param("channelId"))
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channelId param required"})
		return
	}

	addr, err := h.store.ResolvePayoutByChannelID(c.Request.Context(), db.ResolvePayoutByChannelIDParams{
		Platform:       "youtube",
		PlatformUserID: channelID,
		Chain:          "ethereum",
	})
	if err != nil {
		// Not claimed or no payout set -> tell client to use escrow
		c.JSON(http.StatusOK, gin.H{"status": "unclaimed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "direct", "address": addr})
}
