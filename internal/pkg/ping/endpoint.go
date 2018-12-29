package ping

import (
	"net/http"

	"github.com/teera123/gin"
)

//Endpoint for test call
func Endpoint(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "ping"})
}
