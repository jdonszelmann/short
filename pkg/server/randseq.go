package server

import (
	"math/rand"
	"strings"
	"time"
)

func RandSeq(n int, params ...string) string {
	var chars []rune
	if len(params) == 0 {
		chars = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	} else if len(params) == 1 {
		chars = []rune(params[0])
	} else {
		panic("too many parameters")
	}
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}

	return b.String()
}

func NonExistentRandom(store *Store) (string, error) {
	for {
		res := RandSeq(5, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		alias, err := store.GetAlias(res)
		if err != nil {
			return "", err
		}
		if alias == nil {
			return res, nil
		}
	}
}
