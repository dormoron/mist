package redisess

import (
	"errors"
	"strings"
	"time"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security"
	"github.com/dormoron/mist/security/auth"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// keyRefreshToken is a constant string key used for storing the refresh token in the session.
var keyRefreshToken = "refresh_token"

// Ensure that SessionProvider implements the security.Provider interface.
var _ security.Provider = &SessionProvider{}

// SessionProvider is a struct that manages session creation, token renewal, and claims update using Redis as backend storage.
type SessionProvider struct {
	client      redis.Cmdable                 // Redis client used to interact with the Redis server.
	m           auth.Manager[security.Claims] // Authentication manager to handle token generation and verification.
	tokenHeader string                        // Header for token extraction.
	atHeader    string                        // Header for storing Access Token.
	rtHeader    string                        // Header for storing Refresh Token.
	expiration  time.Duration                 // The expiration duration of the session.
}

// ClearToken is a method of the SessionProvider that clears the access and refresh tokens for a session.
// This method will validate the refresh token from the request, and if valid, will remove the session
// associated with the token from Redis and clear the token headers on the response.
//
// Parameters:
//   - ctx: The mist.Context object representing the current HTTP request and response.
//
// Returns:
//   - An error object if any step fails, otherwise it returns nil.
func (rsp *SessionProvider) ClearToken(ctx *mist.Context) error {
	// Extract the refresh token from the Authorization header in the request.
	rt := rsp.extractTokenString(ctx)

	// Verify the refresh token using the token manager.
	// If the token is invalid or expired, an error is returned.
	jwtClaims, err := rsp.m.VerifyRefreshToken(rt)
	if err != nil {
		return err
	}

	// Retrieve the claims data which includes user information and session ID.
	claims := jwtClaims.Data

	// Initialize a new Redis session using the session ID and expiration time from the claims.
	sess := initRedisSession(claims.SessionID, rsp.expiration, rsp.client, claims)

	// Retrieve the stored refresh token from the Redis session for comparison.
	storedToken := sess.Get(ctx, keyRefreshToken).StringOrDefault("")
	// If the stored token does not match the provided token, return an error indicating an invalid or expired refresh token.
	if storedToken != rt {
		return errors.New("invalid or expired refresh token")
	}

	// Destroy the session in Redis, effectively clearing the stored refresh token.
	err = sess.Destroy(ctx)
	if err != nil {
		return err
	}

	// Clear the access token and refresh token headers in the HTTP response.
	ctx.Header(rsp.atHeader, "")
	ctx.Header(rsp.rtHeader, "")

	// Return nil to indicate the operation was successful.
	return nil
}

// UpdateClaims updates the JWT claims and sets new access and refresh tokens in the response headers.
// Parameters:
// - ctx: The context for the request (*mist.Context).
// - claims: The claims to be updated (security.Claims).
// Returns:
// - error: An error if the update fails.
func (rsp *SessionProvider) UpdateClaims(ctx *mist.Context, claims security.Claims) error {
	// Generate a new access token.
	accessToken, err := rsp.m.GenerateAccessToken(claims)
	if err != nil {
		return err
	}

	// Generate a new refresh token.
	refreshToken, err := rsp.m.GenerateRefreshToken(claims)
	if err != nil {
		return err
	}

	// Set the new tokens in the response headers.
	ctx.Header(rsp.atHeader, accessToken)
	ctx.Header(rsp.rtHeader, refreshToken)
	return nil
}

// RenewAccessToken renews the access token using the existing refresh token stored in Redis.
// Parameters:
// - ctx: The context for the request (*mist.Context).
// Returns:
// - error: An error if the renewal fails.
func (rsp *SessionProvider) RenewAccessToken(ctx *mist.Context) error {
	rt := rsp.extractTokenString(ctx)              // Extract the refresh token from the request.
	jwtClaims, err := rsp.m.VerifyRefreshToken(rt) // Verify the refresh token.
	if err != nil {
		return err
	}

	claims := jwtClaims.Data
	sess := initRedisSession(claims.SessionID, rsp.expiration, rsp.client, claims)
	oldToken := sess.Get(ctx, keyRefreshToken).StringOrDefault("")
	_ = sess.Del(ctx, keyRefreshToken) // Delete the old refresh token.
	if oldToken != rt {
		return errors.New("refresh_token has expired") // Check if the old refresh token matches the current refresh token.
	}

	// Generate a new access token.
	accessToken, err := rsp.m.GenerateAccessToken(claims)
	if err != nil {
		return err
	}

	// Generate a new refresh token.
	refreshToken, err := rsp.m.GenerateRefreshToken(claims)
	if err != nil {
		return err
	}

	// Set the new tokens in the response headers.
	ctx.Header(rsp.rtHeader, refreshToken)
	ctx.Header(rsp.atHeader, accessToken)

	// Set the new refresh token in the session.
	return sess.Set(ctx, keyRefreshToken, refreshToken)
}

// InitSession initializes a new session and sets initial JWT claims.
// Parameters:
// - ctx: The context for the request (*mist.Context).
// - userId: The user ID for the session (int64).
// - jwtData: JWT-related data to be included in the session (map[string]any).
// - sessData: Additional session data (map[string]any).
// Returns:
// - security.Session: The newly created session.
// - error: An error if the session creation fails.
func (rsp *SessionProvider) InitSession(ctx *mist.Context,
	userId int64,
	jwtData map[string]any,
	sessData map[string]any) (security.Session, error) {

	sessionId := uuid.New().String() // Generate a unique session ID.
	claims := security.Claims{UserID: userId, SessionID: sessionId, Data: jwtData}

	// Generate a new access token.
	accessToken, err := rsp.m.GenerateAccessToken(claims)
	if err != nil {
		return nil, err
	}

	// Generate a new refresh token.
	refreshToken, err := rsp.m.GenerateRefreshToken(claims)
	if err != nil {
		return nil, err
	}

	// Set the new tokens in the response headers.
	ctx.Header(rsp.rtHeader, refreshToken)
	ctx.Header(rsp.atHeader, accessToken)

	// Initialize the session with the new session ID, expiration time, Redis client, and claims.
	res := initRedisSession(sessionId, rsp.expiration, rsp.client, claims)
	if sessData == nil {
		sessData = make(map[string]any, 2)
	}
	sessData["uId"] = userId
	sessData[keyRefreshToken] = refreshToken

	// Initialize the session data in Redis.
	err = res.init(ctx, sessData)
	return res, err
}

// extractTokenString extracts the token string from the request header.
// Parameters:
// - ctx: The context for the request (*mist.Context).
// Returns:
// - string: The extracted token string.
func (rsp *SessionProvider) extractTokenString(ctx *mist.Context) string {
	authCode := ctx.Request.Header.Get(rsp.tokenHeader) // Get the token from the request header.
	const bearerPrefix = "Bearer "
	if strings.HasPrefix(authCode, bearerPrefix) { // Check if the token has the Bearer prefix.
		return authCode[len(bearerPrefix):]
	}
	return ""
}

// Get retrieves the session associated with the given context or verifies the access token
// and returns a session based on the verified claims.
// Parameters:
// - ctx: The context for the request (*mist.Context).
// Returns:
// - security.Session: The session associated with the context.
// - error: An error if the session retrieval fails.
func (rsp *SessionProvider) Get(ctx *mist.Context) (security.Session, error) {
	val, _ := ctx.Get(security.CtxSessionKey)
	res, ok := val.(security.Session)
	if ok {
		return res, nil
	}

	// Verify the access token and extract the claims.
	claims, err := rsp.m.VerifyAccessToken(rsp.extractTokenString(ctx))
	if err != nil {
		return nil, err
	}

	// Initialize a Redis session with the extracted claims.
	res = initRedisSession(claims.Data.SessionID, rsp.expiration, rsp.client, claims.Data)
	return res, nil
}

// InitSessionProvider initializes and returns a new instance of SessionProvider with the given Redis client and JWT key.
// Parameters:
// - client: The Redis client used to interact with the Redis server (redis.Cmdable).
// - jwtKey: The key used to sign JWT tokens (string).
// Returns:
// - *SessionProvider: A pointer to a newly created SessionProvider instance.
func InitSessionProvider(client redis.Cmdable, jwtKey string) *SessionProvider {
	expiration := time.Hour * 24 * 30 // Set the refresh token expiration time (30 days).

	// Initialize the authentication manager with access token and refresh token expiration times.
	m := auth.InitManagement[security.Claims](auth.InitOptions(time.Hour, jwtKey),
		auth.WithRefreshJWTOptions[security.Claims](auth.InitOptions(expiration, jwtKey)))

	return &SessionProvider{
		client:      client,
		atHeader:    "X-Access-Token",
		rtHeader:    "X-Refresh-Token",
		tokenHeader: "Authorization",
		m:           m,
		expiration:  expiration,
	}
}
