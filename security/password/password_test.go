package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试默认参数
func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	// 验证默认参数
	assert.Equal(t, uint32(64*1024), params.Memory, "默认内存应为64MB")
	assert.Equal(t, uint32(3), params.Iterations, "默认迭代次数应为3")
	assert.Equal(t, uint8(4), params.Parallelism, "默认并行度应为4")
	assert.Equal(t, uint32(16), params.SaltLength, "默认盐长度应为16字节")
	assert.Equal(t, uint32(32), params.KeyLength, "默认密钥长度应为32字节")
}

// 测试密码哈希生成和验证
func TestPasswordHashingAndVerification(t *testing.T) {
	// 测试用例
	testCases := []struct {
		name     string
		password string
	}{
		{"简单密码", "password123"},
		{"复杂密码", "P@ssw0rd!ComplexPassword123"},
		{"包含特殊字符", "!@#$%^&*()_+=-[]{}|;':,./<>?"},
		{"中文密码", "密码123测试"},
		{"长密码", strings.Repeat("abcdefgh", 8)}, // 64个字符
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 使用默认参数生成哈希
			hash, err := HashPassword(tc.password)
			require.NoError(t, err, "哈希生成应该成功")
			assert.NotEmpty(t, hash, "哈希不应为空")

			// 验证哈希格式
			assert.True(t, strings.HasPrefix(hash, "$argon2id$"), "应以$argon2id$开头")
			parts := strings.Split(hash, "$")
			require.Equal(t, 6, len(parts), "哈希应该有6个部分")

			// 验证正确密码
			err = CheckPassword(tc.password, hash)
			assert.NoError(t, err, "正确密码验证应通过")

			// 验证错误密码
			err = CheckPassword(tc.password+"wrong", hash)
			assert.Error(t, err, "错误密码验证应失败")
			assert.Equal(t, ErrMismatchedHashAndPassword, err, "错误应为密码不匹配")
		})
	}
}

// 测试GenerateFromPassword和CompareHashAndPassword函数
func TestGenerateAndCompare(t *testing.T) {
	password := []byte("secure_password_for_testing")
	params := &Params{
		Memory:      32 * 1024,
		Iterations:  2,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}

	// 生成哈希
	hash, err := GenerateFromPassword(password, params)
	require.NoError(t, err, "哈希生成应该成功")

	// 验证哈希包含参数
	assert.Contains(t, hash, "m=32768", "哈希应包含正确的内存参数")
	assert.Contains(t, hash, "t=2", "哈希应包含正确的迭代次数")
	assert.Contains(t, hash, "p=2", "哈希应包含正确的并行度")

	// 验证密码
	err = CompareHashAndPassword(hash, password)
	assert.NoError(t, err, "验证应该成功")

	// 验证错误密码
	wrongPassword := []byte("wrong_password")
	err = CompareHashAndPassword(hash, wrongPassword)
	assert.Error(t, err, "错误密码验证应失败")
}

// 测试NeedsRehash函数
func TestNeedsRehash(t *testing.T) {
	password := []byte("test_password")

	// 使用参数集1生成哈希
	params1 := &Params{
		Memory:      32 * 1024,
		Iterations:  2,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
	hash, err := GenerateFromPassword(password, params1)
	require.NoError(t, err, "哈希生成应该成功")

	// 使用相同参数检查是否需要重新哈希
	needsRehash, err := NeedsRehash(hash, params1)
	require.NoError(t, err, "检查是否需要重新哈希应该成功")
	assert.False(t, needsRehash, "相同参数不应需要重新哈希")

	// 使用不同参数检查是否需要重新哈希
	params2 := &Params{
		Memory:      64 * 1024, // 内存增加
		Iterations:  3,         // 迭代次数增加
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
	needsRehash, err = NeedsRehash(hash, params2)
	require.NoError(t, err, "检查是否需要重新哈希应该成功")
	assert.True(t, needsRehash, "不同参数应需要重新哈希")
}

// 测试无效哈希处理
func TestInvalidHash(t *testing.T) {
	// 测试格式错误的哈希
	invalidHash := "invalid-hash-format"
	err := CheckPassword("password", invalidHash)
	assert.Error(t, err, "无效哈希应返回错误")

	// 测试格式正确但内容错误的哈希
	badFormatHash := "$argon2id$v=19$m=65536,t=3,p=4$invalidSalt$invalidHash"
	err = CheckPassword("password", badFormatHash)
	assert.Error(t, err, "格式正确但内容错误的哈希应返回错误")

	// 测试不兼容版本
	incompatibleHash := "$argon2id$v=18$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g"
	_, _, _, err = decodeHash(incompatibleHash)
	assert.Error(t, err, "不兼容版本应返回错误")
}

// 测试密码强度检查
func TestCheckPasswordStrength(t *testing.T) {
	testCases := []struct {
		password string
		expected PasswordStrength
	}{
		{"123", VeryWeak},                   // 短且只有数字
		{"password", Weak},                  // 只有小写字母
		{"Password1", Medium},               // 包含大小写字母和数字
		{"P@ssword1", Medium},               // 包含大小写字母、数字和特殊字符，总分为6，为Medium
		{"P@ssw0rd!ComplexABC", VeryStrong}, // 长且复杂
	}

	for _, tc := range testCases {
		t.Run(tc.password, func(t *testing.T) {
			strength := CheckPasswordStrength(tc.password)
			assert.Equal(t, tc.expected, strength, "密码强度检查结果应匹配预期")
		})
	}
}

// 测试密码强度描述
func TestGetPasswordStrengthDescription(t *testing.T) {
	testCases := []struct {
		strength PasswordStrength
		expected string
	}{
		{VeryWeak, "非常弱：密码太简单，容易被破解"},
		{Weak, "弱：密码强度不足，建议增加复杂度"},
		{Medium, "中等：密码强度一般，可以使用但建议增强"},
		{Strong, "强：密码强度良好"},
		{VeryStrong, "非常强：密码强度极佳"},
		{PasswordStrength(99), "未知强度"}, // 测试未知强度
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			desc := GetPasswordStrengthDescription(tc.strength)
			assert.Equal(t, tc.expected, desc, "密码强度描述应匹配预期")
		})
	}
}

// 测试辅助函数
func TestHelperFunctions(t *testing.T) {
	// 测试containsDigit
	assert.True(t, containsDigit("abc123"), "应检测到数字")
	assert.False(t, containsDigit("abcdef"), "不应检测到数字")

	// 测试containsLower
	assert.True(t, containsLower("ABCdef"), "应检测到小写字母")
	assert.False(t, containsLower("ABC123"), "不应检测到小写字母")

	// 测试containsUpper
	assert.True(t, containsUpper("abcDEF"), "应检测到大写字母")
	assert.False(t, containsUpper("abc123"), "不应检测到大写字母")

	// 测试containsSpecial
	assert.True(t, containsSpecial("abc!@#"), "应检测到特殊字符")
	assert.False(t, containsSpecial("abc123"), "不应检测到特殊字符")
}
