package middlewarex

import (
	"log"
	"net/http"
	errorsx "todo_list/internal/errors"

	"github.com/labstack/echo/v4"
)

func ErrorHandler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Request().Context().Value(RqID)

		err := next(c)
		if err == nil {
			return nil
		} else {
			log.Printf("ERROR: %v %v", id, err)
		}
		if appErr, ok := err.(*errorsx.AppError); ok {
			return c.JSON(appErr.Status, appErr)
		}
		if httpErr, ok := err.(*echo.HTTPError); ok {
			status := http.StatusInternalServerError
			if httpErr.Code != 0 {
				status = httpErr.Code
			}
			return c.JSON(status, map[string]interface{}{
				"code":    http.StatusText(status),
				"message": httpErr.Message,
				"status":  status,
			})
		}
		return c.JSON(http.StatusInternalServerError, errorsx.AppError{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
	}

}
