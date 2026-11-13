package server

import (
	"io"
	"strings"
	"unicode"

	"aihub/internal/observability"
	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func accessLogMiddlewares(output io.Writer) []fiber.Handler {
	return []fiber.Handler{
		sanitizeInboundRequestID,
		requestid.New(requestid.Config{Header: correlationHeader}),
		propagateRequestID,
		fiberlogger.New(fiberlogger.Config{
			Output:        output,
			DisableColors: true,
			TimeFormat:    "2006-01-02T15:04:05.000Z07:00",
			TimeZone:      "UTC",
			Format:        "${time} method=${method} path=${path} status=${status} latency=${latency} request_id=${locals:requestid} cf_ray=${cf_ray}\n",
			CustomTags: map[string]fiberlogger.LogFunc{
				"cf_ray": cloudflareRayLogTag,
			},
		}),
	}
}

func accessLogEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true":
		return true
	default:
		return false
	}
}

func sanitizeInboundRequestID(c *fiber.Ctx) error {
	requestID := c.Get(fiber.HeaderXRequestID)
	if !observability.IsValidCorrelationID(requestID) {
		c.Request().Header.Del(fiber.HeaderXRequestID)
	}
	return c.Next()
}

func propagateRequestID(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestid").(string)
	c.Request().Header.Set(fiber.HeaderXRequestID, requestID)
	c.Locals(accessLogManagedContextKey{}, true)
	return c.Next()
}

func cloudflareRayLogTag(output fiberlogger.Buffer, c *fiber.Ctx, _ *fiberlogger.Data, _ string) (int, error) {
	return output.WriteString(sanitizeAccessLogValue(c.Get("Cf-Ray"), 128))
}

func sanitizeAccessLogValue(value string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}

	result := make([]rune, 0, min(len(value), maxLength))
	for _, char := range value {
		if len(result) >= maxLength {
			break
		}
		if unicode.IsControl(char) {
			continue
		}
		result = append(result, char)
	}
	return string(result)
}
