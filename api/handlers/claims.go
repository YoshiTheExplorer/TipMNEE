package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"

	"github.com/YoshiTheExplorer/TipMNEE/api/middleware"
	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"

	util "github.com/YoshiTheExplorer/TipMNEE/util"
)

type ClaimsHandler struct {
	store 		 	*db.Queries
	chainID         int64
	escrowContract  common.Address
	verifierPrivKey string
}

func NewClaimsHandler(store *db.Queries) (*ClaimsHandler, error) {
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

	verifierPK := strings.TrimSpace(os.Getenv("VERIFIER_PRIVATE_KEY"))
	if verifierPK == "" {
		return nil, errEnv("VERIFIER_PRIVATE_KEY")
	}

	return &ClaimsHandler{
		store: 			 store,
		chainID:         chainID,
		escrowContract:  common.HexToAddress(escrowStr),
		verifierPrivKey: verifierPK,
	}, nil
}

type errEnv string

func (e errEnv) Error() string { return "missing/invalid env: " + string(e) }

type youtubeClaimReq struct {
	ChannelID     string `json:"channel_id" binding:"required"`
	PayoutAddress string `json:"payout_address" binding:"required"`
}

func (h *ClaimsHandler) SignYouTubeClaim(c *gin.Context) {
	// RequireJWT middleware should set this
	if _, ok := c.Get("user_id"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth"})
		return
	}

	userID := middleware.MustUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req youtubeClaimReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channelID := strings.TrimSpace(req.ChannelID)
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id required"})
		return
	}

	if !common.IsHexAddress(req.PayoutAddress) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payout_address"})
		return
	}
	payout := common.HexToAddress(req.PayoutAddress)

	ctx := c.Request.Context()
    sl, err := h.store.GetSocialLinkByPlatformUser(ctx, db.GetSocialLinkByPlatformUserParams{
        Platform:       "youtube",
        PlatformUserID: channelID,
    })
    if err != nil {
        c.JSON(http.StatusForbidden, gin.H{"error": "channel not linked"})
        return
    }
    if sl.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "channel linked to another user"})
        return
    }
    if !sl.VerifiedAt.Valid {
        c.JSON(http.StatusForbidden, gin.H{"error": "channel not verified"})
        return
    }

	payload, err := util.BuildClaimPayload(
		h.verifierPrivKey,
		h.chainID,
		h.escrowContract,
		channelID,
		payout,
		10*time.Minute,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign claim"})
		return
	}

	c.JSON(http.StatusOK, payload)
}
