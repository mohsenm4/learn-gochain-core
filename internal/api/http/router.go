package http

import (
	"github.com/Mohsen20031203/learn-gochain-core/internal/api/http/handler"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func NewRouter(handler *handler.Handler) *gin.Engine {
	router := gin.Default()

	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://65.109.219.253:3000", "http://65.109.219.253:3001", "http://localhost:3000", "http://localhost:3001", "https://doctor.actelmon.ir", "https://admin.actelmon.ir", "https://app.actelmon.ir", "http://192.168.254.23:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
	}))

	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/info", handler.GetInfo)
	router.GET("/network", handler.GetNetwork)
	router.GET("/walletinfo", handler.WalletInfo)
	router.GET("/chain", handler.GetChain)
	router.POST("/transactions", handler.SubmitTransactions)
	router.GET("/transactions/:id", handler.GetTxStatus)
	router.GET("/block/:hash", handler.GetBlockByHash)
	router.GET("/mempool", handler.GetMempool)
	router.GET("/balance/:address", handler.GetBalance)
	router.GET("/utxos/:address", handler.GetUTXOs)
	router.POST("/mine", handler.Mine)

	return router
}
