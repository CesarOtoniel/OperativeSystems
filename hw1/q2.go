//Prompts used: "Explain how goroutines work"
//				"How does the code in goroutine flows"
//				"Does go handle the excecution in all platforms?"
//				"Check this: <Checks all code for syntax>"
//				"How do I format time as human readable?"
//				""
//				""
//				""
//				""


package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

/************* Shared helpers *************/
func check(err error) { if err != nil { panic(err) } }
func ts() string      { return time.Now().Format(time.RFC3339Nano) }

/************* CHILD ROLES (true processes) *************/
func childProducer(count int, verbose bool) {
	dataW := os.NewFile(uintptr(3), "dataW") // from parent
	ackR  := os.NewFile(uintptr(4), "ackR")
	defer dataW.Close()
	defer ackR.Close()

	enc := gob.NewEncoder(dataW)
	buf := make([]byte, 1)

	for i := 1; i <= count; i++ {
		if verbose { fmt.Printf("[%s] Producer: %d\n", ts(), i) }
		check(enc.Encode(i))
		_, err := io.ReadFull(ackR, buf)
		check(err)
	}
}

func childConsumer(count int, verbose bool) {
	dec  := gob.NewDecoder(os.Stdin)
	ackW := os.NewFile(uintptr(3), "ackW")
	defer ackW.Close()

	for i := 0; i < count; i++ {
		var v int
		check(dec.Decode(&v))
		if verbose { fmt.Printf("[%s] Consumer: %d\n", ts(), v) }
		_, err := ackW.Write([]byte{1})
		check(err)
	}
}

/************* PROCESS-BASED SEQUENTIAL *************/
func runOneProcessPair(count, id int, verbose bool) {
	// pipes: data Producer→Consumer, ack Consumer→Producer
	dataR, dataW, err := os.Pipe(); check(err)
	ackR,  ackW,  err := os.Pipe(); check(err)

	// Producer child: gets FD3=dataW, FD4=ackR
	prod := exec.Command(os.Args[0], "producer",
		fmt.Sprintf("%d", count), boolToArg(verbose))
	prod.Stdout = os.Stdout
	prod.Stderr = os.Stderr
	prod.ExtraFiles = []*os.File{dataW, ackR}

	// Consumer child: stdin=dataR, FD3=ackW
	cons := exec.Command(os.Args[0], "consumer",
		fmt.Sprintf("%d", count), boolToArg(verbose))
	cons.Stdin  = dataR
	cons.Stdout = os.Stdout
	cons.Stderr = os.Stderr
	cons.ExtraFiles = []*os.File{ackW}

	if verbose { fmt.Printf("[%s] --- PROC Pair %d begin ---\n", ts(), id) }

	check(prod.Start())
	check(cons.Start())

	// parent closes its copies so EOFs propagate
	dataW.Close(); dataR.Close()
	ackR.Close();  ackW.Close()

	prod.Wait()
	cons.Wait()

	if verbose { fmt.Printf("[%s] --- PROC Pair %d end ---\n", ts(), id) }
}

func runProcessSequential(pairs, count int, verbose bool) time.Duration {
	start := time.Now()
	fmt.Printf("[%s] SEQ(PROC) start (pairs=%d, count=%d)\n", ts(), pairs, count)
	for id := 1; id <= pairs; id++ {
		runOneProcessPair(count, id, verbose)
	}
	d := time.Since(start)
	fmt.Printf("[%s] SEQ(PROC) end   duration=%v\n", ts(), d)
	return d
}

/************* GOROUTINE-BASED CONCURRENT *************/
func goProducer(id, count int, data chan<- int, ack <-chan struct{}, verbose bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 1; i <= count; i++ {
		if verbose { fmt.Printf("[%s] Producer[%d] -> %d\n", ts(), id, i) }
		data <- i
		<-ack
	}
	close(data)
}

func goConsumer(id int, data <-chan int, ack chan<- struct{}, verbose bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for v := range data {
		if verbose { fmt.Printf("[%s] Consumer[%d] <- %d\n", ts(), id, v) }
		ack <- struct{}{}
	}
	close(ack)
}

func runGoroutineConcurrent(pairs, count int, verbose bool) time.Duration {
	start := time.Now()
	fmt.Printf("[%s] CONC(GO) start (pairs=%d, count=%d)\n", ts(), pairs, count)

	var wg sync.WaitGroup
	wg.Add(pairs * 2)
	for id := 1; id <= pairs; id++ {
		data := make(chan int)
		ack := make(chan struct{})
		go goProducer(id, count, data, ack, verbose, &wg)
		go goConsumer(id, data, ack, verbose, &wg)
	}
	wg.Wait()

	d := time.Since(start)
	fmt.Printf("[%s] CONC(GO) end   duration=%v\n", ts(), d)
	return d
}

/************* MAIN *************/
func boolToArg(b bool) string {
	if b { return "1" }
	return "0"
}
func argToBool(s string) bool { return s == "1" }

func main() {
	// CHILD ROLES (invoked by exec)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "producer":
			// args: producer <count> <verbose(0|1)>
			count := 5
			verbose := false
			if len(os.Args) > 2 { fmt.Sscanf(os.Args[2], "%d", &count) }
			if len(os.Args) > 3 { verbose = argToBool(os.Args[3]) }
			childProducer(count, verbose)
			return
		case "consumer":
			// args: consumer <count> <verbose(0|1)>
			count := 5
			verbose := false
			if len(os.Args) > 2 { fmt.Sscanf(os.Args[2], "%d", &count) }
			if len(os.Args) > 3 { verbose = argToBool(os.Args[3]) }
			childConsumer(count, verbose)
			return
		}
	}

	// PARENT BENCHMARK DRIVER
	pairs   := flag.Int("pairs", 4, "number of producer/consumer pairs")
	count   := flag.Int("count", 5, "items produced per pair")
	verbose := flag.Bool("v", false, "print per-item timestamps")
	flag.Parse()

	seq := runProcessSequential(*pairs, *count, *verbose)
	con := runGoroutineConcurrent(*pairs, *count, *verbose)

	fmt.Printf("\nSUMMARY\n")
	fmt.Printf("Sequential (process): %v\n", seq)
	fmt.Printf("Concurrent (goroutine): %v\n", con)
	if con < seq {
		fmt.Printf("Speedup: %.2fx\n", float64(seq)/float64(con))
	}
}
