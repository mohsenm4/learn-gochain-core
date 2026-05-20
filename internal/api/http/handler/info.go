package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetInfo(c *gin.Context) {
	c.JSON(http.StatusOK, h.node.GetChainInfo())
}

func (h *Handler) GetNetwork(c *gin.Context) {
	c.JSON(http.StatusOK, h.node.GetNetworkInfo())
}
