package security

import "github.com/dormoron/mist"

type Builder struct {
	ctx      *mist.Context
	uid      int64
	jwtData  map[string]string
	sessData map[string]any
	sp       Provider
}

func NewSessionBuilder(ctx *mist.Context, uid int64) *Builder {
	return &Builder{
		ctx: ctx,
		uid: uid,
		sp:  defaultProvider,
	}
}

func (b *Builder) SetProvider(p Provider) *Builder {
	b.sp = p
	return b
}

func (b *Builder) SetJwtData(data map[string]string) *Builder {
	b.jwtData = data
	return b
}

func (b *Builder) SetSessData(data map[string]any) *Builder {
	b.sessData = data
	return b
}

func (b *Builder) Build() (Session, error) {
	return b.sp.NewSession(b.ctx, b.uid, b.jwtData, b.sessData)
}
