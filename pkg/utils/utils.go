package utils

import (
	"fmt"
	"os"
	"runtime/debug"
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
