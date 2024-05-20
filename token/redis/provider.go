package redis

import (
	"errors"
	"github.com/dormoron/mist"
	ijwt "github.com/dormoron/mist/internal/jwt"
	"github.com/dormoron/mist/token"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

// keyRefreshToken is the key used for storing the refresh token in the session.
var keyRefreshToken = "refresh_token"

// Ensure SessionProvider implements the token.Provider interface.
var _ token.Provider = &SessionProvider{}

// SessionProvider is responsible for managing session tokens using a JWT manager and a Redis client.
type SessionProvider struct {
	client      redis.Cmdable              // Provides access to execute commands with the Redis server.
	m           ijwt.Manager[token.Claims] // Manages JWT token generation and validation.
	tokenHeader string                     // The HTTP header key used to retrieve the token from the request.
	atHeader    string                     // The HTTP header key used to send the access token to the client.
	rtHeader    string                     // The HTTP header key used to send the refresh token to the client.
	expiration  time.Duration              // Specifies the duration before the session expires.
}

// UpdateClaims handles updating the session with new JWT claims.
// Parameters:
// - ctx: The request context containing headers and session data.
// - claims: The claims to embed within the generated JWT.
// Returns:
// - error: Any error encountered during access or refresh token generation.
func (rsp *SessionProvider) UpdateClaims(ctx *mist.Context, claims token.Claims) error {
	accessToken, err := rsp.m.GenerateAccessToken(claims) // Generate a new access token.
	if err != nil {
		return err // Return error on failure to generate access token.
	}
	refreshToken, err := rsp.m.GenerateRefreshToken(claims) // Generate a new refresh token.
	if err != nil {
		return err // Return error on failure to generate refresh token.
	}
	ctx.Header(rsp.atHeader, accessToken)  // Set the access token in the response header.
	ctx.Header(rsp.rtHeader, refreshToken) // Set the refresh token in the response header.
	return nil                             // Return nil error on success.
}

// RenewAccessToken handles the renewal of an access token using a refresh token.
// Parameters:
// - ctx: The request context containing session information and headers.
// Returns:
// - error: Any error occurred during refreshing the token, or if the refresh token is expired.
func (rsp *SessionProvider) RenewAccessToken(ctx *mist.Context) error {
	rt := rsp.extractTokenString(ctx)              // Extract the refresh token from the context.
	jwtClaims, err := rsp.m.VerifyRefreshToken(rt) // Verify the refresh token.
	if err != nil {
		return err // Return verification error.
	}
	claims := jwtClaims.Data                                                         // Extract claims from the verified refresh token.
	sess := initRedisSession(claims.SSID, rsp.expiration, rsp.client, claims)        // Initialize a Redis session.
	oldToken := sess.Get(ctx.Request.Context(), keyRefreshToken).StringOrDefault("") // Retrieve the existing refresh token.
	_ = sess.Del(ctx.Request.Context(), keyRefreshToken)                             // Remove the old refresh token from the session.
	if oldToken != rt {
		return errors.New("refresh token has expired") // Check for refresh token expiry.
	}
	accessToken, err := rsp.m.GenerateAccessToken(claims) // Generate a new access token.
	if err != nil {
		return err // Return error on access token generation failure.
	}
	refreshToken, err := rsp.m.GenerateRefreshToken(claims) // Generate a new refresh token.
	if err != nil {
		return err // Return error on refresh token generation failure.
	}
	ctx.Header(rsp.rtHeader, refreshToken)                                // Set the new refresh token in the response header.
	ctx.Header(rsp.atHeader, accessToken)                                 // Set the new access token in the response header.
	return sess.Set(ctx.Request.Context(), keyRefreshToken, refreshToken) // Update the session with the new refresh token.
}

// InitSession initializes a session for a user with provided UID and JWT data.
// Parameters:
// - ctx: The request context including headers and session information.
// - uid: The user identifier for whom the session is to be created.
// - jwtData: JWT claims to be included in the token.
// - sessData: Session-specific data that needs to be persisted.
// Returns:
// - token.Session: A newly created session object or nil on error.
// - error: Any error that occurs during initialization of the session.
func (rsp *SessionProvider) InitSession(ctx *mist.Context,
	uid int64,
	jwtData map[string]string,
	sessData map[string]any) (token.Session, error) {
	ssid := uuid.New().String()                                 // Generate a unique session ID.
	claims := token.Claims{Uid: uid, SSID: ssid, Data: jwtData} // Construct the token claims.
	accessToken, err := rsp.m.GenerateAccessToken(claims)       // Generate an access token.
	if err != nil {
		return nil, err // Return an error on access token generation failure.
	}
	refreshToken, err := rsp.m.GenerateRefreshToken(claims) // Generate a refresh token.
	if err != nil {
		return nil, err // Return an error on refresh token generation failure.
	}
	ctx.Header(rsp.rtHeader, refreshToken)                            // Set the refresh token in the response headers.
	ctx.Header(rsp.atHeader, accessToken)                             // Set the access token in the response headers.
	res := initRedisSession(ssid, rsp.expiration, rsp.client, claims) // Initialize a Redis-based session with the new claims.
	if sessData == nil {
		sessData = make(map[string]any) // If no session data was provided, initialize an empty map.
	}
	sessData["uid"] = uid                           // Add the UID to the session data.
	sessData[keyRefreshToken] = refreshToken        // Add the refresh token to the session data.
	err = res.init(ctx.Request.Context(), sessData) // Store the session data in Redis.
	return res, err                                 // Return the session object and any error occurred.
}

// extractTokenString retrieves the JWT token as a string from the Authorization header in the request.
// Parameters:
// - ctx: The context of the current request incorporating the HTTP response writer for header manipulation.
// Returns:
// - string: The JWT token extracted from the Authorization header, without the 'Bearer ' prefix. Returns an empty string if the prefix is not found or the header is missing.
func (rsp *SessionProvider) extractTokenString(ctx *mist.Context) string {
	authCode := ctx.ResponseWriter.Header().Get(rsp.tokenHeader) // Retrieve the Authorization header value.
	const bearerPrefix = "Bearer "                               // Define the expected prefix for Authorization tokens.
	if strings.HasPrefix(authCode, bearerPrefix) {
		// If authCode starts with bearerPrefix, remove bearerPrefix and return the token string.
		return authCode[len(bearerPrefix):]
	}
	return "" // Return an empty string if the prefix is not present, implying the token is missing or malformed.
}

// Get attempts to retrieve an existing session or establishes a new one based on a verified access token.
// Parameters:
// - ctx: The request context including session data and request/response utilities.
// Returns:
// - token.Session: The existing or newly established session.
// - error: An error if the access token verification fails or session initialization encounters an error.
func (rsp *SessionProvider) Get(ctx *mist.Context) (token.Session, error) {
	val, _ := ctx.Get(token.CtxSessionKey) // Attempt to retrieve a session using the session key.
	res, ok := val.(*Session)
	if ok {
		return res, nil // Return the retrieved session directly if it exists.
	}
	// If no session is found, verify the access token and initiate a new session.
	claims, err := rsp.m.VerifyAccessToken(rsp.extractTokenString(ctx)) // Verify the access token extracted from the request.
	if err != nil {
		return nil, err // Return error if token verification fails.
	}
	// Initialize a new session using the verified claims and return it.
	res = initRedisSession(claims.Data.SSID, rsp.expiration, rsp.client, claims.Data)
	return res, nil
}

// InitSessionProvider initializes a new SessionProvider with specified Redis client and JWT settings.
// Parameters:
// - client: The Redis client used for session data storage.
// - jwtKey: The secret key used to sign and verify JWT tokens.
// Returns:
// - *SessionProvider: A pointer to the newly created SessionProvider instance configured with the provided Redis client and JWT manager.
func InitSessionProvider(client redis.Cmdable, jwtKey string) *SessionProvider {
	expiration := time.Hour * 24 * 30 // Set a default session expiration time of 30 days.
	// Initialize JWT management with options for access and refresh tokens.
	m := ijwt.InitManagement[token.Claims](ijwt.InitOptions(time.Hour, jwtKey),
		ijwt.WithRefreshJWTOptions[token.Claims](ijwt.InitOptions(expiration, jwtKey)))
	return &SessionProvider{
		client:      client,            // The Redis client for session management.
		atHeader:    "X-Access-Token",  // The header key for access tokens.
		rtHeader:    "X-Refresh-Token", // The header key for refresh tokens.
		tokenHeader: "Authorization",   // The header key for HTTP Authorization header, typically bearing JWT.
		m:           m,                 // The JWT manager configured with provided key.
		expiration:  expiration,        // The configured expiration duration for session tokens.
	}
}
