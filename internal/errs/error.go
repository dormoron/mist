package errs

import (
	"errors"
	"fmt"
)

var (
	// base
	errInvalidType = errors.New("base: type conversion failed, expected type")
	// web
	errKeyNotFound        = errors.New("session: key not found")
	errSessionNotFound    = errors.New("session: session not found")
	errIdSessionNotFound  = errors.New("session: session corresponding to id does not exist")
	errVerificationFailed = errors.New("session: verification failed")
	errEmptyRefreshOpts   = errors.New("refreshJWTOptions are nil")
	// context error
	errInputNil = errors.New("web: input cannot be nil")
	errBodyNil  = errors.New("web: body is nil")
	errKeyNil   = errors.New("web: key does not exist")
	//  router errors
	errPathNotAllowWildcardAndPath        = errors.New("web: illegal route, path parameter route already exists. Cannot register wildcard route and parameter route at the same time")
	errPathNotAllowPathAndRegular         = errors.New("web: illegal route, path parameter route already exists. Cannot register regular route and parameter route at the same time")
	errRegularNotAllowWildcardAndRegular  = errors.New("web: illegal route, regular route already exists. Cannot register wildcard route and regular route at the same time")
	errRegularNotAllowRegularAndPath      = errors.New("web: illegal route, regular route already exists. Cannot register regular route and parameter route at the same time")
	errWildcardNotAllowWildcardAndPath    = errors.New("web: illegal route, wildcard route already exists. Cannot register wildcard route and parameter route at the same time")
	errWildcardNotAllowWildcardAndRegular = errors.New("web: illegal route, wildcard route already exists. Cannot register wildcard route and regular route at the same time")
	errPathClash                          = errors.New("web: route conflict, parameter routes clash")
	errRegularClash                       = errors.New("web: route conflict, regular routes clash")
	errRegularExpression                  = errors.New("web: regular expression error")
	errRouterNotString                    = errors.New("web: route is an empty string")
	errRouterFront                        = errors.New("web: route must start with '/'")
	errRouterBack                         = errors.New("web: route cannot end with '/'")
	errRouterGroupFront                   = errors.New("web: route group must start with '/'")
	errRouterGroupBack                    = errors.New("web: route group cannot end with '/'")
	errRouterChildConflict                = errors.New("web: Child routes must start with '/'")
	errRouterConflict                     = errors.New("web: route conflict")
	errRouterNotSymbolic                  = errors.New("web: illegal route. Routes like //a/b, /a//b etc. are not allowed")
)

func ErrInvalidType(want string, got any) error {
	return fmt.Errorf("%w :%s, actual value:%#v", errInvalidType, want, got)
}

func ErrKeyNotFound(key string) error {
	return fmt.Errorf("%w, key %s", errKeyNotFound, key)
}

func ErrSessionNotFound() error {
	return fmt.Errorf("%w", errSessionNotFound)
}

func ErrIdSessionNotFound() error {
	return fmt.Errorf("%w", errIdSessionNotFound)
}

func ErrVerificationFailed(err error) error {
	return fmt.Errorf("%w, %w", errVerificationFailed, err)
}

func ErrEmptyRefreshOpts() error {
	return fmt.Errorf("%w", errEmptyRefreshOpts)
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
	return fmt.Errorf("%w: existing parameter route %s, attempting to register new %s", errPathClash, pathParam, path)
}

func ErrRegularClash(pathParam string, path string) error {
	return fmt.Errorf("%w: existing regular route %s, attempting to register new %s", errRegularClash, pathParam, path)
}
func ErrRegularExpression(err error) error {
	return fmt.Errorf("%w %w", errRegularExpression, err)
}

func ErrRouterNotString() error {
	return fmt.Errorf("%w", errRouterNotString)
}

func ErrRouterFront() error {
	return fmt.Errorf("%w", errRouterFront)
}

func ErrRouterBack() error {
	return fmt.Errorf("%w", errRouterBack)
}

func ErrRouterGroupFront() error {
	return fmt.Errorf("%w", errRouterGroupFront)
}

func ErrRouterGroupBack() error {
	return fmt.Errorf("%w", errRouterGroupBack)
}

func ErrRouterChildConflict() error {
	return fmt.Errorf("%w", errRouterChildConflict)
}

func ErrRouterConflict(val string) error {
	return fmt.Errorf("%w [%s]", errRouterConflict, val)
}

func ErrRouterNotSymbolic(path string) error {
	return fmt.Errorf("%w, [%s]", errRouterNotSymbolic, path)
}
