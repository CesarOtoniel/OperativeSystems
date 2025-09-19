package main

import (
    "fmt"
    "os"
)

// Stack data structure using an array of fixed size
type Stack struct {
    data [100]int // fixed-size array
    top  int      // index of the top element (-1 means empty)
}

// initialize stack
func (s *Stack) Init() {
    s.top = -1
}

// push element onto stack
func (s *Stack) Push(value int) {
    if s.top >= len(s.data)-1 {
        fmt.Println("Stack overflow!")
        return
    }
    s.top++
    s.data[s.top] = value
    fmt.Printf("Pushed %d\n", value)
}

// pop element from stack
func (s *Stack) Pop() int {
    if s.top < 0 {
        fmt.Println("Stack underflow!")
        return -1 // sentinel error value
    }
    val := s.data[s.top]
    s.top--
    fmt.Printf("Popped %d\n", val)
    return val
}

func main() {
    var s Stack
    s.Init()

    if len(os.Args) < 2 {
        fmt.Println("Usage: go run stack.go [normal|overflow|underflow]")
        return
    }

    mode := os.Args[1]

    switch mode {
    case "normal":
        // push 5 values, then pop them
        for i := 1; i <= 5; i++ {
            s.Push(i * 10)
        }
        for i := 0; i < 5; i++ {
            s.Pop()
        }

    case "overflow":
        // push more than 100 values to trigger overflow
        for i := 1; i <= 105; i++ {
            s.Push(i)
        }

    case "underflow":
        // pop without pushing anything
        for i := 0; i < 5; i++ {
            s.Pop()
        }

    default:
        fmt.Println("Unknown mode:", mode)
    }
}
