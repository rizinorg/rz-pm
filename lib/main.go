package main

import (
	"log"
	"os"
	"strconv"

	"github.com/rizinorg/rz-pm/internal/features"
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
	log.SetPrefix("librz-pm: ")

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
func rizin_pm_delete(rzpmDir *C.char) C.int {
	err := features.Delete(C.GoString(rzpmDir))
	return getReturnValue(err)
}

//export rizin_pm_init
func rizin_pm_init(rzpmDir *C.char) C.int {
	err := features.Init(C.GoString(rzpmDir))
	return getReturnValue(err)
}

//export rizin_pm_install
func rizin_pm_install(rzpmDir, packageName *C.char) C.int {
	err := features.Install(C.GoString(rzpmDir), C.GoString(packageName))
	return getReturnValue(err)
}

//export rizin_pm_list_available
func rizin_pm_list_available(rzpmDir *C.char, list **C.struct_rizin_pm_string_list) C.int {
	entries, err := features.ListAvailable(C.GoString(rzpmDir))
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
func rizin_pm_list_installed(rzpmDir *C.char) (*C.struct_test, C.int) {
	entries, err := features.ListInstalled(C.GoString(rzpmDir))

	// TODO do not return nil
	_ = entries

	return nil, getReturnValue(err)
}

//export rizin_pm_uninstall
func rizin_pm_uninstall(rzpmDir, packageName *C.char) C.int {
	err := features.Uninstall(C.GoString(rzpmDir), C.GoString(packageName))
	return getReturnValue(err)
}

//export rizin_pm_set_debug
func rizin_pm_set_debug(value C.int) {
	features.SetDebug(value != 0)
}

func main() {}
