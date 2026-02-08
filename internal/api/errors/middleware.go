package errors

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			log.Printf("[ERROR] %v", err)

			var appErr *AppError
			if errors, ok := err.(*AppError); ok {
				appErr = errors
			} else {
				appErr = NewError(ErrCodeInternalError, "internal error", err.Error())
			}

			statusCode := MapCodeToStatus(appErr.Code)
			c.JSON(statusCode, Response{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
				Data:    nil,
			})
		}
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				c.JSON(http.StatusInternalServerError, Response{
					Code:    ErrCodeInternalError,
					Message: "internal error",
					Data:    nil,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
