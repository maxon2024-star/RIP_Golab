package api

import (
	"RIP_Golab/internal/app/handler"
	"log"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	log.Println("Starting server")

	h := handler.NewHandler()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources/static")

	r.GET("/", h.GetServiceList)
	r.GET("/service/:id", h.GetServiceDetail)
	r.GET("/request", h.GetRequest)
	r.POST("/add-to-request", h.AddToRequest)
	r.POST("/update-request", h.UpdateRequest)
	r.POST("/remove-from-request", h.RemoveFromRequest)

	r.Run()
	log.Println("Server down")
}
