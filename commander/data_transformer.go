package commander

import (
	"fmt"

	"github.com/leep-frog/command/commondels"
)

// DataTransformer transforms the value in commondels.Data under `key` using the provided function.
func DataTransformer[I, O any](key string, f func(I) (O, error)) commondels.Processor {
	return SuperSimpleProcessor(func(i *commondels.Input, d *commondels.Data) error {
		if !d.Has(key) {
			return fmt.Errorf("[DataTransformer] key is not set in commondels.Data")
		}

		output, err := f(commondels.GetData[I](d, key))
		if err != nil {
			return fmt.Errorf("[DataTransformer] failed to convert data: %v", err)
		}
		d.Set(key, output)

		return nil
	})
}
