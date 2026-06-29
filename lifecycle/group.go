package lifecycle

import (
	"context"
	"fmt"
)

var _ Component = (*Group)(nil)

// Group runs a named set of child components in registration order.
type Group struct {
	name       string
	components []Component
}

func NewGroup(name string, components ...Component) *Group {
	return &Group{
		name:       name,
		components: components,
	}
}

func (g *Group) Name() string { return g.name }

func (g *Group) Start(ctx context.Context) error {
	for _, component := range g.components {
		if err := component.Start(ctx); err != nil {
			return fmt.Errorf("start %s: %w", component.Name(), err)
		}
	}
	return nil
}

func (g *Group) Stop(ctx context.Context) error {
	var errs []error
	for i := len(g.components) - 1; i >= 0; i-- {
		if err := g.components[i].Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop %s: %w", g.components[i].Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("group %s stop failed: %v", g.name, errs)
	}
	return nil
}

// Components returns child components for flat registration in main.
func (g *Group) Components() []Component {
	return g.components
}
