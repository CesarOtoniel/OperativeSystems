
// q2.go based on ipc.go from hw0



package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func check(err error) { if err != nil { panic(err) } }

// Child processes 
func runProducerProc(n int) {
	dataW := os.NewFile(uintptr(3), "dataW") // send ints to consumer
	ackR := os.NewFile(uintptr(4), "ackR")   // wait for 1-byte acks
	defer dataW.Close(); defer ackR.Close()
	enc := gob.NewEncoder(dataW)
	buf := make([]byte, 1)
	for i := 1; i <= n; i++ {
		check(enc.Encode(i))
		_, err := io.ReadFull(ackR, buf)
		check(err)
	}
}
func runConsumerProc(n int) {
	dec := gob.NewDecoder(os.Stdin)          // read ints from producer
	ackW := os.NewFile(uintptr(3), "ackW")   // write 1-byte acks
	defer ackW.Close()
	for i := 0; i < n; i++ {
		var v int
		check(dec.Decode(&v))
		_, err := ackW.Write([]byte{1})
		check(err)
	}
}

// test runners
func runProcOnce(n int) time.Duration {
	// pipes: data (P->C) and ack (C->P)
	dataR, dataW, err := os.Pipe(); check(err)
	ackR, ackW, err := os.Pipe(); check(err)

	prod := exec.Command(os.Args[0], "producer", strconv.Itoa(n))
	prod.Stdout, prod.Stderr = nil, os.Stderr
	prod.ExtraFiles = []*os.File{dataW, ackR}

	cons := exec.Command(os.Args[0], "consumer", strconv.Itoa(n))
	cons.Stdin = dataR
	cons.Stdout, cons.Stderr = nil, os.Stderr
	cons.ExtraFiles = []*os.File{ackW}

	start := time.Now()
	check(prod.Start()); check(cons.Start())

	// Close parent copies so EOFs propagate
	dataW.Close(); dataR.Close()
	ackR.Close(); ackW.Close()

	check(prod.Wait()); check(cons.Wait())
	return time.Since(start)
}

func runGoroutineOnce(n int) time.Duration {
	data := make(chan int)        // unbuffered -> strict handoff
	ack := make(chan struct{})    // 1-byte ack equivalent

	// consumer
	done := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			_ = <-data
			ack <- struct{}{}
		}
		close(done)
	}()

	start := time.Now()
	// producer
	for i := 1; i <= n; i++ {
		data <- i
		<-ack
	}
	close(data)
	<-done
	return time.Since(start)
}

// ---------- MAIN / BENCH HARNESS ----------
func main() {
	// child roles
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "producer":
			n, _ := strconv.Atoi(os.Args[2])
			runProducerProc(n)
			return
		case "consumer":
			n, _ := strconv.Atoi(os.Args[2])
			runConsumerProc(n)
			return
		}
	}

	// flags for the benchmakrs
	mode := flag.String("mode", "both", "proc|gor|both")
	n := flag.Int("n", 1_000_000, "numbers exchanged")
	warm := flag.Int("warmup", 1, "warmup runs per mode")
	reps := flag.Int("reps", 3, "timed runs per mode")
	flag.Parse()

	runAndReport := func(name string, fn func(int) time.Duration) {
		for i := 0; i < *warm; i++ { _ = fn(*n) } // warmup
		var total time.Duration
		best := time.Duration(1<<63 - 1)
		for i := 0; i < *reps; i++ {
			d := fn(*n)
			if d < best { best = d }
			total += d
			fmt.Printf("%s run %d: %v (%.2f Mmsgs/s)\n",
				name, i+1, d, float64(*n)/d.Seconds()/1e6)
		}
		avg := total / time.Duration(*reps)
		fmt.Printf("%s best: %v, avg: %v, throughput(avg): %.2f Mmsgs/s\n\n",
			name, best, avg, float64(*n)/avg.Seconds()/1e6)
	}

	switch *mode {
	case "proc":
		runAndReport("process", runProcOnce)
	case "gor":
		runAndReport("goroutine", runGoroutineOnce)
	default:
		runAndReport("process", runProcOnce)
		runAndReport("goroutine", runGoroutineOnce)
	}
}
