// Producer process that produces number and consumer that consumes number
//processes run concurrently and comunicate via a OS pipe.
package main

import (

	"fmt"
	"os"
	"os/exec"
	
)

func main () {
	//create a pipe

	r, w, _ := os.pipe()

	cmd := exec.Comman("cat")

	cmd.Stdion = r

	cmd.Stdout = os.Stout

	if err := cmd.Start(); err != nil {
		panic(err)		
	}

	fmt.Fprintln("Hello")
	fmt.Fprintln("Hello again")

	w.Close()

	cmd.Wait()
}
