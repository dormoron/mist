package errs

import (
	"errors"
	"fmt"
)

var (
	errKeyNotFound       = errors.New("session: 找不到 key")
	errSessionNotFound   = errors.New("session: 找不到 session")
	errIdSessionNotFound = errors.New("session: id 对应的 session 不存在")
	// context error
	errInputNil = errors.New("web: 输入不能为 nil")
	errBodyNil  = errors.New("web: body 为 nil")
	errKeyNil   = errors.New("web: key 不存在")
	//  router errors
	errPathNotAllowWildcardAndPath        = errors.New("web: 非法路由，已有路径参数路由。不允许同时注册通配符路由和参数路由")
	errPathNotAllowPathAndRegular         = errors.New("web: 非法路由，已有路径参数路由。不允许同时注册正则路由和参数路由")
	errRegularNotAllowWildcardAndRegular  = errors.New("web: 非法路由，已有正则路由。不允许同时注册通配符路由和正则路由")
	errRegularNotAllowRegularAndPath      = errors.New("web: 非法路由，已有正则路由。不允许同时注册正则路由和参数路由")
	errWildcardNotAllowWildcardAndPath    = errors.New("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由")
	errWildcardNotAllowWildcardAndRegular = errors.New("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和正则路由")
	errPathClash                          = errors.New("web: 路由冲突，参数路由冲突")
	errRegularClash                       = errors.New("web: 路由冲突，正则路由冲突")
	errRegularExpression                  = errors.New("web: 正则表达式错误")
)

func ErrKeyNotFound(key string) error {
	return fmt.Errorf("%w, key %s", errKeyNotFound, key)
}
func ErrSessionNotFound() error {
	return fmt.Errorf("%w", errSessionNotFound)
}

func ErrIdSessionNotFound() error {
	return fmt.Errorf("%w", errIdSessionNotFound)
}

func ErrInputNil() error {
	return fmt.Errorf("%w", errInputNil)
}

func ErrBodyNil() error {
	return fmt.Errorf("%w", errBodyNil)
}

func ErrKeyNil() error {
	return fmt.Errorf("%w", errKeyNil)
}

func ErrPathNotAllowWildcardAndPath(path string) error {
	return fmt.Errorf("%w [%s]", errPathNotAllowWildcardAndPath, path)
}

func ErrPathNotAllowPathAndRegular(path string) error {
	return fmt.Errorf("%w [%s]", errPathNotAllowPathAndRegular, path)
}

func ErrRegularNotAllowWildcardAndRegular(path string) error {
	return fmt.Errorf("%w [%s]", errRegularNotAllowWildcardAndRegular, path)
}

func ErrRegularNotAllowRegularAndPath(path string) error {
	return fmt.Errorf("%w [%s]", errRegularNotAllowRegularAndPath, path)
}

func ErrWildcardNotAllowWildcardAndPath(path string) error {
	return fmt.Errorf("%w [%s]", errWildcardNotAllowWildcardAndPath, path)
}

func ErrWildcardNotAllowWildcardAndRegular(path string) error {
	return fmt.Errorf("%w [%s]", errWildcardNotAllowWildcardAndRegular, path)
}

func ErrPathClash(pathParam string, path string) error {
	return fmt.Errorf("%w，已有 %s，新注册 %s", errPathClash, pathParam, path)
}

func ErrRegularClash(pathParam string, path string) error {
	return fmt.Errorf("%w，已有 %s，新注册 %s", errRegularClash, pathParam, path)
}

func ErrRegularExpression(err error) error {
	return fmt.Errorf("%w %w", errRegularExpression, err)
}
