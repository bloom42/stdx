package httpx

import (
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
)

// func TestWebapphandlerTypes(t *testing.T) {
// 	var fileInfo webappFileInfo
// 	fmt.Printf("size(webappFileInfo): %d\n", unsafe.Sizeof(fileInfo))
// }

func TestEncodeEtagOptimized(t *testing.T) {
	elems := []struct {
		Input    string
		Expected string
		Hash     [32]byte
	}{
		{
			Input:    "1",
			Expected: "",
		},
		{
			Input:    "2",
			Expected: "",
		},
		{
			Input:    "3",
			Expected: "",
		},
		{
			Input:    "Hello World!",
			Expected: "",
		},
	}
	for i := range elems {
		elems[i].Hash = sha256.Sum256([]byte(elems[i].Input))
		elems[i].Expected = strconv.Quote(base64.RawURLEncoding.EncodeToString(elems[i].Hash[:]))
	}

	for _, elem := range elems {
		value := encodeEtagOptimized(elem.Hash)
		if value != elem.Expected {
			t.Errorf("expected: %s | got: %s | input: %s\n", elem.Expected, value, elem.Input)
		}
	}
}

func BenchmarkEncodeEtagOptimized(b *testing.B) {
	hash := sha256.Sum256([]byte("Hello World!"))

	b.Run("strconv.Quote(base64.RawURLEncoding.EncodeToString(hash))", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(hash)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			strconv.Quote(base64.RawURLEncoding.EncodeToString(hash[:]))
		}
	})

	b.Run("encodeEtag(hash)", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(hash)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encodeEtagOptimized(hash)
		}
	})

	// b.Run("toEtagPlus(hash)", func(b *testing.B) {
	// 	b.ReportAllocs()
	// 	b.SetBytes(int64(len(hash)))
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		toEtagPlus(hash)
	// 	}
	// })
}
