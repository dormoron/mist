package main

import (
	"fmt"
	"sort"

	"github.com/dormoron/mist/security/id"
)

func main() {
	// 生成基本的ULID
	ulid, err := id.GenerateULID()
	if err != nil {
		fmt.Printf("生成ULID出错: %v\n", err)
		return
	}
	fmt.Printf("基本ULID: %s (长度: %d)\n", ulid, len(ulid))

	// 生成多个单调递增的ULID
	var monotonicIds []string
	for i := 0; i < 5; i++ {
		id, err := id.GenerateMonotonicULID()
		if err != nil {
			fmt.Printf("生成单调ULID出错: %v\n", err)
			return
		}
		monotonicIds = append(monotonicIds, id)
	}

	fmt.Println("\n单调递增的ULIDs:")
	for i, mid := range monotonicIds {
		fmt.Printf("  %d: %s\n", i+1, mid)
	}

	// 确认单调递增的特性
	sortedIds := make([]string, len(monotonicIds))
	copy(sortedIds, monotonicIds)
	sort.Strings(sortedIds)

	fmt.Println("\n排序后是否相同:", equal(monotonicIds, sortedIds))

	// 显示时间戳和随机部分
	if len(ulid) >= 26 {
		// ULID的前10字符是时间戳部分
		fmt.Printf("\n时间戳部分: %s\n", ulid[:10])
		// 后16字符是随机部分
		fmt.Printf("随机部分: %s\n", ulid[10:])
	}
}

// equal 比较两个字符串切片是否相等
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
