package redis

import (
	"context"
	"fmt"
	"github.com/dormoron/mist/internal/errs"
	"github.com/dormoron/mist/session"
	"github.com/redis/go-redis/v9"
	"time"
)

type Store struct {
	prefix     string
	client     redis.Cmdable
	expiration time.Duration
}

type StoreOptions func(store *Store)

func InitStore(client redis.Cmdable, opts ...StoreOptions) *Store {
	res := &Store{
		client:     client,
		expiration: time.Minute * 15,
		prefix:     "sessionId",
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func StoreWithExpiration(expiration time.Duration) StoreOptions {
	return func(store *Store) {
		store.expiration = expiration
	}
}

func StoreWithPrefix(prefix string) StoreOptions {
	return func(store *Store) {
		store.prefix = prefix
	}
}

func (s *Store) Generate(ctx context.Context, id string) (session.Session, error) {
	key := redisKey(s.prefix, id)
	_, err := s.client.HSet(ctx, key, id, id).Result()
	if err != nil {
		return nil, err
	}
	_, err = s.client.Expire(ctx, key, s.expiration).Result()
	if err != nil {
		return nil, err
	}
	return &Session{
		id:     id,
		key:    key,
		client: s.client,
	}, nil
}

func (s *Store) Refresh(ctx context.Context, id string) error {
	key := redisKey(s.prefix, id)
	ok, err := s.client.Expire(ctx, key, s.expiration).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errs.ErrIdSessionNotFound()
	}
	return nil
}

func (s *Store) Remove(ctx context.Context, id string) error {
	key := redisKey(s.prefix, id)
	_, err := s.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	return err
}

func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {

	key := redisKey(s.prefix, id)
	cnt, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if cnt != 1 {
		return nil, errs.ErrIdSessionNotFound()
	}
	return &Session{
		id:     id,
		key:    key,
		client: s.client,
	}, nil
}

type Session struct {
	id     string
	key    string
	client redis.Cmdable
}

func (s *Session) Get(ctx context.Context, key string) (any, error) {
	val, err := s.client.HGet(ctx, s.key, key).Result()
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (s *Session) Set(ctx context.Context, key string, value any) error {
	const lua = `
if redis.call("exists", KEYS[1])
then
	return redis.call("hset", KEYS[1], ARGV[1], ARGV[2])
else
	return -1
end
`
	res, err := s.client.Eval(ctx, lua, []string{s.key}, key, value).Int()
	if err != nil {
		return err
	}
	if res < 0 {
		return errs.ErrIdSessionNotFound()
	}
	return nil
}

func (s *Session) ID() string {
	return s.id
}

func redisKey(prefix, id string) string {
	return fmt.Sprintf("%s-%s", prefix, id)
}
