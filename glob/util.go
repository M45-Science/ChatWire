package glob

import (
	"crypto/rand"
	"encoding/base64"
	"math"
	mathrand "math/rand"
)

// Ptr returns a pointer to the provided value.
func Ptr[T any](v T) *T {
	return &v
}

// RandomBase64String generates a random base64 string of length l.
// This is primarily used when generating map names.
func RandomBase64String(l int) string {
	buff := make([]byte, int(math.Ceil(float64(l)/float64(1.33333333333))))
	if _, err := rand.Read(buff); err != nil {
		if CWLogger != nil {
			CWLogger("RandomBase64String: rand.Read failure: %v", err)
		}
		for i := range buff {
			buff[i] = byte(mathrand.Intn(256))
		}
	}

	str := base64.RawURLEncoding.EncodeToString(buff)
	/* strip 1 extra character we get from odd length results */
	if len(str) < l {
		return ""
	}
	return str[:l]
}
