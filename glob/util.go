package glob

import (
	"crypto/rand"
	"encoding/base64"
	"math"
)

func Ptr[T any](v T) *T {
	return &v
}

/* Used for map names */
func RandomBase64String(l int) string {
	buff := make([]byte, int(math.Ceil(float64(l)/float64(1.33333333333))))
	_, _ = rand.Read(buff)

	str := base64.RawURLEncoding.EncodeToString(buff)
	/* strip 1 extra character we get from odd length results */
	return str[:l]
}
