package snappy

import (
	"bytes"
	"testing"
)

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

func FuzzEncode(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		t.Run("xerial", func(t *testing.T) {
			encoded := EncodeStream(make([]byte, 0, len(data)/2), data)
			decoded, err := Decode(encoded)
			if err != nil {
				t.Errorf("input: %+v, encoded: %+v", data, encoded)
				t.Fatal(err)
			}
			if !bytes.Equal(decoded, data) {
				t.Fatal("mismatch")
			}

		})
		t.Run("snappy", func(t *testing.T) {
			encoded := Encode(data)
			decoded, err := Decode(encoded)
			if err != nil {
				t.Errorf("input: %+v, encoded: %+v", data, encoded)
				t.Fatal(err)
			}
			if !bytes.Equal(decoded, data) {
				t.Fatal("mismatch")
			}
		})
	})
}
