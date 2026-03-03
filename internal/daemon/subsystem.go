package daemon

import "context"

type Subsystem interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() SubsystemHealth
}

type SubsystemHealth struct {
	Name    string
	Status  string
	Message string
}
