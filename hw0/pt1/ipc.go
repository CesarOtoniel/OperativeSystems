// ipc.go

//chatgpt prompts: How to compile and build go?
//   			 : Explain succintly unless especified otherwise, use one liners if possible.
//   			 : Reffer me to functions useful when creating pipes in go.
//   			 : what is gob encoding in go?
//   			 : explain the := syntax
//   			 : explain _ in the go syntax.
//   			 : explain how to use pointers in go. Make references compared to C. Extend to as much as needed to be useful.
//   			 : is nil equivalent to a null pointer?
//   			 : I get errors on my code **debug from terminal**
//   			 : What is a FD?
//   			 : So, it's like the values you get when you run a command normally loke -1 for error 0 for success and 1 for



package main

// import libraries necessary for encoding, formatting, access to OS-level features, I/O, and external processes.
import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// utility function to panic on error
func check(err error) { 
	if err != nil { panic(err) } 
}

// Runs Producer process
func runProducer() {
	dataW := os.NewFile(uintptr(3), "dataW") // FD 3 from parent (write end for sending data)
	ackR  := os.NewFile(uintptr(4), "ackR")  // FD 4 from parent (read end for acknowledgements)
	defer dataW.Close()						// ensure data pipe closes gracefully
	defer ackR.Close()						// ensure ack pipe closes gracefully

	enc := gob.NewEncoder(dataW)				// gob encoder to send integers
	buf := make([]byte, 1)						// buffer for reading ack byte

	for i := 1; i <= 5; i++ {
		fmt.Printf("Producer: %d\n", i)			// print Producer log to stdout
		check(enc.Encode(i))					// send integer to Consumer via data pipe
		_, err := io.ReadFull(ackR, buf)		// block until ack received from Consumer
		check(err)
	}
}

// Runs Consumer process
func runConsumer() {
	dec  := gob.NewDecoder(os.Stdin)       		// decoder reads from stdin (connected to Producer’s data pipe)
	ackW := os.NewFile(uintptr(3), "ackW")  	// FD 3 from parent (write end for acknowledgements)
	defer ackW.Close()							// close ack pipe gracefully

	for i := 0; i < 5; i++ {
		var v int
		check(dec.Decode(&v))					// read integer from Producer
		fmt.Printf("Consumer: %d\n", v)			// print Consumer log to stdout
		_, err := ackW.Write([]byte{1})			// send one-byte ack to Producer
		check(err)
	}
}

// Main function manages both Consumer and Producer processes
func main() {
	// if the process was launched as "producer" or "consumer", run respective function
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "producer": runProducer(); return
		case "consumer": runConsumer(); return
		}
	}

	// create two pipes: one for data Producer→Consumer, one for ack Consumer→Producer
	dataR, dataW, err := os.Pipe() 
	check(err)
	ackR,  ackW,  err := os.Pipe()
	check(err)

	// Producer: logs to stdout, sends data via FD 3, receives ack via FD 4
	prod := exec.Command(os.Args[0], "producer")
	prod.Stdout = os.Stdout
	prod.Stderr = os.Stderr
	prod.ExtraFiles = []*os.File{dataW, ackR}

	// Consumer: reads data from stdin (connected to dataR), sends ack via FD 3
	cons := exec.Command(os.Args[0], "consumer")
	cons.Stdin  = dataR
	cons.Stdout = os.Stdout
	cons.Stderr = os.Stderr
	cons.ExtraFiles = []*os.File{ackW}

	// start both child processes
	check(prod.Start())
	check(cons.Start())

	// close parent’s copies of the pipe ends (important so EOFs propagate)
	dataW.Close(); dataR.Close()
	ackR.Close();  ackW.Close()

	// wait for both processes to finish
	prod.Wait()
	cons.Wait()
}

