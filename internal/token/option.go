package token

import (
	"github.com/dormoron/mist/internal/token/kit"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Options struct {
	Expire        time.Duration
	EncryptionKey string
	DecryptKey    string
	Method        jwt.SigningMethod
	Issuer        string
	genIDFn       func() string
}

func NewOptions(expire time.Duration, encryptionKey string,
	opts ...kit.Option[Options]) Options {
	dOpts := Options{
		Expire:        expire,
		EncryptionKey: encryptionKey,
		DecryptKey:    encryptionKey,
		Method:        jwt.SigningMethodHS256,
		genIDFn:       func() string { return "" },
	}

	kit.Apply[Options](&dOpts, opts...)

	return dOpts
}

func WithDecryptKey(decryptKey string) kit.Option[Options] {
	return func(o *Options) {
		o.DecryptKey = decryptKey
	}
}

func WithMethod(method jwt.SigningMethod) kit.Option[Options] {
	return func(o *Options) {
		o.Method = method
	}
}

func WithIssuer(issuer string) kit.Option[Options] {
	return func(o *Options) {
		o.Issuer = issuer
	}
}

func WithGenIDFunc(fn func() string) kit.Option[Options] {
	return func(o *Options) {
		o.genIDFn = fn
	}
}
