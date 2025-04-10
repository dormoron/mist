package blocklist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// 错误定义
var (
	// ErrStorageOperationFailed 表示存储操作失败
	ErrStorageOperationFailed = errors.New("存储操作失败")
)

// Storage 定义黑名单存储接口
type Storage interface {
	// SaveIPRecord 保存IP记录
	SaveIPRecord(record *IPRecord) error
	// GetIPRecord 获取IP记录
	GetIPRecord(ip string) (*IPRecord, error)
	// DeleteIPRecord 删除IP记录
	DeleteIPRecord(ip string) error
	// ListIPRecords 列出所有IP记录
	ListIPRecords() ([]*IPRecord, error)
	// ListBlockedIPs 列出所有被封禁的IP
	ListBlockedIPs() ([]*IPRecord, error)
	// Close 关闭存储连接
	Close() error
}

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	records map[string]*IPRecord
}

// NewMemoryStorage 创建新的内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		records: make(map[string]*IPRecord),
	}
}

// SaveIPRecord 保存IP记录到内存
func (s *MemoryStorage) SaveIPRecord(record *IPRecord) error {
	s.records[record.IP] = record
	return nil
}

// GetIPRecord 从内存获取IP记录
func (s *MemoryStorage) GetIPRecord(ip string) (*IPRecord, error) {
	record, exists := s.records[ip]
	if !exists {
		return nil, nil
	}
	return record, nil
}

// DeleteIPRecord 从内存删除IP记录
func (s *MemoryStorage) DeleteIPRecord(ip string) error {
	delete(s.records, ip)
	return nil
}

// ListIPRecords 列出所有内存中的IP记录
func (s *MemoryStorage) ListIPRecords() ([]*IPRecord, error) {
	records := make([]*IPRecord, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	return records, nil
}

// ListBlockedIPs 列出所有被封禁的IP
func (s *MemoryStorage) ListBlockedIPs() ([]*IPRecord, error) {
	now := time.Now()
	var blockedRecords []*IPRecord

	for _, record := range s.records {
		if record.BlockedUntil.After(now) {
			blockedRecords = append(blockedRecords, record)
		}
	}
	return blockedRecords, nil
}

// Close 关闭内存存储（无操作）
func (s *MemoryStorage) Close() error {
	return nil
}

// RedisStorage Redis存储实现
type RedisStorage struct {
	client *redis.Client
	prefix string
	ctx    context.Context
}

// NewRedisStorage 创建新的Redis存储
func NewRedisStorage(addr, password string, db int, prefix string) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &RedisStorage{
		client: client,
		prefix: prefix,
		ctx:    ctx,
	}, nil
}

// getKey 生成Redis键
func (s *RedisStorage) getKey(ip string) string {
	return fmt.Sprintf("%s:ip:%s", s.prefix, ip)
}

// getBlocklistKey 生成Redis黑名单集合键
func (s *RedisStorage) getBlocklistKey() string {
	return fmt.Sprintf("%s:blocklist", s.prefix)
}

// SaveIPRecord 保存IP记录到Redis
func (s *RedisStorage) SaveIPRecord(record *IPRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	key := s.getKey(record.IP)
	pipe := s.client.Pipeline()

	// 保存IP记录
	pipe.Set(s.ctx, key, data, 0)

	// 如果IP被封禁，添加到黑名单集合
	now := time.Now()
	if record.BlockedUntil.After(now) {
		// 设置过期时间为封禁结束时间
		expiry := record.BlockedUntil.Sub(now)
		pipe.Set(s.ctx, key, data, expiry)
		// 添加到黑名单集合
		pipe.ZAdd(s.ctx, s.getBlocklistKey(), &redis.Z{
			Score:  float64(record.BlockedUntil.Unix()),
			Member: record.IP,
		})
	} else {
		// 如果不再被封禁，从黑名单集合中移除
		pipe.ZRem(s.ctx, s.getBlocklistKey(), record.IP)
	}

	_, err = pipe.Exec(s.ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageOperationFailed, err)
	}

	return nil
}

// GetIPRecord 从Redis获取IP记录
func (s *RedisStorage) GetIPRecord(ip string) (*IPRecord, error) {
	key := s.getKey(ip)
	data, err := s.client.Get(s.ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // IP记录不存在
		}
		return nil, fmt.Errorf("%w: %v", ErrStorageOperationFailed, err)
	}

	var record IPRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	return &record, nil
}

// DeleteIPRecord 从Redis删除IP记录
func (s *RedisStorage) DeleteIPRecord(ip string) error {
	key := s.getKey(ip)
	pipe := s.client.Pipeline()

	pipe.Del(s.ctx, key)
	pipe.ZRem(s.ctx, s.getBlocklistKey(), ip)

	_, err := pipe.Exec(s.ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageOperationFailed, err)
	}

	return nil
}

// ListIPRecords 列出Redis中的所有IP记录
func (s *RedisStorage) ListIPRecords() ([]*IPRecord, error) {
	pattern := s.getKey("*")
	var cursor uint64
	var keys []string
	var err error
	var records []*IPRecord

	// 使用SCAN迭代所有键
	for {
		keys, cursor, err = s.client.Scan(s.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStorageOperationFailed, err)
		}

		for _, key := range keys {
			data, err := s.client.Get(s.ctx, key).Bytes()
			if err != nil {
				continue // 跳过无法读取的记录
			}

			var record IPRecord
			if err := json.Unmarshal(data, &record); err != nil {
				continue // 跳过无法解析的记录
			}

			records = append(records, &record)
		}

		if cursor == 0 {
			break
		}
	}

	return records, nil
}

// ListBlockedIPs 列出Redis中所有被封禁的IP
func (s *RedisStorage) ListBlockedIPs() ([]*IPRecord, error) {
	now := time.Now()
	blocklistKey := s.getBlocklistKey()

	// 获取所有当前时间仍在封禁中的IP
	ips, err := s.client.ZRangeByScore(s.ctx, blocklistKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", now.Unix()),
		Max: "+inf",
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageOperationFailed, err)
	}

	var records []*IPRecord
	for _, ip := range ips {
		record, err := s.GetIPRecord(ip)
		if err == nil && record != nil {
			records = append(records, record)
		}
	}

	return records, nil
}

// Close 关闭Redis连接
func (s *RedisStorage) Close() error {
	return s.client.Close()
}

// PostgresStorage 实现可添加到此处...
