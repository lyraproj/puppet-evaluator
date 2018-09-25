package threadlocal

import "fmt"

func ExampleGetgid() {
	fmt.Println(getg() > 0)

	// Output: true
}
