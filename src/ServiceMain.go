package main

import "fmt"
import "flag"
import "tempconv"

var test = 2
var p *int

var n = flag.Bool("n", false, "omit trailing newline")
var sep = flag.String("s", " ", "separator")

type Celsius float64    //test
type Fahrenheit float64 //

func main() {
	fmt.Println("Hello webrtcmcu")
	b := Celsius(212.0)
	fmt.Println(b.String()) // "100°C"
	fmt.Printf("%g\n", b)
}

func (c Celsius) String() string { return fmt.Sprintf("%g°C", c) }

func fib(n int) int {
	x, y := 0, 1
	for i := 0; i < n; i++ {
		x, y = y, x+y
	}
	return x
}
