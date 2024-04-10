package cookie

import "net/http"

type PropagatorOptions func(p *Propagator)

type Propagator struct {
	cookieName   string
	cookieOption func(cookie *http.Cookie)
}

func InitPropagator() *Propagator {
	return &Propagator{
		cookieName: "sessionId",
		cookieOption: func(cookie *http.Cookie) {

		},
	}
}

func WithCookieName(name string) PropagatorOptions {
	return func(p *Propagator) {
		p.cookieName = name
	}
}

func (p *Propagator) Inject(id string, writer http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:  p.cookieName,
		Value: id,
	}
	p.cookieOption(cookie)
	http.SetCookie(writer, cookie)
	return nil
}

func (p *Propagator) Extract(req *http.Request) (string, error) {
	cookie, err := req.Cookie(p.cookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (p *Propagator) Remove(writer http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:   p.cookieName,
		MaxAge: -1,
	}
	p.cookieOption(cookie)
	http.SetCookie(writer, cookie)
	return nil
}
