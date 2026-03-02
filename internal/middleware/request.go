package middlewarex

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Rqctx string

var RqID Rqctx

func RequestID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := uuid.NewString()
		ctx := c.Request().Context()
		ctx = context.WithValue(ctx, RqID, id)

		req := c.Request().WithContext(ctx)
		c.SetRequest(req)

		return next(c)

	}

}

func RequestCtx(c echo.Context, sec int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request().Context(), time.Duration(sec)*time.Second)
}
