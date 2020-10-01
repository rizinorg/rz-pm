package main

import (
	"log"
	"os"
	"strconv"

	"github.com/radareorg/r2pm/internal/features"
)

/*
#include <stdlib.h>

struct rizin_pm_string_list{
	struct rizin_pm_string_list* next;
	char* s;
};
*/
import "C"

const (
	Error   = -1
	Success = 0
)

func init() {
	log.SetPrefix("libr2pm: ")

	// Enable the logger if the environment variable is a valid boolean
	env := os.Getenv(features.DebugEnvVar)

	if val, err := strconv.ParseBool(env); err == nil && val {
		rizin_pm_set_debug(1)
	} else {
		rizin_pm_set_debug(0)
	}
}

func getReturnValue(err error) C.int {
	if err != nil {
		log.Fatal(err.Error())
	}

	return Success
}

//export rizin_pm_delete
func rizin_pm_delete(r2pmDir *C.char) C.int {
	err := features.Delete(C.GoString(r2pmDir))
	return getReturnValue(err)
}

//export rizin_pm_init
func rizin_pm_init(r2pmDir *C.char) C.int {
	err := features.Init(C.GoString(r2pmDir))
	return getReturnValue(err)
}

//export rizin_pm_install
func rizin_pm_install(r2pmDir, packageName *C.char) C.int {
	err := features.Install(C.GoString(r2pmDir), C.GoString(packageName))
	return getReturnValue(err)
}

//export rizin_pm_list_available
func rizin_pm_list_available(r2pmDir *C.char, list **C.struct_rizin_pm_string_list) C.int {
	entries, err := features.ListAvailable(C.GoString(r2pmDir))
	if err != nil {
		return Error
	}

	if len(entries) == 0 {
		*list = nil
		return Success
	}

	newNode := func() *C.struct_rizin_pm_string_list {
		m := C.calloc(1, C.sizeof_struct_rizin_pm_string_list)
		return (*C.struct_rizin_pm_string_list)(m)
	}

	start := newNode()
	start.s = C.CString(entries[0].Name)

	previous := start

	for _, e := range entries[1:] {
		previous.next = newNode()
		previous.next.s = C.CString(e.Name)

		previous = previous.next
	}

	*list = start

	return Success
}

//export rizin_pm_list_installed
func rizin_pm_list_installed(r2pmDir *C.char) (*C.struct_test, C.int) {
	entries, err := features.ListInstalled(C.GoString(r2pmDir))

	// TODO do not return nil
	_ = entries

	return nil, getReturnValue(err)
}

//export rizin_pm_uninstall
func rizin_pm_uninstall(r2pmDir, packageName *C.char) C.int {
	err := features.Uninstall(C.GoString(r2pmDir), C.GoString(packageName))
	return getReturnValue(err)
}

//export rizin_pm_set_debug
func rizin_pm_set_debug(value C.int) {
	features.SetDebug(value != 0)
}

func main() {}
