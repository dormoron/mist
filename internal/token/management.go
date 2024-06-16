package token

import (
	"errors"
	"fmt"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/token/kit"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const bearerPrefix = "Bearer"

var (
	errEmptyRefreshOpts = errors.New("refreshJWTOptions are nil")
)

type Management[T any] struct {
	allowTokenHeader    string
	exposeAccessHeader  string
	exposeRefreshHeader string

	accessJWTOptions   Options
	refreshJWTOptions  *Options
	rotateRefreshToken bool
	nowFunc            func() time.Time
}

func NewManagement[T any](accessJWTOptions Options,
	opts ...kit.Option[Management[T]]) *Management[T] {
	dOpts := defaultManagementOptions[T]()
	dOpts.accessJWTOptions = accessJWTOptions
	kit.Apply[Management[T]](&dOpts, opts...)

	return &dOpts
}

func defaultManagementOptions[T any]() Management[T] {
	return Management[T]{
		allowTokenHeader:    "authorization",
		exposeAccessHeader:  "x-access-token",
		exposeRefreshHeader: "x-refresh-token",
		rotateRefreshToken:  false,
		nowFunc:             time.Now,
	}
}

func WithAllowTokenHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.allowTokenHeader = header
	}
}

func WithExposeAccessHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.exposeAccessHeader = header
	}
}

func WithExposeRefreshHeader[T any](header string) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.exposeRefreshHeader = header
	}
}

func WithRefreshJWTOptions[T any](refreshOpts Options) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.refreshJWTOptions = &refreshOpts
	}
}

func WithRotateRefreshToken[T any](isRotate bool) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.rotateRefreshToken = isRotate
	}
}

func WithNowFunc[T any](nowFunc func() time.Time) kit.Option[Management[T]] {
	return func(m *Management[T]) {
		m.nowFunc = nowFunc
	}
}

func (m *Management[T]) Refresh(ctx *mist.Context) {
	if m.refreshJWTOptions == nil {
		slog.Error("refreshJWTOptions 为 nil, 请使用 WithRefreshJWTOptions 设置 refresh 相关的配置")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tokenStr := m.extractTokenString(ctx)
	clm, err := m.VerifyRefreshToken(tokenStr,
		jwt.WithTimeFunc(m.nowFunc))
	if err != nil {
		slog.Debug("refresh token verification failed")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	accessToken, err := m.GenerateAccessToken(clm.Data)
	if err != nil {
		slog.Error("failed to generate access token")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ctx.Header(m.exposeAccessHeader, accessToken)

	if m.rotateRefreshToken {
		refreshToken, err := m.GenerateRefreshToken(clm.Data)
		if err != nil {
			slog.Error("failed to generate refresh token")
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		ctx.Header(m.exposeRefreshHeader, refreshToken)
	}
	ctx.AbortWithStatus(http.StatusNoContent)
}

func (m *Management[T]) MiddlewareBuilder() *MiddlewareBuilder[T] {
	return newMiddlewareBuilder[T](m)
}

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

func (m *Management[T]) VerifyAccessToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error) {
	t, err := jwt.ParseWithClaims(token, &RegisteredClaims[T]{},
		func(*jwt.Token) (interface{}, error) {
			return []byte(m.accessJWTOptions.DecryptKey), nil
		},
		opts...,
	)
	if err != nil || !t.Valid {
		return RegisteredClaims[T]{}, fmt.Errorf("验证失败: %v", err)
	}
	clm, _ := t.Claims.(*RegisteredClaims[T])
	return *clm, nil
}

func (m *Management[T]) GenerateRefreshToken(data T) (string, error) {
	if m.refreshJWTOptions == nil {
		return "", errEmptyRefreshOpts
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

func (m *Management[T]) VerifyRefreshToken(token string, opts ...jwt.ParserOption) (RegisteredClaims[T], error) {
	if m.refreshJWTOptions == nil {
		return RegisteredClaims[T]{}, errEmptyRefreshOpts
	}
	t, err := jwt.ParseWithClaims(token, &RegisteredClaims[T]{},
		func(*jwt.Token) (interface{}, error) {
			return []byte(m.refreshJWTOptions.DecryptKey), nil
		},
		opts...,
	)
	if err != nil || !t.Valid {
		return RegisteredClaims[T]{}, fmt.Errorf("验证失败: %v", err)
	}
	clm, _ := t.Claims.(*RegisteredClaims[T])
	return *clm, nil
}

func (m *Management[T]) SetClaims(ctx *mist.Context, claims RegisteredClaims[T]) {
	ctx.Set("claims", claims)
}
