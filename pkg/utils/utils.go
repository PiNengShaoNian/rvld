package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

func Fatal(v any) {
	fmt.Printf("rvld: \033[0;1;31mfatal:\033[0m %v\n", v)
	debug.PrintStack()
	os.Exit(0)
}

func MustNo(err error) {
	if err != nil {
		Fatal(err)
	}
}

func Assert(condition bool) {
	if !condition {
		Fatal("assert failed")
	}
}

func Read[T any](data []byte) (val T) {
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.LittleEndian, &val)
	MustNo(err)
	return
}

func ReadSlice[T any](data []byte, size int) []T {
	len := len(data) / size
	res := make([]T, 0, len)
	for len > 0 {
		res = append(res, Read[T](data))
		data = data[size:]
		len--
	}
	return res
}

func RemovePrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		s = strings.TrimPrefix(s, prefix)
		return s, true
	}
	return s, false
}

func RemoveIf[T any](elems []T, condition func(T) bool) []T {
	i := 0
	for _, elem := range elems {
		if condition(elem) {
			continue
		}
		elems[i] = elem
		i++
	}

	return elems[:i]
}

func AllZeros(bytes []byte) bool {
	b := byte(0)
	for _, s := range bytes {
		b |= s
	}

	return b == 0
}
