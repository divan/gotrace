package main

import (
	"container/list"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/divan/gotrace/trace"
)

func ConvertEvents(events []*trace.Event) (Commands, error) {
	c := NewCommands()

	sends := list.New()

	debug := os.Getenv("GOTRACE_DEBUG") == "1"
	var lastG uint64
	for _, ev := range events {
		switch ev.Type {
		case trace.EvGoStart:
			if debug {
				fmt.Println(ev.Ts, "GoStart:", ev.G, "from", lastG, ev.Args)
			}
			if ev.G == 1 && lastG == 0 {
				c.StartGoroutine(ev.Ts, ev.Stk[0].Fn, ev.G, lastG)
			}
			lastG = ev.G
			c.UnblockGoroutine(ev.Ts, lastG)
		case trace.EvGoCreate:
			if len(ev.Stk) > 0 {
				if strings.HasPrefix(ev.Stk[0].Fn, "runtime") {
					if ev.Stk[0].Fn != "runtime.main" {
						break
					}
				}
				c.StartGoroutine(ev.Ts, ev.Stk[0].Fn, ev.Args[0], ev.G)
				if debug {
					fmt.Println(ev.Ts, "GoCreate:", ev.Args[0], "from", ev.G)
				}
			}
		case trace.EvGoUnblock:
			if len(ev.Stk) > 0 {
				if strings.HasPrefix(ev.Stk[0].Fn, "runtime") {
					if ev.Stk[0].Fn != "runtime.main" {
						break
					}
				}
			}
			lastG = ev.Args[0]
			c.UnblockGoroutine(ev.Ts, lastG)
			if debug {
				fmt.Println(ev.Ts, "GoUnblock: set lastG to", lastG, ev.Args)
			}
		case trace.EvGoEnd:
			if debug {
				fmt.Println(ev.Ts, "GoEnd:", ev.G)
			}
			c.StopGoroutine(ev.Ts, "", ev.G)
			lastG = ev.G
		case trace.EvGoSend:
			if debug {
				fmt.Printf("[DD] %d, Send: G:%d, CH: %d, EvID: %d, Val:%d\n", ev.Ts, ev.G, ev.Args[1], ev.Args[0], ev.Args[2])
			}
			sends.PushBack(ev)
		case trace.EvGoRecv:
			if debug {
				fmt.Printf("[DD] %d, Recv: G:%d, CH: %d, EvID: %d, Val:%d - %v\n", ev.Ts, ev.G, ev.Args[1], ev.Args[0], ev.Args[2], ev)
			}
			send := findSource(sends, ev)
			if send == nil {
				// it's either channel close() or error in findSource
				continue
			}
			if debug {
				fmt.Printf("[DD] %d, Recv->Send: FromG:%d, ToG: %d, CH: %d, EvID: %d, Val:%d\n", send.Ts, send.G, ev.G, ev.Args[1], ev.Args[0], ev.Args[2])
			}
			c.ChanSend(send.Ts, ev.Ts, ev.Args[1], send.G, ev.G, send.Args[2])
		case trace.EvGCStart, trace.EvGCDone, trace.EvGCSweepStart, trace.EvGCSweepDone:
			if debug {
				fmt.Println(ev.Ts, "GoGC...", ev.Type, ev.Args)
			}
			lastG = 1
		case trace.EvGoSched, trace.EvGoPreempt,
			trace.EvGoBlock, trace.EvGoBlockSelect, trace.EvGoBlockSend, trace.EvGoBlockRecv,
			trace.EvGoBlockSync, trace.EvGoBlockCond, trace.EvGoBlockNet,
			trace.EvGoSysBlock:
			if debug {
				fmt.Println("[DD] Block:", ev.Ts, ev.G, ev.Args)
			}
			c.BlockGoroutine(ev.Ts, ev.G)
		case trace.EvGoSleep:
			if debug {
				fmt.Println("[DD] Sleep:", ev.Ts, ev.G, ev.Args)
			}
			c.SleepGoroutine(ev.Ts, ev.G)
		case trace.EvGoStop:
			if debug {
				fmt.Println(ev.Ts, "GoStop:", ev.G)
			}
			lastG = 1
		default:
			if debug {
				fmt.Println(ev.Ts, "Ev:", ev.Type, ev.G, ev.Args)
			}
		}
	}

	// sort events
	sort.Sort(ByTimestamp(c.cmds))

	// insert stop main
	// TODO: figure out why it's not in the trace
	lastTs := c.cmds[len(c.cmds)-1].Time
	c.StopGoroutine(lastTs+10, "", 1)

	return c, nil
}

// findSource tries to find corresponding Send event to ev.
func findSource(sends *list.List, ev *trace.Event) *trace.Event {
	for e := sends.Back(); e != nil; e = e.Prev() {
		send := e.Value.(*trace.Event)
		if send.Args[1] == ev.Args[1] && send.Args[0] == ev.Args[0] {
			sends.Remove(e)
			return send
		}
	}
	return nil
}
