package tcp

import (
	"strings"
	"math/rand"
	"time"
)

func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}
	// TODO: remove this string search and be more like the Windows
	// case below. That might involve modifying the standard library
	// to return better error types.
	str := err.Error()
	if strings.Contains(str, "use of closed network connection") {
		return true
	}
	return false
}

func RandString(slen int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bt := []byte(str)
	result := make([]byte, 0)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	//slen := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1024)
	for i := 0; i < slen; i++ {
		result = append(result, bt[r.Intn(len(bt))])
	}
	return string(result)
}

