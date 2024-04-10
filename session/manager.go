package session

import (
	"github.com/dormoron/mist"
	"github.com/google/uuid"
)

type Manager struct {
	Store
	Propagator
	CtxSessionKey string
}

func (m *Manager) GetSession(ctx *mist.Context) (Session, error) {
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any, 1)
	}
	val, ok := ctx.UserValues[m.CtxSessionKey]
	if ok {
		return val.(Session), nil
	}
	sessId, err := m.Propagator.Extract(ctx.Request)
	if err != nil {
		return nil, err
	}
	session, err := m.Get(ctx.Request.Context(), sessId)
	if err != nil {
		return nil, err
	}
	ctx.UserValues[m.CtxSessionKey] = session
	return session, nil
}

func (m *Manager) InitSession(ctx *mist.Context) (Session, error) {
	id := uuid.New().String()
	sess, err := m.Generate(ctx.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	err = m.Inject(id, ctx.ResponseWriter)
	return sess, err
}

func (m *Manager) RefreshSession(ctx *mist.Context) error {
	sess, err := m.GetSession(ctx)
	if err != nil {
		return err
	}
	return m.Refresh(ctx.Request.Context(), sess.ID())
}

func (m *Manager) RemoveSession(ctx *mist.Context) error {
	sess, err := m.GetSession(ctx)
	if err != nil {
		return err
	}
	err = m.Store.Remove(ctx.Request.Context(), sess.ID())
	if err != nil {
		return err
	}
	return m.Propagator.Remove(ctx.ResponseWriter)
}
