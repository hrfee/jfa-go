package main

import "github.com/gin-gonic/gin"

func (app *appContext) HelloWorld(gc *gin.Context) {
	gc.JSON(200, stringResponse{"It worked!", "none"})
}
