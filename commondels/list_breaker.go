package commondels

// InputBreakerFunc is a function that can be used in ListBreaker.Validators (and
// matches the InputBreaker.Break function).
type InputBreakerFunc func(string, *Data) bool

// ListBreaker is an implementer of `InputBreaker`
type ListBreaker struct {
	Validators []InputBreakerFunc
	Discard    bool
	UsageFunc  func(*Usage)
}

func (lb *ListBreaker) Break(s string, d *Data) bool {
	for _, b := range lb.Validators {
		if b(s, d) {
			return true
		}
	}
	return false
}

func (lb *ListBreaker) DiscardBreak() bool { return lb.Discard }

func (lb *ListBreaker) Usage(u *Usage) {
	panic("AHAHA")
	// if lb.UsageFunc != nil {
	// lb.UsageFunc(u)
	// }
}

// BreakAtSymbol returns an `InputBreakerFunc` that breaks when encountering the
// provided symbol.
func BreakAtSymbol(symbol string) InputBreakerFunc {
	return func(s string, d *Data) bool {
		return s == symbol
	}
}
