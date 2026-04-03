package shortner

import (
	"strings"
)

const (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	base     = uint64(len(alphabet))
)

// return base 64 string
func Encode(id uint64) string {
	if id == 0 {
		return string(alphabet[0])
	}

	var res strings.Builder
	for id > 0 {
		res.WriteByte((alphabet[id%base]))
		id /= base
	}

	// flipping the string
	return reverse(res.String())
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)

}
