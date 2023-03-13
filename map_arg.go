package command

import (
	"fmt"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
)

// MapArg returns a `Processor` that converts an input key into it's value.
func MapArg[K constraints.Ordered, V any](name, desc string, m map[K]V, allowMissing bool) *MapArgument[K, V] {
	var keys []string
	for _, k := range maps.Keys(m) {
		keys = append(keys, fmt.Sprintf("%v", k))
	}
	ma := &MapArgument[K, V]{}
	opts := []ArgumentOption[K]{
		SimpleCompleter[K](keys...),
		&CustomSetter[K]{F: func(key K, d *Data) {
			d.Set(name, m[key])
			ma.key = key
		}},
	}

	if !allowMissing {
		opts = append(opts, &ValidatorOption[K]{
			func(k K, d *Data) error {
				if _, ok := m[k]; !ok {
					return fmt.Errorf("[MapArg] key (%v) is not in map", k)
				}
				return nil
			},
			"MapArg",
		})
	}
	ma.Argument = Arg(name, desc, opts...)
	return ma
}

type MapArgument[K constraints.Ordered, V any] struct {
	*Argument[K]
	key K
}

// Get overrides the Arg.Get function to return V (rather than type K).
func (man *MapArgument[K, V]) Get(d *Data) V {
	return GetData[V](d, man.name)
}

// GetKey returns the key that was set by the am
func (man *MapArgument[K, V]) GetKey() K {
	return man.key
}

// GetOrDefault overrides the Arg.GetOrDefault function to return V (rather than type K).
func (man *MapArgument[K, V]) GetOrDefault(d *Data, dflt V) V {
	if d.Has(man.name) {
		return GetData[V](d, man.name)
	}
	return dflt
}
