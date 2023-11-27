package commondels

type OS interface {
	// SetEnvVar returns a shell command that sets the environment variable
	// `envVar` to `value`. Environment variable modifications can't and shouldn't
	// be done by os.Setenv because the go CLI executable is run in a sub-shell.
	SetEnvVar(envVar, value string) string

	// UnsetEnvVar returns a shell command that unsets the environment variable
	// `envVar`. Environment variable changes can't and shouldn't be done by
	// os.Unsetenv because the go CLI executable is run in a sub-shell.
	UnsetEnvVar(envVar string) string

	// ShellCommandFileRunner returns the command and command arguments
	// to run a file in the shell
	// ShellCommandFileRunner(file string) (string, []string)
}

// Data contains argument data.
type Data struct {
	// Values is a map from argument name to the data for that argument.
	Values map[string]interface{}
	// complexecute indictes whether we are running complexecute logic.
	complexecute bool
	// OS is the current operating system
	OS OS
}

// Set sets the provided key-value pair in the `Data` object.
func (d *Data) Set(k string, i interface{}) {
	if d.Values == nil {
		d.Values = map[string]interface{}{}
	}
	d.Values[k] = i
}

// GetData fetches the value for a given key.
func GetData[T any](d *Data, key string) T {
	var ret T
	if d == nil || d.Values == nil {
		return ret
	}
	i, ok := d.Values[key]
	if !ok {
		return ret
	}
	return i.(T)
}

// Has returns whether or not key has been set in the `Data` object.
func (d *Data) Has(k string) bool {
	_, ok := d.Values[k]
	return ok
}

// Removed Regexp. Callers are responsible for storing that

// String returns the string data for an argument.
func (d *Data) String(k string) string {
	return GetData[string](d, k)
}

// Get returns the interface data for an argument.
func (d *Data) Get(k string) interface{} {
	return GetData[interface{}](d, k)
}

// StringList returns the string slice data for an argument.
func (d *Data) StringList(k string) []string {
	return GetData[[]string](d, k)
}

// Int returns the int data for an argument.
func (d *Data) Int(k string) int {
	return GetData[int](d, k)
}

// IntList returns the int slice data for an argument.
func (d *Data) IntList(k string) []int {
	return GetData[[]int](d, k)
}

// Float returns the float data for an argument.
func (d *Data) Float(k string) float64 {
	return GetData[float64](d, k)
}

// FloatList returns the float slice data for an argument.
func (d *Data) FloatList(k string) []float64 {
	return GetData[[]float64](d, k)
}

// Bool returns the bool data for an argument.
func (d *Data) Bool(k string) bool {
	return GetData[bool](d, k)
}
