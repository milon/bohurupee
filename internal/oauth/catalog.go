package oauth

import (
	"fmt"
	"strings"
)

type Catalog struct {
	list []Persona
	byID map[string]Persona
}

func NewCatalog(personas []Persona) (*Catalog, error) {
	if len(personas) == 0 {
		return nil, fmt.Errorf("at least one persona is required")
	}
	c := &Catalog{
		list: make([]Persona, 0, len(personas)),
		byID: make(map[string]Persona, len(personas)),
	}
	for _, p := range personas {
		key := strings.ToLower(p.ID)
		if _, dup := c.byID[key]; dup {
			return nil, fmt.Errorf("duplicate persona id %q", p.ID)
		}
		c.list = append(c.list, p)
		c.byID[key] = p
	}
	return c, nil
}

func MustCatalog(personas []Persona) *Catalog {
	c, err := NewCatalog(personas)
	if err != nil {
		panic(err)
	}
	return c
}

func DefaultCatalog() *Catalog {
	return MustCatalog([]Persona{Alice})
}

func (c *Catalog) All() []Persona {
	out := make([]Persona, len(c.list))
	copy(out, c.list)
	return out
}

func (c *Catalog) Lookup(id string) (Persona, bool) {
	p, ok := c.byID[strings.ToLower(strings.TrimSpace(id))]
	return p, ok
}

func (c *Catalog) Default() Persona {
	if p, ok := c.Lookup(Alice.ID); ok {
		return p
	}
	return c.list[0]
}
