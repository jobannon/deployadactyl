package interfaces

import "github.com/gin-gonic/gin"

type Endpoints interface {
	Deploy(c *gin.Context)
}
