package ginerr

import (
	"context"
	"fmt"
	"net/http"
)

const defaultCode = http.StatusInternalServerError

// DefaultErrorRegistry is the default ErrorRegistry for the application, can be overridden for rare use-cases.
var DefaultErrorRegistry = NewErrorRegistry()

type internalHandler func(ctx context.Context, err error) (int, any)
type internalStringHandler func(ctx context.Context, err string) (int, any)

// ErrorHandler is the template of an error handler in the ErrorRegistry. The E type is the error type that
// the handler is registered for. The R type is the type of the response body.
type ErrorHandler[E error] interface {
	func(E) (int, any) |
		func(context.Context, E) (int, any)
}

// NewErrorRegistry is ideal for testing or overriding the default one.
func NewErrorRegistry() *ErrorRegistry {
	registry := &ErrorRegistry{
		handlers:       make(map[string]internalHandler),
		stringHandlers: make(map[string]internalStringHandler),
		DefaultCode:    defaultCode,
	}

	// Make sure the stringHandlers are available in the handlers
	registry.handlers["*errors.errorString"] = func(ctx context.Context, err error) (int, any) {
		// Check if the error string exists
		if handler, ok := registry.stringHandlers[err.Error()]; ok {
			return handler(ctx, err.Error())
		}

		return registry.DefaultCode, registry.DefaultResponse
	}

	return registry
}

// ErrorRegistry contains a map of ErrorHandlers.
type ErrorRegistry struct {
	// handlers are used when we know the type of the error
	handlers map[string]internalHandler

	// stringHandlers are used when the error is only a string
	stringHandlers map[string]internalStringHandler

	// DefaultCode to return when no handler is found
	DefaultCode int

	// DefaultResponse to return when no handler is found
	DefaultResponse any
}

func (e *ErrorRegistry) SetDefaultResponse(code int, response any) {
	e.DefaultCode = code
	e.DefaultResponse = response
}

// NewErrorResponse Returns an error response using the DefaultErrorRegistry. If no specific handler could be found,
// it will return the defaults.
func NewErrorResponse(ctx context.Context, err error) (int, any) {
	return NewErrorResponseFrom(DefaultErrorRegistry, ctx, err)
}

// NewErrorResponseFrom Returns an error response using the given registry. If no specific handler could be found,
// it will return the defaults.
func NewErrorResponseFrom(registry *ErrorRegistry, ctx context.Context, err error) (int, any) {
	errorType := fmt.Sprintf("%T", err)

	// If a handler is registered for the error type, use it.
	if entry, ok := registry.handlers[errorType]; ok {
		return entry(ctx, err)
	}

	// In production, we should return a generic error message. If you want to know why, read this:
	// https://owasp.org/www-community/Improper_Error_Handling
	return registry.DefaultCode, registry.DefaultResponse
}

// RegisterErrorHandler registers an error handler in DefaultErrorRegistry. The R type is the type of the response body.
func RegisterErrorHandler[E error](handler func(context.Context, E) (int, any)) {
	RegisterErrorHandlerOn(DefaultErrorRegistry, handler)
}

// RegisterErrorHandlerOn registers an error handler in the given registry. The R type is the type of the response body.
func RegisterErrorHandlerOn[E error](registry *ErrorRegistry, handler func(context.Context, E) (int, any)) {
	// Name of the type
	errorType := fmt.Sprintf("%T", *new(E))

	// Wrap it in a closure, we can't save it directly because err E is not available in NewErrorResponseFrom. It will
	// be available in the closure when it is called. Check out TestErrorResponseFrom_ReturnsErrorBInInterface for an example.
	registry.handlers[errorType] = func(ctx context.Context, err error) (int, any) {
		return handler(ctx, err.(E))
	}
}

// RegisterCustomErrorTypeHandler registers an error handler in DefaultErrorRegistry. Same as RegisterErrorHandler,
// but you can set the fmt.Sprint("%T", err) error yourself. Allows you to register error types that aren't exported
// from their respective packages such as the uuid error or *errors.errorString. The R type is the type of the response body.
func RegisterCustomErrorTypeHandler(errorType string, handler func(ctx context.Context, err error) (int, any)) {
	RegisterCustomErrorTypeHandlerOn(DefaultErrorRegistry, errorType, handler)
}

// RegisterCustomErrorTypeHandlerOn registers an error handler in the given registry. Same as RegisterErrorHandlerOn,
// but you can set the fmt.Sprint("%T", err) error yourself. Allows you to register error types that aren't exported
// from their respective packages such as the uuid error or *errors.errorString. The R type is the type of the response body.
func RegisterCustomErrorTypeHandlerOn(registry *ErrorRegistry, errorType string, handler func(ctx context.Context, err error) (int, any)) {
	// Wrap it in a closure, we can't save it directly
	registry.handlers[errorType] = handler
}

// RegisterStringErrorHandler allows you to register an error handler for a simple errorString created with
// errors.New() or fmt.Errorf(). Can be used in case you are dealing with libraries that don't have exported
// error objects. Uses the DefaultErrorRegistry. The R type is the type of the response body.
func RegisterStringErrorHandler(errorString string, handler func(ctx context.Context, err string) (int, any)) {
	RegisterStringErrorHandlerOn(DefaultErrorRegistry, errorString, handler)
}

// RegisterStringErrorHandlerOn allows you to register an error handler for a simple errorString created with
// errors.New() or fmt.Errorf(). Can be used in case you are dealing with libraries that don't have exported
// error objects. The R type is the type of the response body.
func RegisterStringErrorHandlerOn(registry *ErrorRegistry, errorString string, handler func(ctx context.Context, err string) (int, any)) {
	registry.stringHandlers[errorString] = handler
}
