package threadlocal

import (
	"fmt"
	"runtime"
	"sync"
)

// Getgid returns the ID of the current Go routine
//
// The solution here is not the fastest in the world but it gets the job done
// in a portable way. For a faster ways to do this that involves assembler code
// for different platforms, please look at: https://github.com/huandu/go-tls
func getg() int64 {
	const prefixLen = 10 // Length of prefix "goroutine "
	var buf [64]byte

	l := runtime.Stack(buf[:64], false)
	n := int64(0)
	for i := prefixLen; i < l; i++ {
		d := buf[i]
		if d < 0x30 || d > 0x39 {
			break
		}
		n = n*10 + int64(d-0x30)
	}
	if n == 0 {
		panic(fmt.Errorf(`unable to retrieve id of current go routine`))
	}
	return n
}

var tlsLock sync.RWMutex
var tls = make(map[int64]map[string]interface{}, 7)

// Init initializes a go routine local storage for the current go routine
func Init() {
	gid := getg()
	ls := make(map[string]interface{})
	tlsLock.Lock()
	tls[gid] = ls
	tlsLock.Unlock()
}

// Cleanup deletes the local storage for the current go routine
func Cleanup() {
	gid := getg()
	tlsLock.Lock()
	delete(tls, gid)
	tlsLock.Unlock()
}

// Get returns a variable from the local storage of the current go routine
func Get(key string) (interface{}, bool) {
	gid := getg()
	tlsLock.RLock()
	ls, ok := tls[gid]
	tlsLock.RUnlock()
	var found interface{}
	if ok {
		found, ok = ls[key]
	}
	return found, ok
}

// Go executes the given function in a go routine and ensures that the local
// storage is initialized before the function is called and deleted before
// after the function returns or panics
func Go(f func()) {
	go func() {
		defer Cleanup()
		Init()
		f()
	}()
}

// Delete deletes a variable from the local storage of the current go routine
func Delete(key string) {
	gid := getg()
	tlsLock.RLock()
	ls, ok := tls[gid]
	tlsLock.RUnlock()
	if ok {
		delete(ls, key)
	}
}

// Set adds or replaces a variable to the local storage of the current go routine
func Set(key string, value interface{}) {
	gid := getg()
	tlsLock.RLock()
	ls, ok := tls[gid]
	tlsLock.RUnlock()
	if !ok {
		panic(`thread local not initialized for current go routine`)
	}
	ls[key] = value
}
