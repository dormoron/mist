package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// 错误定义
var (
	// ErrInvalidHash 表示哈希格式无效
	ErrInvalidHash = errors.New("提供的密码哈希格式无效")
	// ErrIncompatibleVersion 表示哈希使用了不支持的版本
	ErrIncompatibleVersion = errors.New("不兼容的哈希版本")
	// ErrMismatchedHashAndPassword 表示密码与存储的哈希不匹配
	ErrMismatchedHashAndPassword = errors.New("密码与哈希不匹配")
)

// Params 结构定义了用于Argon2id哈希算法的参数
type Params struct {
	// Memory 内存使用量(KB)
	Memory uint32
	// Iterations 迭代次数
	Iterations uint32
	// Parallelism 并行度
	Parallelism uint8
	// SaltLength 盐长度(字节)
	SaltLength uint32
	// KeyLength 输出密钥长度(字节)
	KeyLength uint32
}

// DefaultParams 返回默认的Argon2id参数
// 这些默认参数基于OWASP和Argon2id的建议
func DefaultParams() *Params {
	return &Params{
		Memory:      64 * 1024, // 64MB
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// GenerateFromPassword 使用Argon2id算法从密码生成加密哈希
// 返回的哈希格式为：$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
// 注意：该函数将创建一个新的随机盐用于每次哈希生成
func GenerateFromPassword(password []byte, params *Params) (string, error) {
	// 生成随机盐
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// 使用Argon2id计算哈希
	hash := argon2.IDKey(
		password,
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// 使用base64编码盐和哈希值
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// 创建标准格式的密码哈希
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// CompareHashAndPassword 比较密码与其哈希值
// 成功时返回nil，失败时返回错误
func CompareHashAndPassword(encodedHash string, password []byte) error {
	// 解析哈希串以获取参数、盐和哈希
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return err
	}

	// 使用相同的参数和盐重新计算哈希
	otherHash := argon2.IDKey(
		password,
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// 使用常量时间比较避免计时攻击
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return nil
	}

	return ErrMismatchedHashAndPassword
}

// NeedsRehash 检查给定的哈希是否使用过期的参数，需要重新哈希
func NeedsRehash(encodedHash string, params *Params) (bool, error) {
	// 解析已有的哈希参数
	oldParams, _, _, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	// 检查参数是否匹配
	return oldParams.Memory != params.Memory ||
		oldParams.Iterations != params.Iterations ||
		oldParams.Parallelism != params.Parallelism ||
		oldParams.KeyLength != params.KeyLength, nil
}

// 内部函数：解码哈希串
func decodeHash(encodedHash string) (*Params, []byte, []byte, error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &Params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	params.SaltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}

// HashPassword 是对GenerateFromPassword的简单封装，使用默认参数
func HashPassword(password string) (string, error) {
	return GenerateFromPassword([]byte(password), DefaultParams())
}

// CheckPassword 是对CompareHashAndPassword的简单封装
func CheckPassword(password, hash string) error {
	return CompareHashAndPassword(hash, []byte(password))
}

// PasswordStrength 表示密码强度级别
type PasswordStrength int

const (
	// VeryWeak 表示非常弱的密码
	VeryWeak PasswordStrength = iota
	// Weak 表示弱密码
	Weak
	// Medium 表示中等强度密码
	Medium
	// Strong 表示强密码
	Strong
	// VeryStrong 表示非常强的密码
	VeryStrong
)

// CheckPasswordStrength 检查密码强度
func CheckPasswordStrength(password string) PasswordStrength {
	var score int

	// 长度检查
	length := len(password)
	if length < 8 {
		score += 1
	} else if length < 12 {
		score += 2
	} else if length < 16 {
		score += 3
	} else {
		score += 4
	}

	// 检查是否包含数字
	if containsDigit(password) {
		score += 1
	}

	// 检查是否包含小写字母
	if containsLower(password) {
		score += 1
	}

	// 检查是否包含大写字母
	if containsUpper(password) {
		score += 1
	}

	// 检查是否包含特殊字符
	if containsSpecial(password) {
		score += 1
	}

	// 根据总分返回密码强度
	switch {
	case score <= 2:
		return VeryWeak
	case score <= 4:
		return Weak
	case score <= 6:
		return Medium
	case score <= 7:
		return Strong
	default:
		return VeryStrong
	}
}

// 辅助函数：检查是否包含数字
func containsDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

// 辅助函数：检查是否包含小写字母
func containsLower(s string) bool {
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			return true
		}
	}
	return false
}

// 辅助函数：检查是否包含大写字母
func containsUpper(s string) bool {
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

// 辅助函数：检查是否包含特殊字符
func containsSpecial(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return true
		}
	}
	return false
}

// GetPasswordStrengthDescription 获取密码强度描述
func GetPasswordStrengthDescription(strength PasswordStrength) string {
	switch strength {
	case VeryWeak:
		return "非常弱：密码太简单，容易被破解"
	case Weak:
		return "弱：密码强度不足，建议增加复杂度"
	case Medium:
		return "中等：密码强度一般，可以使用但建议增强"
	case Strong:
		return "强：密码强度良好"
	case VeryStrong:
		return "非常强：密码强度极佳"
	default:
		return "未知强度"
	}
}
