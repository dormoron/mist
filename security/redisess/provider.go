package redisess

import (
	"errors"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/token"
	"github.com/dormoron/mist/security"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

var (
	keyRefreshToken = "refresh_token"
)

var _ security.Provider = &SessionProvider{}

type SessionProvider struct {
	client      redis.Cmdable
	m           token.Manager[security.Claims]
	tokenHeader string
	atHeader    string
	rtHeader    string
	expiration  time.Duration
}

func (rsp *SessionProvider) UpdateClaims(ctx *mist.Context, claims security.Claims) error {
	accessToken, err := rsp.m.GenerateAccessToken(claims)
	if err != nil {
		return err
	}
	refreshToken, err := rsp.m.GenerateRefreshToken(claims)
	if err != nil {
		return err
	}
	ctx.Header(rsp.atHeader, accessToken)
	ctx.Header(rsp.rtHeader, refreshToken)
	return nil
}

func (rsp *SessionProvider) RenewAccessToken(ctx *mist.Context) error {
	rt := rsp.extractTokenString(ctx)
	jwtClaims, err := rsp.m.VerifyRefreshToken(rt)
	if err != nil {
		return err
	}
	claims := jwtClaims.Data
	sess := newRedisSession(claims.SSID, rsp.expiration, rsp.client, claims)
	oldToken := sess.Get(ctx, keyRefreshToken).StringOrDefault("")
	_ = sess.Del(ctx, keyRefreshToken)
	if oldToken != rt {
		return errors.New("refresh_token has expired")
	}
	accessToken, err := rsp.m.GenerateAccessToken(claims)
	if err != nil {
		return err
	}
	refreshToken, err := rsp.m.GenerateRefreshToken(claims)
	if err != nil {
		return err
	}
	ctx.Header(rsp.rtHeader, refreshToken)
	ctx.Header(rsp.atHeader, accessToken)
	return sess.Set(ctx, keyRefreshToken, refreshToken)
}

func (rsp *SessionProvider) NewSession(ctx *mist.Context,
	uid int64,
	jwtData map[string]string,
	sessData map[string]any) (security.Session, error) {
	ssid := uuid.New().String()
	claims := security.Claims{Uid: uid, SSID: ssid, Data: jwtData}
	accessToken, err := rsp.m.GenerateAccessToken(claims)
	if err != nil {
		return nil, err
	}
	refreshToken, err := rsp.m.GenerateRefreshToken(claims)
	if err != nil {
		return nil, err
	}

	ctx.Header(rsp.rtHeader, refreshToken)
	ctx.Header(rsp.atHeader, accessToken)

	res := newRedisSession(ssid, rsp.expiration, rsp.client, claims)
	if sessData == nil {
		sessData = make(map[string]any, 2)
	}
	sessData["uid"] = uid
	sessData[keyRefreshToken] = refreshToken
	err = res.init(ctx, sessData)
	return res, err
}

func (rsp *SessionProvider) extractTokenString(ctx *mist.Context) string {
	authCode := ctx.Request.Header.Get(rsp.tokenHeader)
	const bearerPrefix = "Bearer "
	if strings.HasPrefix(authCode, bearerPrefix) {
		return authCode[len(bearerPrefix):]
	}
	return ""
}

func (rsp *SessionProvider) Get(ctx *mist.Context) (security.Session, error) {
	val, _ := ctx.Get(security.CtxSessionKey)
	// 对接口断言，而不是对实现断言
	res, ok := val.(security.Session)
	if ok {
		return res, nil
	}

	claims, err := rsp.m.VerifyAccessToken(rsp.extractTokenString(ctx))
	if err != nil {
		return nil, err
	}
	res = newRedisSession(claims.Data.SSID, rsp.expiration, rsp.client, claims.Data)
	return res, nil
}

func NewSessionProvider(client redis.Cmdable, jwtKey string) *SessionProvider {
	expiration := time.Hour * 24 * 30
	m := token.NewManagement[security.Claims](token.NewOptions(time.Hour, jwtKey),
		token.WithRefreshJWTOptions[security.Claims](token.NewOptions(expiration, jwtKey)))
	return &SessionProvider{
		client:      client,
		atHeader:    "X-Access-Token",
		rtHeader:    "X-Refresh-Token",
		tokenHeader: "Authorization",
		m:           m,
		expiration:  expiration,
	}
}
