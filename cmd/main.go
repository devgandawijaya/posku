package main

import (
	"posku/internal/config"
	"posku/internal/database"
	"posku/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()
	database.Connect()
	database.Migrate()
	r := gin.Default()
	routes.SetupRoutes(r)
	r.Run(":" + config.AppPort)
}
