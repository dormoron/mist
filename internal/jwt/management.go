package jwt

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/errs"
	"github.com/dormoron/mist/kit"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// bearerPrefix is a constant that defines the standard prefix used in authorization headers for bearer tokens.
const bearerPrefix = "Bearer"

// Management holds configurations for JWT authentication and refresh tokens, with support for generic data types.
//
// Fields:
// - allowTokenHeader: Name of the HTTP header to check for the access token.
// - exposeAccessHeader: Name of the HTTP header used to expose the access token in the response.
// - exposeRefreshHeader: Name of the HTTP header used to expose the refresh token in the response.
// - accessJWTOptions: Configuration for the JWT access tokens.
// - refreshJWTOptions: Optional configuration for the JWT refresh tokens (may be nil if not used).
// - rotateRefreshToken: Flag indicating whether to issue a new refresh token when refreshing an access token.
// - nowFunc: Function that returns the current time, used for setting token issuance and expiration timestamps.
//
// Type Parameter:
// - T: Represents the general type of the data included in the JWT claims.
type Management[T any] struct {
	allowTokenHeader    string
	exposeAccessHeader  string
	exposeRefreshHeader string

	accessJWTOptions   Options
	refreshJWTOptions  *Options
	rotateRefreshToken bool
	nowFunc            func() time.Time
}

// InitManagement initializes a Management structure with specified JWT options and additional optional configurations.
//
// Parameters:
// - accessJWTOptions: A set of options used to create access JWTs.
// - opts: A variadic parameter that can include additional options to customize the Management structure.
//
// Returns:
// - *Management[T]: A pointer to the newly initialized Management structure parameterized by T.
func InitManagement[T any](accessJWTOptions Options, opts ...kit.Option[Management[T]]) *Management[T] {
	dOpts := defaultManagementOptions[T]()
	dOpts.accessJWTOptions = accessJWTOptions
	kit.Apply[Management[T]](&dOpts, opts...)

	return &dOpts
}

// defaultManagementOptions creates default settings for the Management structure.
//
// Returns:
// - Management[T]: An instance of Management parameterized by T with default configuration.
func defaultManagementOptions[T any]() Management[T] {
	return Management[T]{
		allowTokenHeader:    "authorization",
		exposeAccessHeader:  "x-access-token",
		exposeRefreshHeader: "x-refresh-token",
		rotateRefreshToken:  false,
		nowFunc:             time.Now,
	}
}

// The following With* functions are option-setting functions used to create functional options for flexible configurations of the Management structure.

// WithAllowTokenHeader sets the name of the HTTP header from which the access token will be retrieved.
//
// Parameters:
// - header: The name of the HTTP header.
//
// Returns:
// - kit.Option[Management[T]]: A functional option to set the allowTokenHeader field in the Management structure.
func WithAllowTokenHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.allowTokenHeader = header
	}
}

// WithExposeAccessHeader sets the name of the HTTP header that will expose the access token in the response.
//
// Parameters:
// - header: The name of the HTTP header.
//
// Returns:
// - kit.Option[Management[T]]: A functional option to set the exposeAccessHeader field in the Management structure.
func WithExposeAccessHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.exposeAccessHeader = header
	}
}

// WithExposeRefreshHeader sets the name of the HTTP header that will expose the refresh token in the response.
//
// Parameters:
// - header: The name of the HTTP header.
//
// Returns:
// - kit.Option[Management[T]]: A functional option to set the exposeRefreshHeader field in the Management structure.
func WithExposeRefreshHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.exposeRefreshHeader = header
	}
}

// WithRefreshJWTOptions sets the configurations for creating refresh JWTs.
//
// Parameters:
// - refreshOpts: A set of options used to configure refresh JWTs.
//
// Returns:
// - kit.Option[Management[T]]: A functional option to assign the refreshJWTOptions field in the Management structure.
func WithRefreshJWTOptions[T any](refreshOpts Options) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.refreshJWTOptions = &refreshOpts
	}
}

// WithRotateRefreshToken determines whether a new refresh token should be generated when refreshing an access token.
//
// Parameters:
// - isRotate: A boolean flag indicating if refresh token rotation should occur.
//
// Returns:
// - kit.Option[Management[T]]: A functional option to set the rotateRefreshToken field in the Management structure.
func WithRotateRefreshToken[T any](isRotate bool) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.rotateRefreshToken = isRotate
	}
}

// WithNowFunc customizes the function used to obtain the current time, useful for time-related operations like token expiry.
//
// Parameters:
// - nowFunc: A function that returns the current time.
//
// Returns:
// - kit.Option[Management[T]]: A functional option to set the nowFunc field in the Management structure.
func WithNowFunc[T any](nowFunc func() time.Time) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.nowFunc = nowFunc
	}
}

// Refresh handles the token refresh mechanism within the given request context.
// If the refresh token configurations are not set, an internal server error response is returned.
//
// Parameters:
// - ctx: The request context that contains the HTTP request and response details.
func (m *Management[T]) Refresh(ctx *mist.Context) {
	if m.refreshJWTOptions == nil {
		slog.Error("RefreshJWTOptions is nil. Use WithRefreshJWTOptions to set refresh-related configuration.")
		ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokenStr := m.extractTokenString(ctx)
	clm, err := m.VerifyRefreshToken(tokenStr,
		jwt.WithTimeFunc(m.nowFunc))
	if err != nil {
		slog.Debug("refresh token verification failed")
		ctx.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		return
	}
	accessToken, err := m.GenerateAccessToken(clm.Data)
	if err != nil {
		slog.Error("failed to generate access token")
		ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	ctx.Header(m.exposeAccessHeader, accessToken)

	if m.rotateRefreshToken {
		refreshToken, err := m.GenerateRefreshToken(clm.Data)
		if err != nil {
			slog.Error("failed to generate refresh token")
			ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx.Header(m.exposeRefreshHeader, refreshToken)
	}
	ctx.ResponseWriter.WriteHeader(http.StatusNoContent)
}

// MiddlewareBuilder builds and returns a new instance of MiddlewareBuilder,
// which is used to create middleware based on the Management[T] configuration.
//
// Returns:
// - *MiddlewareBuilder[T]: An instance of MiddlewareBuilder that can be used to create middleware.
func (m *Management[T]) MiddlewareBuilder() *MiddlewareBuilder[T] {
	return initMiddlewareBuilder[T](m)
}

// extractTokenString retrieves the token string from the request headers based on the configured header name.
//
// Parameters:
// - ctx: The context of the request containing HTTP header information.
//
// Returns:
// - string: The JWT token without the 'Bearer ' prefix; or an empty string if no valid token is found.
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

// GenerateAccessToken creates a new JSON Web Token (JWT) as an access token for the provided data.
//
// Parameters:
// - data: The payload or claims to be embedded within the access token.
//
// Returns:
// - string: The newly generated JWT access token.
// - Error: Error returned in case of failure in token generation.
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

// VerifyAccessToken verifies the given access token string and returns the associated claims if the token is valid.
//
// Parameters:
// - token: The JWT token to be verified.
// - opts: Parser options to provide additional conditions for token validation.
//
// Returns:
// - RegisteredClaims[T]: The claims extracted from the validated token.
// - error: Error returned if the token is invalid or the verification process fails.
func (m *Management[T]) VerifyAccessToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error) {
	t, err := jwt.ParseWithClaims(token, &RegisteredClaims[T]{},
		func(*jwt.Token) (interface{}, error) {
			return []byte(m.accessJWTOptions.DecryptKey), nil
		},
		opts...,
	)
	if err != nil || !t.Valid {
		return RegisteredClaims[T]{}, errs.ErrVerificationFailed(err)
	}
	clm, _ := t.Claims.(*RegisteredClaims[T])
	return *clm, nil
}

// GenerateRefreshToken creates a new refresh token for the supplied data.
//
// Parameters:
// - data: The payload or specific data for which the refresh token is to be generated.
//
// Returns:
// - string: The newly created refresh token.
// - Error: Error returned in case of failure in refresh token generation.
func (m *Management[T]) GenerateRefreshToken(data T) (string, error) {
	if m.refreshJWTOptions == nil {
		return "", errs.ErrEmptyRefreshOpts()
	}

	nowTime := m.nowFunc()
	claims := RegisteredClaims[T]{
		Data: data,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.refreshJWTOptions.Issuer,
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(m.refreshJWTOptions.Expire)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			ID:        m.refreshJWTOptions.genIDFn(),
		},
	}

	token := jwt.NewWithClaims(m.refreshJWTOptions.Method, claims)
	return token.SignedString([]byte(m.refreshJWTOptions.EncryptionKey))
}

// VerifyRefreshToken checks the validity of the given refresh token and extracts its claims.
//
// Parameters:
// - token: The refresh token to be validated.
// - opts: Additional parser options for the verification process.
//
// Returns:
// - RegisteredClaims[T]: The registered claims present in the refresh token.
// - error: Error returned if verification fails or the refresh token is invalid.
func (m *Management[T]) VerifyRefreshToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error) {
	if m.refreshJWTOptions == nil {
		return RegisteredClaims[T]{}, errs.ErrEmptyRefreshOpts()
	}
	t, err := jwt.ParseWithClaims(token, &RegisteredClaims[T]{},
		func(*jwt.Token) (interface{}, error) {
			return []byte(m.refreshJWTOptions.DecryptKey), nil
		},
		opts...,
	)
	if err != nil || !t.Valid {
		return RegisteredClaims[T]{}, errs.ErrVerificationFailed(err)
	}
	clm, _ := t.Claims.(*RegisteredClaims[T])
	return *clm, nil
}

// SetClaims is a helper function that stores the claims in the context of the request.
//
// Parameters:
// - ctx: The context of the request where the claims should be stored.
// - claims: The claims to be stored in the request context for further processing in the security flow.
func (m *Management[T]) SetClaims(ctx *mist.Context, claims RegisteredClaims[T]) {
	ctx.Set("claims", claims)
}
