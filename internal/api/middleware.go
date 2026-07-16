package api

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

func requestMetadataMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
		return c.Next()
	}
}

func recoveryMiddleware() fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("api panic recovered: %v", recovered)
				err = internalError("internal server error", nil)
			}
		}()

		return c.Next()
	}
}
