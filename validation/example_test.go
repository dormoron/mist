package validation

import (
	"fmt"
	"testing"
)

// 简单验证示例
func TestBasicValidation(t *testing.T) {
	v := NewValidator()

	// 验证字符串
	v.Required("", "username")               // 应该失败
	v.Required("user123", "email")           // 应该通过
	v.Email("invalid-email", "email")        // 应该失败
	v.Email("user@example.com", "email")     // 应该通过
	v.MinLength("abc", 5, "password")        // 应该失败
	v.MaxLength("abcdefghijklmn", 10, "bio") // 应该失败

	// 验证数字
	v.Range(15, 18, 60, "age") // 应该失败
	v.Range(25, 18, 60, "age") // 应该通过

	// 自定义验证
	v.Custom(false, "agreement", "必须同意条款") // 应该失败

	if v.Valid() {
		t.Error("期望验证失败，但验证通过了")
	}

	// 输出错误信息
	fmt.Println("基础验证错误:")
	for _, err := range v.Errors {
		fmt.Printf("- %s: %s\n", err.Field, err.Message)
	}
}

// 用于测试的用户结构体
type User struct {
	Username string   `validate:"required,min=3,max=20,alphanum"`
	Email    string   `validate:"required,email"`
	Password string   `validate:"required,min=8"`
	Age      int      `validate:"min=18,max=120"`
	Phone    string   `validate:"phone"`
	Website  string   `validate:"url"`
	Tags     []string `validate:"min=1,max=5"`
	IsActive bool     `validate:"required"`
	Balance  float64  `validate:"min=0"`
}

// 结构体验证示例
func TestStructValidation(t *testing.T) {
	// 创建一个无效的用户
	invalidUser := User{
		Username: "u",             // 太短
		Email:    "invalid-email", // 无效邮箱
		Password: "short",         // 太短
		Age:      16,              // 太小
		Phone:    "12345",         // 无效手机号
		Website:  "invalid-url",   // 无效URL
		Tags:     []string{},      // 空数组
		IsActive: false,           // 需要为true
		Balance:  -100,            // 负数余额
	}

	v := NewValidator()
	ValidateStruct(v, &invalidUser)

	if v.Valid() {
		t.Error("期望结构体验证失败，但验证通过了")
	}

	// 输出错误信息
	fmt.Println("\n结构体验证错误:")
	for _, err := range v.Errors {
		fmt.Printf("- %s: %s\n", err.Field, err.Message)
	}

	// 创建一个有效的用户
	validUser := User{
		Username: "user123",
		Email:    "user@example.com",
		Password: "securepassword",
		Age:      25,
		Phone:    "13812345678",
		Website:  "https://example.com",
		Tags:     []string{"go", "coding"},
		IsActive: true,
		Balance:  1000.50,
	}

	v = NewValidator()
	ValidateStruct(v, &validUser)

	if !v.Valid() {
		t.Error("期望结构体验证通过，但验证失败了")
		for _, err := range v.Errors {
			fmt.Printf("- %s: %s\n", err.Field, err.Message)
		}
	}
}

// 示例：如何在HTTP处理程序中使用
func ExampleValidator_http() {
	// 这是一个示例HTTP处理程序
	/*
		func RegisterHandler(w http.ResponseWriter, r *http.Request) {
			var user User

			// 解析请求JSON数据到结构体
			if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
				http.Error(w, "无效的请求数据", http.StatusBadRequest)
				return
			}

			// 验证用户数据
			v := validation.NewValidator()
			validation.ValidateStruct(v, &user)

			if !v.Valid() {
				// 返回验证错误
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)

				// 构建错误响应
				errorResponse := make(map[string]string)
				for _, err := range v.Errors {
					errorResponse[err.Field] = err.Message
				}

				json.NewEncoder(w).Encode(map[string]any{
					"errors": errorResponse,
				})
				return
			}

			// 继续处理有效请求...
		}
	*/
}
