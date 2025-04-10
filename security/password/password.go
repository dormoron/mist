package password

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
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
	// ErrPasswordTooWeak 表示密码强度不满足要求
	ErrPasswordTooWeak = errors.New("密码强度不满足要求")
	// ErrPasswordInHistory 表示密码在历史记录中已存在
	ErrPasswordInHistory = errors.New("密码不能与最近使用的密码相同")
	// ErrPasswordContainsPersonalInfo 表示密码包含个人信息
	ErrPasswordContainsPersonalInfo = errors.New("密码不能包含个人信息")
	// ErrPasswordInDictionary 表示密码在常见密码字典中
	ErrPasswordInDictionary = errors.New("密码过于常见，请选择一个更独特的密码")
)

// 常用密码列表（示例，实际应用中应该使用更完整的列表）
var commonPasswords = map[string]bool{
	"password":  true,
	"123456":    true,
	"qwerty":    true,
	"admin":     true,
	"welcome":   true,
	"login":     true,
	"abc123":    true,
	"iloveyou":  true,
	"password1": true,
	"12345678":  true,
}

// 键盘邻近字符组
var keyboardPatterns = []string{
	"qwerty", "asdfgh", "zxcvbn", "1234567890",
	"qwertyuiop", "asdfghjkl", "zxcvbnm",
}

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

// PasswordPolicy 定义密码策略
type PasswordPolicy struct {
	// MinLength 最小长度
	MinLength int
	// RequireUppercase 是否要求大写字母
	RequireUppercase bool
	// RequireLowercase 是否要求小写字母
	RequireLowercase bool
	// RequireDigits 是否要求数字
	RequireDigits bool
	// RequireSpecialChars 是否要求特殊字符
	RequireSpecialChars bool
	// MinimumStrength 最低强度要求
	MinimumStrength PasswordStrength
	// DisallowCommonPasswords 是否禁止常见密码
	DisallowCommonPasswords bool
	// CheckKeyboardPatterns 是否检查键盘规律
	CheckKeyboardPatterns bool
	// MaxHistoryCount 最大历史记录数量
	MaxHistoryCount int
}

// DefaultPasswordPolicy 返回默认密码策略
func DefaultPasswordPolicy() *PasswordPolicy {
	return &PasswordPolicy{
		MinLength:               8,
		RequireUppercase:        true,
		RequireLowercase:        true,
		RequireDigits:           true,
		RequireSpecialChars:     true,
		MinimumStrength:         Medium,
		DisallowCommonPasswords: true,
		CheckKeyboardPatterns:   true,
		MaxHistoryCount:         5,
	}
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

	// 检查重复字符
	if repeatedChars(password) > 3 {
		score -= 1
	}

	// 检查字符串的熵（简化版）
	if uniqueCharsRatio(password) < 0.5 {
		score -= 1
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

// 检查重复字符的数量
func repeatedChars(s string) int {
	if len(s) == 0 {
		return 0
	}

	counts := make(map[rune]int)
	for _, c := range s {
		counts[c]++
	}

	var repeated int
	for _, count := range counts {
		if count > 1 {
			repeated += count - 1
		}
	}
	return repeated
}

// 计算唯一字符比例
func uniqueCharsRatio(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	chars := make(map[rune]bool)
	for _, c := range s {
		chars[c] = true
	}

	return float64(len(chars)) / float64(len(s))
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
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return true
		}
	}
	return false
}

// GetPasswordStrengthDescription 获取密码强度的描述
func GetPasswordStrengthDescription(strength PasswordStrength) string {
	switch strength {
	case VeryWeak:
		return "非常弱"
	case Weak:
		return "弱"
	case Medium:
		return "中等"
	case Strong:
		return "强"
	case VeryStrong:
		return "非常强"
	default:
		return "未知"
	}
}

// ValidatePasswordPolicy 验证密码是否符合策略
func ValidatePasswordPolicy(password string, policy *PasswordPolicy, personalInfo []string, passwordHistory []string) error {
	// 基本长度检查
	if len(password) < policy.MinLength {
		return fmt.Errorf("密码长度必须至少为 %d 个字符", policy.MinLength)
	}

	// 字符类型检查
	if policy.RequireUppercase && !containsUpper(password) {
		return errors.New("密码必须包含至少一个大写字母")
	}
	if policy.RequireLowercase && !containsLower(password) {
		return errors.New("密码必须包含至少一个小写字母")
	}
	if policy.RequireDigits && !containsDigit(password) {
		return errors.New("密码必须包含至少一个数字")
	}
	if policy.RequireSpecialChars && !containsSpecial(password) {
		return errors.New("密码必须包含至少一个特殊字符")
	}

	// 密码强度检查
	if CheckPasswordStrength(password) < policy.MinimumStrength {
		return ErrPasswordTooWeak
	}

	// 常见密码检查
	if policy.DisallowCommonPasswords {
		pwdLower := strings.ToLower(password)
		if commonPasswords[pwdLower] {
			return ErrPasswordInDictionary
		}
	}

	// 键盘规律检查
	if policy.CheckKeyboardPatterns {
		pwdLower := strings.ToLower(password)
		for _, pattern := range keyboardPatterns {
			if strings.Contains(pwdLower, pattern) {
				return errors.New("密码包含键盘连续字符，这容易被猜测")
			}
		}
	}

	// 个人信息检查
	if len(personalInfo) > 0 {
		pwdLower := strings.ToLower(password)
		for _, info := range personalInfo {
			if info != "" && len(info) > 3 && strings.Contains(pwdLower, strings.ToLower(info)) {
				return ErrPasswordContainsPersonalInfo
			}
		}
	}

	// 历史密码检查
	if len(passwordHistory) > 0 && policy.MaxHistoryCount > 0 {
		for i := 0; i < len(passwordHistory) && i < policy.MaxHistoryCount; i++ {
			if err := CompareHashAndPassword(passwordHistory[i], []byte(password)); err == nil {
				return ErrPasswordInHistory
			}
		}
	}

	return nil
}

// PasswordHistory 管理密码历史记录
type PasswordHistory struct {
	// hashes 存储已哈希的密码
	hashes []string
	// maxCount 最大历史记录数
	maxCount int
}

// NewPasswordHistory 创建新的密码历史记录管理器
func NewPasswordHistory(maxCount int) *PasswordHistory {
	if maxCount <= 0 {
		maxCount = 5 // 默认保存最近5个密码
	}
	return &PasswordHistory{
		hashes:   make([]string, 0, maxCount),
		maxCount: maxCount,
	}
}

// Add 添加新的密码哈希到历史记录
func (ph *PasswordHistory) Add(hash string) {
	// 将新哈希添加到列表开头
	ph.hashes = append([]string{hash}, ph.hashes...)

	// 如果超过最大数量，删除最旧的
	if len(ph.hashes) > ph.maxCount {
		ph.hashes = ph.hashes[:ph.maxCount]
	}
}

// Contains 检查密码是否在历史记录中
func (ph *PasswordHistory) Contains(password string) bool {
	for _, hash := range ph.hashes {
		if err := CompareHashAndPassword(hash, []byte(password)); err == nil {
			return true
		}
	}
	return false
}

// GetHashes 获取所有哈希值
func (ph *PasswordHistory) GetHashes() []string {
	// 返回一个副本而不是直接引用，防止外部修改
	result := make([]string, len(ph.hashes))
	copy(result, ph.hashes)
	return result
}

// HashIdentifier 创建密码标识符，用于快速比较密码是否相同
// 注意：此标识符不用于验证，仅用于判断密码是否已使用过
func HashIdentifier(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}

// IsPasswordReused 检查新密码是否重复使用了旧密码中的一大部分
func IsPasswordReused(oldPassword, newPassword string, threshold float64) bool {
	// 如果新旧密码长度差异太大，认为不是重复使用
	oldLen := len(oldPassword)
	newLen := len(newPassword)
	if oldLen == 0 || newLen == 0 || float64(newLen)/float64(oldLen) < 0.5 || float64(newLen)/float64(oldLen) > 2.0 {
		return false
	}

	// 检查两个密码中有多少字符相同
	oldChars := make(map[rune]int)
	for _, c := range oldPassword {
		oldChars[c]++
	}

	var matchCount int
	for _, c := range newPassword {
		if count, exists := oldChars[c]; exists && count > 0 {
			matchCount++
			oldChars[c]--
		}
	}

	// 计算相似度
	similarity := float64(matchCount) / float64(oldLen)
	return similarity >= threshold
}

// 检查密码是否有连续重复的字符
func hasRepeatedSequence(password string, minLength int) bool {
	if len(password) < minLength*2 {
		return false
	}

	// 使用正则表达式查找重复序列
	for i := minLength; i <= len(password)/2; i++ {
		pattern := fmt.Sprintf(`(.{%d}).*\1`, i)
		matched, err := regexp.MatchString(pattern, password)
		if err == nil && matched {
			return true
		}
	}

	return false
}
