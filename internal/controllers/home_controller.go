package controllers

import "github.com/gin-gonic/gin"

func Home(c *gin.Context) {
    c.JSON(200, gin.H{
        "message": "Selamat datang di golangku!",
        "status":  "running on port 2040",
    })
}
