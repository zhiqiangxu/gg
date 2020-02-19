package main

import fmt2 "fmt"
import fmt3 "fmt"

type (
	// GlobalType struct
	GlobalType struct {
	}

	// some comment
	root struct {
		GlobalType
	}
)

// Hello method
func (d *GlobalType) Hello() {
	fmt3.Println("GlobalType.Hello")
}

// GlobalFunc is global func
func GlobalFunc() {
	fmt2.Println("GlobalFunc")
}

// GlobalVars is global variable
var GlobalVars int

// GlobalConst is global constant
const GlobalConst = 10

func main() {
	var t GlobalType
	t.Hello()

	GlobalFunc()
	fmt2.Println(GlobalVars, GlobalConst)
}
