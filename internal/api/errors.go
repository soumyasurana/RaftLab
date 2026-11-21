package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// APIError represents an HTTP error that should be rendered to clients.
type APIError struct {
	Status  int
	Message string
	Err     error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// ErrorResponse is the common JSON error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}

func newAPIError(status int, message string, err error) *APIError {
	return &APIError{
		Status:  status,
		Message: message,
		Err:     err,
	}
}

func badRequest(message string) *APIError {
	return newAPIError(http.StatusBadRequest, message, nil)
}

func internalError(message string, err error) *APIError {
	return newAPIError(http.StatusInternalServerError, message, err)
}

func snapshotUnavailable(message string) *APIError {
	return newAPIError(http.StatusConflict, message, nil)
}

func statusFromError(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return http.StatusRequestTimeout
	}

	return http.StatusInternalServerError
}

func messageFromError(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.Message != "" {
			return apiErr.Message
		}
	}

	return err.Error()
}
