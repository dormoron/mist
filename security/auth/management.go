package auth

import (
	"errors"
	"fmt"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/auth/kit"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// bearerPrefix defines the prefix to be used for Bearer tokens.
const bearerPrefix = "Bearer"

// Error definitions.
var (
	errEmptyRefreshOpts = errors.New("refreshJWTOptions are nil") // Error for nil refreshJWTOptions.
)

// Management struct is a generic type that manages the configuration and operations
// for token management, including access and refresh tokens.
type Management[T any] struct {
	allowTokenHeader    string           // Header containing the authorization token.
	exposeAccessHeader  string           // Header to expose the access token.
	exposeRefreshHeader string           // Header to expose the refresh token.
	accessJWTOptions    Options          // Options for access JWT.
	refreshJWTOptions   *Options         // Options for refresh JWT.
	rotateRefreshToken  bool             // Whether to rotate the refresh token.
	nowFunc             func() time.Time // Function to retrieve the current time.
}

// InitManagement initializes a Management instance with the provided access JWT options
// and other optional configurations using variadic functional options.
// Parameters:
// - accessJWTOptions: Options for the access JWT (Options).
// - opts: A variadic list of functional options for configuring the Management instance (kit.Option[Management[T]]).
// Returns:
// - *Management[T]: A pointer to the initialized Management instance.
func InitManagement[T any](accessJWTOptions Options, opts ...kit.Option[Management[T]]) *Management[T] {
	// Initialize default options.
	dOpts := defaultManagementOptions[T]()

	// Set access JWT options.
	dOpts.accessJWTOptions = accessJWTOptions

	// Apply additional options if provided.
	kit.Apply[Management[T]](&dOpts, opts...)

	return &dOpts
}

// defaultManagementOptions returns a Management instance with default settings.
// Returns:
// - Management[T]: A Management instance with default configuration.
func defaultManagementOptions[T any]() Management[T] {
	return Management[T]{
		allowTokenHeader:    "authorization",  // Default header for authorization token.
		exposeAccessHeader:  "x-access-auth",  // Default header to expose access token.
		exposeRefreshHeader: "x-refresh-auth", // Default header to expose refresh token.
		rotateRefreshToken:  false,            // Default to not rotating refresh tokens.
		nowFunc:             time.Now,         // Default function to retrieve current time.
	}
}

// WithAllowTokenHeader is a functional option to set a custom token header in Management.
// Parameters:
// - header: The custom header to be used for tokens (string).
// Returns:
// - kit.Option[Management[T]]: A function that sets the token header in Management.
func WithAllowTokenHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.allowTokenHeader = header
	}
}

// WithExposeAccessHeader is a functional option to set a custom header to expose the access token in Management.
// Parameters:
// - header: The custom header to expose the access token (string).
// Returns:
// - kit.Option[Management[T]]: A function that sets the access token expose header in Management.
func WithExposeAccessHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.exposeAccessHeader = header
	}
}

// WithExposeRefreshHeader is a functional option to set a custom header to expose the refresh token in Management.
// Parameters:
// - header: The custom header to expose the refresh token (string).
// Returns:
// - kit.Option[Management[T]]: A function that sets the refresh token expose header in Management.
func WithExposeRefreshHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.exposeRefreshHeader = header
	}
}

// WithRefreshJWTOptions is a functional option to set options for refresh JWT in Management.
// Parameters:
// - refreshOpts: The options for the refresh JWT (Options).
// Returns:
// - kit.Option[Management[T]]: A function that sets the refresh JWT options in Management.
func WithRefreshJWTOptions[T any](refreshOpts Options) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.refreshJWTOptions = &refreshOpts
	}
}

// WithRotateRefreshToken is a functional option to set refresh token rotation in Management.
// Parameters:
// - isRotate: A boolean value indicating whether to rotate refresh tokens (bool).
// Returns:
// - kit.Option[Management[T]]: A function that sets the refresh token rotation in Management.
func WithRotateRefreshToken[T any](isRotate bool) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.rotateRefreshToken = isRotate
	}
}

// WithNowFunc is a functional option to set a custom function to retrieve the current time in Management.
// Parameters:
// - nowFunc: The custom function to retrieve the current time (func() time.Time).
// Returns:
// - kit.Option[Management[T]]: A function that sets the current time function in Management.
func WithNowFunc[T any](nowFunc func() time.Time) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.nowFunc = nowFunc
	}
}

// Refresh handles the process of refreshing tokens in an HTTP context. It verifies the refresh token,
// generates a new access token, and optionally generates a new refresh token, setting headers accordingly.
// Parameters:
// - ctx: The HTTP context for the incoming request (*mist.Context).
func (m *Management[T]) Refresh(ctx *mist.Context) {
	// Check if the refresh options are set.
	if m.refreshJWTOptions == nil {
		// Log error and abort the request if refreshJWTOptions are not set.
		slog.Error("refreshJWTOptions are nil, please use WithRefreshJWTOptions to configure")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Extract the token from the request context.
	tokenStr := m.extractTokenString(ctx)
	clm, err := m.VerifyRefreshToken(tokenStr, jwt.WithTimeFunc(m.nowFunc))
	if err != nil {
		slog.Debug("refresh auth verification failed")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Generate a new access token.
	accessToken, err := m.GenerateAccessToken(clm.Data)
	if err != nil {
		slog.Error("failed to generate access auth")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ctx.Header(m.exposeAccessHeader, accessToken)

	// Optionally generate a new refresh token.
	if m.rotateRefreshToken {
		refreshToken, err := m.GenerateRefreshToken(clm.Data)
		if err != nil {
			slog.Error("failed to generate refresh auth")
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		ctx.Header(m.exposeRefreshHeader, refreshToken)
	}
	ctx.AbortWithStatus(http.StatusNoContent)
}

// MiddlewareBuilder initializes a new MiddlewareBuilder for generating middleware.
// Returns:
// - *MiddlewareBuilder[T]: A pointer to a new MiddlewareBuilder instance.
func (m *Management[T]) MiddlewareBuilder() *MiddlewareBuilder[T] {
	return initMiddlewareBuilder[T](m)
}

// extractTokenString extracts the token string from the request context's header.
// Parameters:
// - ctx: The HTTP context for the incoming request (*mist.Context).
// Returns:
// - string: The extracted token string, or an empty string if not found.
func (m *Management[T]) extractTokenString(ctx *mist.Context) string {
	authCode := ctx.Request.Header.Get(m.allowTokenHeader)
	if authCode == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(bearerPrefix)
	b.WriteString(" ")
	prefix := b.String()
	if strings.HasPrefix(authCode, prefix) {
		return authCode[len(prefix):]
	}
	return ""
}

// GenerateAccessToken generates a new access token containing the specified data.
// Parameters:
// - data: The data to be included in the token (T).
// Returns:
// - string: The generated access token.
// - error: An error if token generation fails.
func (m *Management[T]) GenerateAccessToken(data T) (string, error) {
	nowTime := m.nowFunc()
	claims := RegisteredClaims[T]{
		Data: data,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.accessJWTOptions.Issuer,
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(m.accessJWTOptions.Expire)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			ID:        m.accessJWTOptions.genIDFn(),
		},
	}
	token := jwt.NewWithClaims(m.accessJWTOptions.Method, claims)
	return token.SignedString([]byte(m.accessJWTOptions.EncryptionKey))
}

// VerifyAccessToken verifies the provided access token.
// Parameters:
// - token: The access token to verify (string).
// - opts: Additional parser options for the JWT (...jwt.ParserOption).
// Returns:
// - RegisteredClaims[T]: The claims extracted from the token.
// - error: An error if token verification fails.
func (m *Management[T]) VerifyAccessToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error) {
	t, err := jwt.ParseWithClaims(token, &RegisteredClaims[T]{},
		func(*jwt.Token) (interface{}, error) {
			return []byte(m.accessJWTOptions.DecryptKey), nil
		},
		opts...,
	)
	if err != nil || !t.Valid {
		return RegisteredClaims[T]{}, fmt.Errorf("verification failed: %v", err)
	}
	clm, _ := t.Claims.(*RegisteredClaims[T])
	return *clm, nil
}

// GenerateRefreshToken generates a new refresh token containing the specified data.
// Parameters:
// - data: The data to be included in the refresh token (T).
// Returns:
// - string: The generated refresh token.
// - error: An error if token generation fails or refresh options are not set.
func (m *Management[T]) GenerateRefreshToken(data T) (string, error) {
	// Check if the refresh options are set.
	if m.refreshJWTOptions == nil {
		return "", errEmptyRefreshOpts // Return error if refresh options are not set.
	}

	nowTime := m.nowFunc() // Get the current time using the configured function.

	// Create claims for the refresh token.
	claims := RegisteredClaims[T]{
		Data: data, // Set the data in the claims.
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.refreshJWTOptions.Issuer,                                  // Set the issuer.
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(m.refreshJWTOptions.Expire)), // Set the expiration time.
			IssuedAt:  jwt.NewNumericDate(nowTime),                                 // Set the issued-at time.
			ID:        m.refreshJWTOptions.genIDFn(),                               // Generate and set the token ID.
		},
	}

	// Create a new token with the specified claims and signing method.
	token := jwt.NewWithClaims(m.refreshJWTOptions.Method, claims)

	// Sign the token using the encryption key.
	return token.SignedString([]byte(m.refreshJWTOptions.EncryptionKey))
}

// VerifyRefreshToken verifies the provided refresh token.
// Parameters:
// - token: The refresh token to verify (string).
// - opts: Additional parser options for the JWT (...jwt.ParserOption).
// Returns:
// - RegisteredClaims[T]: The claims extracted from the token.
// - error: An error if token verification fails or refresh options are not set.
func (m *Management[T]) VerifyRefreshToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error) {
	// Check if the refresh options are set.
	if m.refreshJWTOptions == nil {
		return RegisteredClaims[T]{}, errEmptyRefreshOpts // Return error if refresh options are not set.
	}

	// Parse the token with the provided claims structure and a custom key function.
	t, err := jwt.ParseWithClaims(token, &RegisteredClaims[T]{},
		func(*jwt.Token) (interface{}, error) {
			return []byte(m.refreshJWTOptions.DecryptKey), nil // Provide the decryption key.
		},
		opts...,
	)

	// Check if parsing or validation failed.
	if err != nil || !t.Valid {
		return RegisteredClaims[T]{}, fmt.Errorf("verification failed: %v", err) // Return error if verification fails.
	}

	// Extract the claims from the token.
	clm, _ := t.Claims.(*RegisteredClaims[T])
	return *clm, nil // Return the extracted claims.
}

// SetClaims sets the given claims into the HTTP request context.
// Parameters:
// - ctx: The HTTP context for the incoming request (*mist.Context).
// - claims: The claims to set into the context (RegisteredClaims[T]).
func (m *Management[T]) SetClaims(ctx *mist.Context, claims RegisteredClaims[T]) {
	ctx.Set("claims", claims) // Set the claims into the context.
}
