package handlers

import (
	"net/http"
	"strings"
	"time"

	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type IdentitiesHandler struct {
	store     *db.Queries
	jwtSecret string
}

func NewIdentitiesHandler(store *db.Queries, jwtSecret string) *IdentitiesHandler {
	return &IdentitiesHandler{store: store, jwtSecret: jwtSecret}
}

type loginResp struct {
	AccessToken string `json:"access_token"`
	UserID      int64  `json:"user_id"`
}

func (h *IdentitiesHandler) mintJWT(userID int64) (string, error) {
	claims := struct {
		UserID int64 `json:"user_id"`
		jwt.RegisteredClaims
	}{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.jwtSecret))
}

/*
Wallet login MVP:
- You SHOULD verify signature. For hackathon, you can start with "trust address" (not secure).
- Better: implement EIP-191 / personal_sign verification using go-ethereum crypto.
*/
type walletLoginReq struct {
	Address   string `json:"address" binding:"required"`
	Signature string `json:"signature"` // TODO verify
	Message   string `json:"message"`   // TODO verify
}

func (h *IdentitiesHandler) LoginWithWallet(c *gin.Context) {
	var req walletLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addr := strings.ToLower(strings.TrimSpace(req.Address))
	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}

	// TODO: verify signature here (recommended). For now, just proceed.

	ctx := c.Request.Context()

	// 1) identity exists?
	ident, err := h.store.GetIdentity(ctx, db.GetIdentityParams{
		Provider:       "wallet",
		ProviderUserID: addr,
	})

	var userID int64
	if err == nil {
		userID = ident.UserID
	} else {
		// 2) create user + identity
		u, err2 := h.store.CreateUser(ctx)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		_, err2 = h.store.CreateIdentity(ctx, db.CreateIdentityParams{
			UserID:         u.ID,
			Provider:       "wallet",
			ProviderUserID: addr,
		})
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create identity"})
			return
		}

		userID = u.ID
	}

	token, err := h.mintJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint token"})
		return
	}

	c.JSON(http.StatusOK, loginResp{AccessToken: token, UserID: userID})
}

type googleLoginReq struct {
	IDToken string `json:"id_token" binding:"required"`
}

/*
Google login:
- Verify ID token, extract "sub".
- Then same logic as wallet:
  identities(provider='google', provider_user_id=sub)
*/
func (h *IdentitiesHandler) LoginWithGoogle(c *gin.Context) {
	var req googleLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: verify req.IDToken, extract sub
	// For now, return 501 so you don’t ship insecure auth by accident.
	c.JSON(http.StatusNotImplemented, gin.H{"error": "google login not implemented yet (need to verify id_token)"})
}
