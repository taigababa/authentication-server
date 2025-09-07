package httpx

import (
    "github.com/labstack/echo/v4"
)

type ErrorResponse struct {
    Message string `json:"message"`
    Detail  any    `json:"detail,omitempty"`
}

func JSONError(c echo.Context, code int, msg string, detail any) error {
    return c.JSON(code, ErrorResponse{Message: msg, Detail: detail})
}

