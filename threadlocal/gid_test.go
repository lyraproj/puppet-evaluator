package threadlocal

import (
	"fmt"
	"time"
)

func ExampleGo() {
	Init()

	Go(func() {
		Set(`key`, `value 1`)
		time.Sleep(1 * time.Millisecond)
		v, _ := Get(`key`)
		fmt.Printf("g1 = %s\n", v)
		Delete(`key`)
	})

	Go(func() {
		Set(`key`, `value 2`)
		time.Sleep(1 * time.Millisecond)
		v, _ := Get(`key`)
		fmt.Printf("g2 = %s\n", v)
		Delete(`key`)
	})

	time.Sleep(2 * time.Millisecond)
	_, ok := Get(`key`)
	fmt.Printf("main = %v\n", ok)
	// Unordered output:
	// g1 = value 1
	// g2 = value 2
	// main = false
}
