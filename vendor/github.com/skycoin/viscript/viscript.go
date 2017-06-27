/*

------- NEXT THINGS TODO: -------

* RPC cli:
	add functionality to print running jobs for a given process id
	that can be retrieved by lp or setting the process id as default
	because that already exists

* ExternalProcess:
	Ctrl + c - detach, delete, kill probably
	Ctrl + z - detach and let it be running or pause it (https://repl.it/GeGn/1)?,
	jobs - list all jobs of current terminal
	fg <id> - send to foreground

* auto-run task_ext according to os specific init
	(doing it immediately upon first cli submission good enough?)

* make current command line autoscroll horizontally
	* make it optional (if turned off, always truncate the left)

* back buffer scrolling
	* pgup/pgdn hotkeys
	* 1-3 lines with scrollwheel

* Fix getting a resizing pointer outside of focused terminal.
		When you click outside terminal it can land on a background
		terminal which then pops in front.  Blocking the resize

* Sideways scroll command line when it doesn't fit the dedicated space for it
		(atm, 2 lines are reserved along the bottom of a full screen)
		* block character at end to indicate continuing on next line

* make new window display on top
		(i believe the sorting logic is only triggered by clicking right now)

* scan and do/fix most FIXME/TODO places in the code



------- OLDER TODO: ------- (everything below was for the text editor)

* KEY-BASED NAVIGATION
	* CTRL-HOME/END - PGUP/DN
* BACKSPACE/DELETE at the ends of lines
	pulls us up to prev line, or pulls up next line
* when auto appending to the end of a terminal, scroll all the way down
		(manual activity in the middle could increase size, so do this only when appending to body)


------- LOWER PRIORITY POLISH: -------

* if cursor movement goes past left/right of screen, auto-horizontal-scroll as you type
* same for when newlines/enters/returns push cursor past the bottom of visible space
* vertical scrollbars could have a smaller rendering of the first ~40 chars?
		however not if we map the whole vertical space (when scrollspace is taller than screen),
		because this requires scaling the text.  and keeping the aspect ratio means ~40 (max)
		would alter the width of the scrollbar
* when there is no scrollbar, should be able to see/interact with text in that area

*/

package main

import (
	"os"

	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/config"
	"github.com/skycoin/viscript/hypervisor"
	"github.com/skycoin/viscript/monitor"
	"github.com/skycoin/viscript/rpc/terminalmanager"
	"github.com/skycoin/viscript/viewport"
)

func main() {
	app.MakeHighlyVisibleLogEntry(app.Name, 15)

	err := config.Load("config.yaml")
	if err != nil {
		println(err.Error())
		return
	}

	args := os.Args[1:]
	if len(args) == 1 {
		if args[0] == "-h" || args[0] == "-run_headless" {
			//override the defalt run headless no matter what it's value
			config.Global.Settings.RunHeadless = true
		}
	}

	println("RunHeadless:", config.Global.Settings.RunHeadless)

	hypervisor.Init()
	viewport.Init() //runtime.LockOSThread()
	//rpc concurrency can interrupt the following, so printing NOW
	app.MakeHighlyVisibleLogEntry("Start loop", 7)

	go func() {
		rpcInstance := terminalmanager.NewRPC()
		rpcInstance.Serve()
	}()

	monitor.Init("0.0.0.0:7999").Run() //tcp server monitor for apps

	//actual start of loop
	for viewport.CloseWindow == false {
		viewport.DispatchEvents() //event channel

		hypervisor.TickTasks()
		hypervisor.TickExtTasks()

		viewport.PollUiInputEvents()
		viewport.Tick()
		viewport.UpdateDrawBuffer()
		viewport.SwapDrawBuffer() //with new frame
	}

	viewport.TeardownScreen()
	hypervisor.Teardown()
}
