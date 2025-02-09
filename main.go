package main

import (
	"log"
	"os"

	gin "github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")

	if port == "" {
		port = "8080"
	}
	if host == "" {
		host = "127.0.0.1"
	}

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	router.GET("/", func(c *gin.Context) {
		dial_monero_rpc("monero_rpc.crt", "127.0.0.1", 18081, "", "")
		resp, err := make_rpc_request(MoneroRPCRequest{
			Jsonrpc: "2.0",
			Method:  "get_address",
			Params:  map[string]interface{}{},
			ID:      0,
		})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		log.Println(resp.Status)
		log.Println(resp.Body)

		c.JSON(200, gin.H{
			"wallet_addr": resp.Body,
		})
	})

	router.Run(host + ":" + port)
}
