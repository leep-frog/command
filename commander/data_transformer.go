package commander

import (
	"fmt"

	"github.com/leep-frog/command/command"
)

// DataTransformer transforms the value in command.Data under `key` using the provided function.
func DataTransformer[I, O any](key string, f func(I) (O, error)) command.Processor {
	return SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
		if !d.Has(key) {
			return fmt.Errorf("[DataTransformer] key is not set in command.Data")
		}

		output, err := f(command.GetData[I](d, key))
		if err != nil {
			return fmt.Errorf("[DataTransformer] failed to convert data: %v", err)
		}
		d.Set(key, output)

		return nil
	})
}
