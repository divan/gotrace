package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	output := flag.String("o", "", "Output trace in JSON format to this file")
	flag.Usage = Usage
	flag.Parse()
	args := flag.Args()

	var src EventSource
	if len(args) == 1 {
		if strings.HasSuffix(args[0], ".json") {
			commands, err := ioutil.ReadFile(args[0])
			if err != nil {
				panic(err)
			}
			_ = commands
			//ProcessCommands(*output, commands)
			return
		}

		src = NewNativeRun(args[0])
	} else if len(args) == 2 {
		src = NewTraceSource(args[0])
	} else {
		Usage()
		os.Exit(1)
	}

	events, err := src.Events()
	if err != nil {
		panic(err)
	}

	commands, err := ConvertEvents(events)
	if err != nil {
		panic(err)
	}

	ProcessCommands(*output, commands)
}

// ProcessCommands processes command list.
func ProcessCommands(out string, cmds Commands) {
	params := GuessParams(cmds)

	data := cmds.toJSON()
	if out != "" {
		err := ioutil.WriteFile(out, data, 0644)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Written commands to %s\n", out)
		return
	}

	StartServer(":2000", data, params)
}

// Usage prints usage information, overriding default one.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [-o file] [trace.out] or [trace.json] or [main.go]\n", os.Args[0])
	flag.PrintDefaults()
}
