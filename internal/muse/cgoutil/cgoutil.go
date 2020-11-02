package cgoutil

import (
	"log"
	"unsafe"
)

// #include <stdlib.h>
import "C"

// GoStrings returns a slice of strings.
func GoStrings(uptrIface interface{}) []string {
	var strings []string

	uptr := uptrIface.(unsafe.Pointer)
	arrayPtr := (**C.char)(uptr)

	var charPtr = (*C.char)(*arrayPtr)
	for {
		log.Printf(" charPtr: %p\n", unsafe.Pointer(charPtr))
		log.Printf("*charPtr: %p\n", unsafe.Pointer(uintptr(*charPtr)))

		if uintptr(unsafe.Pointer(charPtr)) < 1 {
			break
		}

		if uintptr(*charPtr) == 0 {
			break
		}

		strings = append(strings, C.GoString(charPtr))
		log.Println(strings)

		arrayPtr = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(arrayPtr)) + 1))
		charPtr = (*C.char)(*arrayPtr)
	}

	return strings
}
