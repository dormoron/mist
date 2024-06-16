package redisess

import (
	"context"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security"
	"github.com/redis/go-redis/v9"
	"time"
)

var _ security.Session = &Session{}

type Session struct {
	client     redis.Cmdable
	key        string
	claims     security.Claims
	expiration time.Duration
}

func (sess *Session) Destroy(ctx context.Context) error {
	return sess.client.Del(ctx, sess.key).Err()
}

func (sess *Session) Del(ctx context.Context, key string) error {
	return sess.client.Del(ctx, sess.key, key).Err()
}

func (sess *Session) Set(ctx context.Context, key string, val any) error {
	return sess.client.HSet(ctx, sess.key, key, val).Err()
}

func (sess *Session) init(ctx context.Context, kvs map[string]any) error {
	pip := sess.client.Pipeline()
	for k, v := range kvs {
		pip.HMSet(ctx, sess.key, k, v)
	}
	pip.Expire(ctx, sess.key, sess.expiration)
	_, err := pip.Exec(ctx)
	return err
}

func (sess *Session) Get(ctx context.Context, key string) mist.AnyValue {
	res, err := sess.client.HGet(ctx, sess.key, key).Result()
	if err != nil {
		return mist.AnyValue{Err: err}
	}
	return mist.AnyValue{
		Val: res,
	}
}

func (sess *Session) Claims() security.Claims {
	return sess.claims
}

func newRedisSession(
	ssid string,
	expiration time.Duration,
	client redis.Cmdable, cl security.Claims) *Session {
	return &Session{
		client:     client,
		key:        "session:" + ssid,
		expiration: expiration,
		claims:     cl,
	}
}
