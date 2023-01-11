package snappy

import "testing"

func FuzzData(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		func() int {
			decode, err := Decode(data)
			if decode == nil && err == nil {
				panic("nil error with nil result")
			}

			if err != nil {
				return 0
			}

			return 1
		}()
	})
}
