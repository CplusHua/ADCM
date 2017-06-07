package main

import (
	"flag"
	"fmt"
	"github.com/nicle-lin/ADCM/lib/update"
	"os"
	"runtime"
	"strings"
)

var usage = `Usage: upgrade [ip] [port] [password] [ssu]`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}
	flag.Parse()
	if flag.NArg() < 4 {
		usageAndExit("")
	}

	ips := os.Args[1]
	ip := strings.Fields(ips)
	port := os.Args[2]
	password := os.Args[3]
	ssu := os.Args[4]
	fmt.Println(ip, port, password, ssu)
	update.ThreadUpgrade(ip, port, password, ssu)

}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
