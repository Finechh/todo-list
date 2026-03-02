package middlewarex

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

func LoggerMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		duration := time.Since(start)

		log.Printf("%s %s [%d ms]", c.Request().Method, c.Request().URL.Path, duration.Milliseconds())
		return err
	}
}
