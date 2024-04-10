package mist

import (
	"encoding/json"
	"github.com/dormoron/mist/internal/errs"
	"net/http"
	"net/url"
	"strconv"
)

type Context struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	PathParams     map[string]string
	queryValues    url.Values
	MatchedRoute   string
	RespData       []byte
	RespStatusCode int
	templateEngine TemplateEngine

	UserValues map[string]any
}

func (c *Context) Render(templateName string, data any) error {
	var err error
	c.RespData, err = c.templateEngine.Render(c.Request.Context(), templateName, data)
	if err != nil {
		c.RespStatusCode = http.StatusInternalServerError
		return err
	}
	c.RespStatusCode = http.StatusOK
	return nil
}

func (c *Context) SetCookie(ck *http.Cookie) {
	http.SetCookie(c.ResponseWriter, ck)
}

func (c *Context) RespJSONOK(val any) error {
	return c.RespJSON(http.StatusOK, val)
}

func (c *Context) RespJSON(status int, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	c.ResponseWriter.WriteHeader(status)
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(len(data)))
	c.RespData = data
	c.RespStatusCode = status
	return err
}

func (c *Context) BindJSON(val any) error {
	if val == nil {
		return errs.ErrInputNil()
	}
	if c.Request.Body == nil {
		return errs.ErrBodyNil()
	}
	decoder := json.NewDecoder(c.Request.Body)
	return decoder.Decode(val)
}

func (c *Context) BindJSONOpt(val any, useNumber bool, disableUnknown bool) error {
	if val == nil {
		return errs.ErrInputNil()
	}
	if c.Request.Body == nil {
		return errs.ErrBodyNil()
	}
	decoder := json.NewDecoder(c.Request.Body)
	if useNumber {
		decoder.UseNumber()
	}
	if disableUnknown {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(val)
}

func (c *Context) FormValue(key string) StringValue {
	err := c.Request.ParseForm()
	if err != nil {
		return StringValue{
			val: "",
			err: err,
		}
	}
	return StringValue{val: c.Request.FormValue(key)}
}

func (c *Context) QueryValue(key string) StringValue {
	if c.queryValues == nil {
		c.queryValues = c.Request.URL.Query()
	}

	vals, ok := c.queryValues[key]
	if !ok {
		return StringValue{
			val: "",
			err: errs.ErrKeyNil(),
		}
	}
	return StringValue{val: vals[0]}
}

func (c *Context) PathValue(key string) StringValue {
	val, ok := c.PathParams[key]
	if !ok {
		return StringValue{
			val: "",
			err: errs.ErrKeyNil(),
		}
	}
	return StringValue{val: val}
}

type StringValue struct {
	val string
	err error
}

func (s *StringValue) AsInt64() (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return strconv.ParseInt(s.val, 10, 64)
}

func (s *StringValue) AsUint64() (uint64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return strconv.ParseUint(s.val, 10, 64)
}

func (s *StringValue) AsFloat64() (float64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return strconv.ParseFloat(s.val, 64)
}
