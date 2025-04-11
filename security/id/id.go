package id

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 错误定义
var (
	// ErrInvalidIDLength 表示ID长度无效
	ErrInvalidIDLength = errors.New("无效的ID长度")
	// ErrIDGenerationFailed 表示ID生成失败
	ErrIDGenerationFailed = errors.New("ID生成失败")
	// ErrInvalidNodeID 表示节点ID无效
	ErrInvalidNodeID = errors.New("无效的节点ID")
	// ErrClockMovedBackwards 表示时钟回拨
	ErrClockMovedBackwards = errors.New("时钟回拨，无法生成ID")
)

// 优化：添加熵源池
var entropyPool = sync.Pool{
	New: func() interface{} {
		return &EntropyReader{reader: rand.Reader}
	},
}

// EntropyReader 包装随机源以提高性能
type EntropyReader struct {
	reader io.Reader
	mutex  sync.Mutex
}

// Read 实现io.Reader接口
func (er *EntropyReader) Read(p []byte) (n int, err error) {
	er.mutex.Lock()
	defer er.mutex.Unlock()
	return er.reader.Read(p)
}

// GetEntropyReader 从池中获取熵源
func GetEntropyReader() *EntropyReader {
	return entropyPool.Get().(*EntropyReader)
}

// ReleaseEntropyReader 归还熵源到池
func ReleaseEntropyReader(er *EntropyReader) {
	if er != nil {
		entropyPool.Put(er)
	}
}

// IDType 定义生成ID的类型
type IDType int

const (
	// TypeUUID 使用标准UUID
	TypeUUID IDType = iota
	// TypeULID 使用ULID (Universally Unique Lexicographically Sortable Identifier)
	TypeULID
	// TypeSnowflake 使用雪花算法
	TypeSnowflake
	// TypeToken 使用随机令牌
	TypeToken
	// TypeSecureSequence 使用抗推测序列
	TypeSecureSequence
)

// FormatType 定义ID格式类型
type FormatType int

const (
	// FormatHex 十六进制格式
	FormatHex FormatType = iota
	// FormatBase32 Base32格式
	FormatBase32
	// FormatBase64 Base64格式
	FormatBase64
	// FormatString 字符串格式 (例如标准UUID字符串)
	FormatString
	// FormatInt 数字格式 (用于雪花算法)
	FormatInt
)

// UUIDVersion 定义UUID版本
type UUIDVersion int

const (
	// UUIDv4 完全随机UUID
	UUIDv4 UUIDVersion = 4
	// UUIDv7 有序的UUID（基于时间）
	UUIDv7 UUIDVersion = 7
)

// ULID结构和常量
const (
	// 时间戳位数
	ulidTimeSize = 10
	// 随机位数 - ULID标准规定是16字节，但我们只使用80位(10字节)随机数
	ulidRandomSize = 10 // 80位随机数，总共20字节二进制数据
	// 编码字母表 - 使用Crockford's Base32 (排除I, L, O, U以避免混淆)
	ulidEncoding = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

// ULID表示一个ULID（Universally Unique Lexicographically Sortable Identifier）
// 26个字符，按照lexicographical排序
// ULID是128位标识符，包含48位时间戳(毫秒)和80位随机数，编码为26个字符的字符串
type ULID [ulidTimeSize + ulidRandomSize]byte

// MonotonicEntropy 单调递增的熵源
type MonotonicEntropy struct {
	lastTime time.Time
	entropy  []byte
	mutex    sync.Mutex
}

// NewMonotonicEntropy 创建单调递增的熵源
func NewMonotonicEntropy() *MonotonicEntropy {
	return &MonotonicEntropy{
		entropy: make([]byte, ulidRandomSize),
	}
}

// 生成单调递增的随机数
func (m *MonotonicEntropy) Generate(t time.Time) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	entropy := make([]byte, ulidRandomSize)

	if t.Equal(m.lastTime) {
		// 如果时间相同，递增上次的熵值
		carry := true
		for i := len(m.entropy) - 1; i >= 0 && carry; i-- {
			m.entropy[i]++
			if m.entropy[i] != 0 {
				carry = false
			}
		}
		copy(entropy, m.entropy)
	} else {
		// 如果时间变了，生成新的随机熵
		er := GetEntropyReader()
		defer ReleaseEntropyReader(er)
		_, err := er.Read(entropy)
		if err != nil {
			return nil, err
		}
		copy(m.entropy, entropy)
		m.lastTime = t
	}

	return entropy, nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Type:           TypeUUID,
		Format:         FormatString,
		UUIDVersion:    UUIDv4,
		TokenLength:    32,
		SequencePrefix: "",
		SnowflakeConfig: SnowflakeConfig{
			NodeID:            1,
			NodeBits:          10,
			StepBits:          12,
			TimeUnit:          time.Millisecond,
			EpochStart:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano() / int64(time.Millisecond),
			SequenceBlockSize: 100,
			UseRandom:         true,
		},
	}
}

// Config 配置ID生成器
type Config struct {
	// Type ID类型
	Type IDType
	// Format ID格式
	Format FormatType
	// UUIDVersion UUID版本
	UUIDVersion UUIDVersion
	// TokenLength 令牌长度
	TokenLength int
	// SequencePrefix 序列前缀
	SequencePrefix string
	// SnowflakeConfig 雪花算法配置
	SnowflakeConfig SnowflakeConfig
	// 优化：单调递增选项
	MonotonicULID bool
	// 优化：时钟回拨处理策略
	ClockDriftHandling ClockDriftStrategy
}

// ClockDriftStrategy 时钟回拨处理策略
type ClockDriftStrategy int

const (
	// StrategyError 出错策略 - 直接返回错误
	StrategyError ClockDriftStrategy = iota
	// StrategyWait 等待策略 - 等待时钟赶上
	StrategyWait
	// StrategyTruncate 截断策略 - 使用最后的时间戳
	StrategyTruncate
)

// SnowflakeConfig 雪花算法配置
type SnowflakeConfig struct {
	// NodeID 节点ID
	NodeID int64
	// NodeBits 节点位数
	NodeBits uint8
	// StepBits 序列位数
	StepBits uint8
	// TimeUnit 时间单位
	TimeUnit time.Duration
	// EpochStart 开始时间戳
	EpochStart int64
	// SequenceBlockSize 序列块大小
	SequenceBlockSize int64
	// UseRandom 使用随机数填充序列
	UseRandom bool

	// 内部字段
	sequence      int64
	lastTimestamp int64
	mutex         sync.Mutex
}

// Generator ID生成器接口
type Generator interface {
	// Generate 生成ID
	Generate() (string, error)
	// GenerateBytes 生成字节形式的ID
	GenerateBytes() ([]byte, error)
	// GenerateInt 生成整数形式的ID（仅适用于雪花算法）
	GenerateInt() (int64, error)
}

// 优化：添加对象池
var (
	ulidGeneratorPool = sync.Pool{
		New: func() interface{} {
			return &ULIDGenerator{
				config:           DefaultConfig(),
				monotonicEntropy: NewMonotonicEntropy(),
			}
		},
	}

	snowflakeGeneratorPool = sync.Pool{
		New: func() interface{} {
			return &SnowflakeGenerator{config: DefaultConfig()}
		},
	}
)

// 创建一个基于配置的ID生成器
func NewGenerator(config *Config) Generator {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Type {
	case TypeUUID:
		return &UUIDGenerator{config: config}
	case TypeULID:
		if !config.MonotonicULID {
			return &ULIDGenerator{
				config:           config,
				monotonicEntropy: nil,
			}
		}
		gen := ulidGeneratorPool.Get().(*ULIDGenerator)
		gen.config = config
		if gen.monotonicEntropy == nil {
			gen.monotonicEntropy = NewMonotonicEntropy()
		}
		return gen
	case TypeSnowflake:
		gen := snowflakeGeneratorPool.Get().(*SnowflakeGenerator)
		gen.config = config
		return gen
	case TypeToken:
		return &TokenGenerator{config: config}
	case TypeSecureSequence:
		return &SecureSequenceGenerator{
			config:    config,
			lastValue: 0,
			mutex:     &sync.Mutex{},
		}
	default:
		return &UUIDGenerator{config: config} // 默认使用UUID
	}
}

// UUID生成器
type UUIDGenerator struct {
	config *Config
}

// 生成UUID
func (g *UUIDGenerator) Generate() (string, error) {
	bytes, err := g.GenerateBytes()
	if err != nil {
		return "", err
	}

	switch g.config.Format {
	case FormatString:
		return uuid.UUID(bytes).String(), nil
	case FormatHex:
		return hex.EncodeToString(bytes), nil
	case FormatBase32:
		return strings.TrimRight(base32.StdEncoding.EncodeToString(bytes), "="), nil
	case FormatBase64:
		return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
	default:
		return uuid.UUID(bytes).String(), nil
	}
}

// 生成UUID字节
func (g *UUIDGenerator) GenerateBytes() ([]byte, error) {
	var id uuid.UUID
	var err error

	switch g.config.UUIDVersion {
	case UUIDv4:
		id, err = uuid.NewRandom()
	case UUIDv7:
		// UUID v7 是基于时间的有序UUID
		// 这里是一个简单的UUID v7实现
		now := time.Now()
		timestamp := uint64(now.Unix())
		var randomBytes [10]byte
		_, err = rand.Read(randomBytes[:])
		if err != nil {
			return nil, err
		}

		// 创建ID
		id = uuid.New()
		// 设置时间戳 (头48位)
		binary.BigEndian.PutUint32(id[0:], uint32(timestamp>>16))
		binary.BigEndian.PutUint16(id[4:], uint16(timestamp&0xFFFF))
		// 设置版本为7
		id[6] = (id[6] & 0x0F) | 0x70
		// 设置变体
		id[8] = (id[8] & 0x3F) | 0x80
		// 复制随机字节
		copy(id[6:], randomBytes[:])
	default:
		id, err = uuid.NewRandom()
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
	}

	return id[:], nil
}

// 生成整数形式的ID (UUID不支持，返回错误)
func (g *UUIDGenerator) GenerateInt() (int64, error) {
	return 0, errors.New("UUID不支持整数表示")
}

// ULID生成器
type ULIDGenerator struct {
	config           *Config
	monotonicEntropy *MonotonicEntropy
}

// 释放ULID生成器到对象池
func (g *ULIDGenerator) Release() {
	if g.config.MonotonicULID {
		ulidGeneratorPool.Put(g)
	}
}

// 从时间创建ULID时间部分
func ulidFromTime(t time.Time) (u ULID) {
	ts := uint64(t.UnixMilli())

	// 填充时间戳部分（前10个字节）
	binary.BigEndian.PutUint64(u[:8], ts)
	u[8] = byte(ts >> 40)
	u[9] = byte(ts >> 48)

	return u
}

// 填充ULID随机部分
func (u *ULID) setRandom() error {
	er := GetEntropyReader()
	defer ReleaseEntropyReader(er)

	// 填充随机部分（后16个字节）
	_, err := er.Read(u[ulidTimeSize:])
	return err
}

// 填充ULID随机部分（使用自定义熵源）
func (u *ULID) setRandomWithEntropy(entropy []byte) {
	copy(u[ulidTimeSize:], entropy)
}

// 将ULID转换为字符串
func (u ULID) String() string {
	// 使用Base32编码将ULID转为字符串
	// 时间部分占10字节，编码后是16个字符
	timeChars := encodeBase32(u[:ulidTimeSize])
	// 随机部分占10字节，编码后是16个字符
	entropyChars := encodeBase32(u[ulidTimeSize:])
	// 截取到标准ULID总长26个字符
	return timeChars[:10] + entropyChars
}

// encodeBase32 将字节数组编码为Base32字符串
func encodeBase32(data []byte) string {
	var result strings.Builder
	result.Grow(base32OutputLen(len(data)))

	// 处理每5个字节为一组
	for i := 0; i < len(data); i += 5 {
		var chunk [5]byte
		chunkSize := copy(chunk[:], data[i:])

		// 将5个字节拆分成5个5位的组，并编码每组
		result.WriteByte(ulidEncoding[chunk[0]>>3])
		result.WriteByte(ulidEncoding[(chunk[0]&0x07)<<2|chunk[1]>>6])

		if chunkSize > 1 {
			result.WriteByte(ulidEncoding[(chunk[1]&0x3F)>>1])
			if chunkSize > 2 {
				result.WriteByte(ulidEncoding[(chunk[1]&0x01)<<4|chunk[2]>>4])
				if chunkSize > 3 {
					result.WriteByte(ulidEncoding[(chunk[2]&0x0F)<<1|chunk[3]>>7])
					result.WriteByte(ulidEncoding[(chunk[3]&0x7F)>>2])
					if chunkSize > 4 {
						result.WriteByte(ulidEncoding[(chunk[3]&0x03)<<3|chunk[4]>>5])
						result.WriteByte(ulidEncoding[chunk[4]&0x1F])
					}
				}
			}
		}
	}

	return result.String()
}

// base32OutputLen 计算Base32编码后的字符串长度
func base32OutputLen(n int) int {
	return (n*8 + 4) / 5
}

// 生成ULID
func (g *ULIDGenerator) Generate() (string, error) {
	bytes, err := g.GenerateBytes()
	if err != nil {
		return "", err
	}

	switch g.config.Format {
	case FormatString:
		var u ULID
		copy(u[:], bytes)
		return u.String(), nil
	case FormatHex:
		return hex.EncodeToString(bytes), nil
	case FormatBase32:
		return strings.TrimRight(base32.StdEncoding.EncodeToString(bytes), "="), nil
	case FormatBase64:
		return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
	default:
		var u ULID
		copy(u[:], bytes)
		return u.String(), nil
	}
}

// 生成ULID字节
func (g *ULIDGenerator) GenerateBytes() ([]byte, error) {
	now := time.Now()

	// 从当前时间创建ULID
	u := ulidFromTime(now)

	var err error
	if g.monotonicEntropy != nil {
		// 使用单调递增的熵源
		entropy, err := g.monotonicEntropy.Generate(now)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
		}
		u.setRandomWithEntropy(entropy)
	} else {
		// 使用普通随机填充
		err = u.setRandom()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
		}
	}

	return u[:], nil
}

// 生成整数形式的ID (ULID不支持，返回错误)
func (g *ULIDGenerator) GenerateInt() (int64, error) {
	return 0, errors.New("ULID不支持整数表示")
}

// 雪花算法生成器
type SnowflakeGenerator struct {
	config *Config
}

// 释放雪花算法生成器到对象池
func (g *SnowflakeGenerator) Release() {
	snowflakeGeneratorPool.Put(g)
}

// 生成雪花算法ID
func (g *SnowflakeGenerator) Generate() (string, error) {
	id, err := g.GenerateInt()
	if err != nil {
		return "", err
	}

	switch g.config.Format {
	case FormatString, FormatInt:
		return fmt.Sprintf("%d", id), nil
	case FormatHex:
		return fmt.Sprintf("%x", id), nil
	case FormatBase32:
		bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, uint64(id))
		return strings.TrimRight(base32.StdEncoding.EncodeToString(bytes), "="), nil
	case FormatBase64:
		bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, uint64(id))
		return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
	default:
		return fmt.Sprintf("%d", id), nil
	}
}

// 生成雪花算法ID字节
func (g *SnowflakeGenerator) GenerateBytes() ([]byte, error) {
	id, err := g.GenerateInt()
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, uint64(id))
	return bytes, nil
}

// 生成雪花算法ID整数
func (g *SnowflakeGenerator) GenerateInt() (int64, error) {
	cfg := &g.config.SnowflakeConfig
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()

	// 获取当前时间戳
	now := time.Now().UnixNano() / int64(cfg.TimeUnit)

	// 处理时钟回拨
	if now < cfg.lastTimestamp {
		switch g.config.ClockDriftHandling {
		case StrategyError:
			// 直接返回错误
			return 0, ErrClockMovedBackwards
		case StrategyWait:
			// 等待直到时钟赶上
			for now < cfg.lastTimestamp {
				time.Sleep(time.Duration(cfg.lastTimestamp-now) * cfg.TimeUnit)
				now = time.Now().UnixNano() / int64(cfg.TimeUnit)
			}
		case StrategyTruncate:
			// 使用最后的时间戳
			now = cfg.lastTimestamp
		}
	}

	// 如果在同一时间单位内
	if now == cfg.lastTimestamp {
		// 使用随机数填充序列或者递增序列
		if cfg.UseRandom {
			// 生成随机序列号
			randBits := make([]byte, 8)
			_, err := rand.Read(randBits)
			if err != nil {
				return 0, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
			}
			cfg.sequence = int64(binary.BigEndian.Uint64(randBits) % (1 << cfg.StepBits))
		} else {
			// 递增序列号
			cfg.sequence = (cfg.sequence + 1) & ((1 << cfg.StepBits) - 1)
			// 如果序列号溢出，等待下一个时间单位
			if cfg.sequence == 0 {
				for now <= cfg.lastTimestamp {
					now = time.Now().UnixNano() / int64(cfg.TimeUnit)
				}
			}
		}
	} else {
		// 不同时间戳，重置序列
		if cfg.UseRandom {
			// 使用随机序列
			randBits := make([]byte, 8)
			_, err := rand.Read(randBits)
			if err != nil {
				return 0, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
			}
			cfg.sequence = int64(binary.BigEndian.Uint64(randBits) % (1 << cfg.StepBits))
		} else {
			cfg.sequence = 0
		}
	}

	// 更新最后时间戳
	cfg.lastTimestamp = now

	// 计算时间差值
	timestamp := now - cfg.EpochStart

	// 计算最大位移
	timestampShift := cfg.NodeBits + cfg.StepBits

	// 生成ID
	id := (timestamp << timestampShift) |
		(cfg.NodeID << cfg.StepBits) |
		cfg.sequence

	return id, nil
}

// 令牌生成器
type TokenGenerator struct {
	config *Config
}

// 生成令牌
func (g *TokenGenerator) Generate() (string, error) {
	bytes, err := g.GenerateBytes()
	if err != nil {
		return "", err
	}

	switch g.config.Format {
	case FormatHex:
		return hex.EncodeToString(bytes), nil
	case FormatBase32:
		return strings.TrimRight(base32.StdEncoding.EncodeToString(bytes), "="), nil
	case FormatBase64:
		return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
	case FormatString:
		return hex.EncodeToString(bytes), nil
	default:
		return hex.EncodeToString(bytes), nil
	}
}

// 生成令牌字节
func (g *TokenGenerator) GenerateBytes() ([]byte, error) {
	length := g.config.TokenLength
	if length <= 0 {
		length = 32 // 默认32字节
	}

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
	}

	return bytes, nil
}

// 生成整数形式的ID (令牌不支持，返回错误)
func (g *TokenGenerator) GenerateInt() (int64, error) {
	return 0, errors.New("令牌不支持整数表示")
}

// 安全序列生成器
type SecureSequenceGenerator struct {
	config    *Config
	lastValue int64
	mutex     *sync.Mutex
}

// 生成安全序列
func (g *SecureSequenceGenerator) Generate() (string, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// 增加序列值并添加随机偏移
	randBytes := make([]byte, 4)
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
	}

	// 使用随机值作为增长的步长，防止连续ID被猜测
	randOffset := int64(binary.BigEndian.Uint32(randBytes)%100) + 1
	g.lastValue += randOffset

	// 构建ID
	var id string
	if g.config.SequencePrefix != "" {
		id = fmt.Sprintf("%s%d", g.config.SequencePrefix, g.lastValue)
	} else {
		id = fmt.Sprintf("%d", g.lastValue)
	}

	return id, nil
}

// 生成安全序列字节
func (g *SecureSequenceGenerator) GenerateBytes() ([]byte, error) {
	id, err := g.Generate()
	if err != nil {
		return nil, err
	}

	return []byte(id), nil
}

// 生成整数形式的ID
func (g *SecureSequenceGenerator) GenerateInt() (int64, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// 增加序列值并添加随机偏移
	randBytes := make([]byte, 4)
	_, err := rand.Read(randBytes)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
	}

	// 使用随机值作为增长的步长，防止连续ID被猜测
	randOffset := int64(binary.BigEndian.Uint32(randBytes)%100) + 1
	g.lastValue += randOffset

	return g.lastValue, nil
}

// 工具函数

// 从MAC地址生成节点ID
func NodeIDFromMAC() (int64, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}

	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && len(i.HardwareAddr) >= 6 {
			mac := i.HardwareAddr
			return int64(binary.BigEndian.Uint16(mac) % 1024), nil
		}
	}

	return 0, ErrInvalidNodeID
}

// UUID 生成标准UUID
func UUID() (string, error) {
	config := DefaultConfig()
	config.Type = TypeUUID
	config.UUIDVersion = UUIDv4
	config.Format = FormatString

	generator := NewGenerator(config)
	return generator.Generate()
}

// GenerateULID 生成ULID
func GenerateULID() (string, error) {
	config := DefaultConfig()
	config.Type = TypeULID
	config.Format = FormatString
	config.MonotonicULID = false

	generator := NewGenerator(config)
	defer func() {
		if releaser, ok := generator.(interface{ Release() }); ok {
			releaser.Release()
		}
	}()
	return generator.Generate()
}

// GenerateMonotonicULID 生成单调递增的ULID
func GenerateMonotonicULID() (string, error) {
	config := DefaultConfig()
	config.Type = TypeULID
	config.Format = FormatString
	config.MonotonicULID = true

	generator := NewGenerator(config)
	defer func() {
		if releaser, ok := generator.(interface{ Release() }); ok {
			releaser.Release()
		}
	}()
	return generator.Generate()
}

// Snowflake 生成雪花算法ID
func Snowflake(nodeID int64) (int64, error) {
	if nodeID < 0 || nodeID > 1023 {
		// 尝试从MAC地址生成节点ID
		var err error
		nodeID, err = NodeIDFromMAC()
		if err != nil {
			// 如果无法从MAC生成，使用默认值
			nodeID = 1
		}
	}

	config := DefaultConfig()
	config.Type = TypeSnowflake
	config.Format = FormatInt
	config.SnowflakeConfig.NodeID = nodeID
	config.ClockDriftHandling = StrategyWait // 默认使用等待策略

	generator := NewGenerator(config)
	defer func() {
		if releaser, ok := generator.(interface{ Release() }); ok {
			releaser.Release()
		}
	}()
	return generator.GenerateInt()
}

// SnowflakeWithStrategy 使用指定时钟回拨策略生成雪花ID
func SnowflakeWithStrategy(nodeID int64, strategy ClockDriftStrategy) (int64, error) {
	if nodeID < 0 || nodeID > 1023 {
		var err error
		nodeID, err = NodeIDFromMAC()
		if err != nil {
			nodeID = 1
		}
	}

	config := DefaultConfig()
	config.Type = TypeSnowflake
	config.Format = FormatInt
	config.SnowflakeConfig.NodeID = nodeID
	config.ClockDriftHandling = strategy

	generator := NewGenerator(config)
	defer func() {
		if releaser, ok := generator.(interface{ Release() }); ok {
			releaser.Release()
		}
	}()
	return generator.GenerateInt()
}

// Token 生成随机令牌
func Token(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	config := DefaultConfig()
	config.Type = TypeToken
	config.Format = FormatHex
	config.TokenLength = length

	generator := NewGenerator(config)
	return generator.Generate()
}

// SecureSequence 生成安全序列
func SecureSequence(prefix string) (string, error) {
	config := DefaultConfig()
	config.Type = TypeSecureSequence
	config.SequencePrefix = prefix

	generator := NewGenerator(config)
	return generator.Generate()
}
