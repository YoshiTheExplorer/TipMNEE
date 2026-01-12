package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct{}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

func (h *ConfigHandler) GetConfig(c *gin.Context) {
	chainID := os.Getenv("CHAIN_ID")
	escrow := os.Getenv("ESCROW_CONTRACT")
	
	// Optional: Return the token address if useful for the extension
	token := os.Getenv("TOKEN_CONTRACT")

	c.JSON(http.StatusOK, gin.H{
		"chain_id":        chainID,
		"escrow_contract": strings.ToLower(escrow),
		"token_contract":  strings.ToLower(token),
	})
}
