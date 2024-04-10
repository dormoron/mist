package mist

type Middleware func(next HandleFunc) HandleFunc
