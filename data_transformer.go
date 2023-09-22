package command

import (
	"fmt"
)

// DataTransformer transforms the value in Data under `key` using the provided function.
func DataTransformer[I, O any](key string, f func(I) (O, error)) Processor {
	return SuperSimpleProcessor(func(i *Input, d *Data) error {
		if !d.Has(key) {
			return fmt.Errorf("[DataTransformer] key is not set in Data")
		}

		output, err := f(GetData[I](d, key))
		if err != nil {
			return fmt.Errorf("[DataTransformer] failed to convert data: %v", err)
		}
		d.Set(key, output)

		return nil
	})
}
