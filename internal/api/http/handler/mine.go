package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Mine manually triggers mining: mines a new block immediately (even if the
// mempool is empty, in which case it contains only the coinbase).
func (h *Handler) Mine(c *gin.Context) {
	if err := h.node.ManualMine(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":        "mined",
		"miner_address": h.node.MinerAddress(),
	})
}

// WalletInfo returns the address used for coinbase outputs on this node.
func (h *Handler) WalletInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"miner_address": h.node.MinerAddress(),
	})
}
