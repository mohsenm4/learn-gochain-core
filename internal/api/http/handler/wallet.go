package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetTxStatus(c *gin.Context) {
	id := c.Param("id")
	status := h.node.GetTxStatus(id)
	c.JSON(http.StatusOK, status)
}

func (h *Handler) GetBalance(c *gin.Context) {
	address := c.Param("address")
	balance := h.node.GetBalance(address)
	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"balance": balance,
	})
}

func (h *Handler) GetUTXOs(c *gin.Context) {
	address := c.Param("address")
	utxos := h.node.GetUTXOs(address)
	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"utxos":   utxos,
	})
}
