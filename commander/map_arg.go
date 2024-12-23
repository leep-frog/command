package commander

import (
	"fmt"
	"slices"

	"github.com/leep-frog/command/command"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
)

// MapArg returns a `command.Processor` that converts an input key into it's value.
func MapArg[K constraints.Ordered, V any](name, desc string, m map[K]V, allowMissing bool) *MapFlargument[K, V] {
	return MapFlag(name, FlagNoShortName, desc, m, allowMissing)
}

// MapFlag returns a `Flag` that converts an input key into it's value.
func MapFlag[K constraints.Ordered, V any](name string, shortName rune, desc string, m map[K]V, allowMissing bool) *MapFlargument[K, V] {
	var keys []string
	for _, k := range maps.Keys(m) {
		keys = append(keys, fmt.Sprintf("%v", k))
	}
	ma := &MapFlargument[K, V]{
		shortName: shortName,
	}
	opts := []ArgumentOption[K]{
		SimpleCompleter[K](keys...),
		&CustomSetter[K]{F: func(key K, d *command.Data) {
			v, ok := m[key]
			d.Set(name, v)
			ma.key = key
			ma.hit = ok
		}},
	}

	if !allowMissing {
		opts = append(opts, &ValidatorOption[K]{
			func(k K, d *command.Data) error {
				if _, ok := m[k]; !ok {
					keys := maps.Keys(m)
					slices.Sort(keys)
					return fmt.Errorf("[MapArg] key (%v) is not in map; expected one of %v", k, keys)
				}
				return nil
			},
			"MapArg",
		})
	}
	ma.Argument = Arg(name, desc, opts...)
	ma.allowMissing = allowMissing
	return ma
}

// MapFlargument is an `Argument` (or `Flag` if included in a `FlagProcessor(...)`)
// that retrieves data from a provided map. Use the `MapArg` to construct it.
type MapFlargument[K constraints.Ordered, V any] struct {
	*Argument[K]
	shortName    rune
	key          K
	hit          bool
	allowMissing bool
}

func (man *MapFlargument[K, V]) ShortName() rune {
	return man.shortName
}

// Get overrides the Arg.Get function to return V (rather than type K).
func (man *MapFlargument[K, V]) Get(d *command.Data) V {
	return command.GetData[V](d, man.name)
}

func (man *MapFlargument[K, V]) Provided(d *command.Data) bool {
	return d.Has(man.name)
}

// GetKey returns the key that was set by the am
func (man *MapFlargument[K, V]) GetKey() K {
	return man.key
}

// GetOrDefault overrides the Arg.GetOrDefault function to return V (rather than type K).
func (man *MapFlargument[K, V]) GetOrDefault(d *command.Data, dflt V) V {
	if d.Has(man.name) {
		return command.GetData[V](d, man.name)
	}
	return dflt
}

func (man *MapFlargument[K, V]) AddOptions(opts ...ArgumentOption[V]) FlagWithType[V] {
	panic("MapFlargument cannot have options added to it")
}

func (man *MapFlargument[K, V]) Options() *FlagOptions {
	return &FlagOptions{}
}

// Hit returns whether the key provided was actually present in the map.
func (man *MapFlargument[K, V]) Hit() bool {
	return man.hit
}

func (man *MapFlargument[K, V]) Processor() command.Processor {
	return man
}

func (man *MapFlargument[K, V]) FlagUsage(d *command.Data, u *command.Usage) error {
	argName := "MAP_KEY"
	if man.allowMissing {
		argName = fmt.Sprintf("MAP_KEY_OR_%s", argifyFlagName(man.name))
	}
	u.AddFlag(man.name, man.shortName, argName, man.desc, 1, 0)
	return nil
}
