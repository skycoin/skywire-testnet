package terminal

import (
	"fmt"

	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/hypervisor"
	"github.com/skycoin/viscript/hypervisor/dbus"
	"github.com/skycoin/viscript/hypervisor/input/keyboard"
	termTask "github.com/skycoin/viscript/hypervisor/process/terminal"
	"github.com/skycoin/viscript/msg"
	"github.com/skycoin/viscript/viewport/gl"
)

var Terms = TerminalStack{}

type TerminalStack struct {
	FocusedId msg.TerminalId
	Focused   *Terminal
	Terms     map[msg.TerminalId]*Terminal

	//private
	//next/new terminal spawn vars
	nextRect   app.Rectangle
	nextDepth  float32
	nextOffset app.Vec2F // how far from previous terminal
}

func (ts *TerminalStack) Init() {
	w := gl.CanvasExtents.X * 1.5 //width of terminal window
	h := gl.CanvasExtents.Y * 1.5 //height

	ts.Terms = make(map[msg.TerminalId]*Terminal)
	ts.nextOffset.X = (gl.CanvasExtents.X*2 - w) / 2
	ts.nextOffset.Y = (gl.CanvasExtents.Y*2 - h) / 2

	ts.nextRect = app.Rectangle{
		gl.CanvasExtents.Y,
		-gl.CanvasExtents.X + w,
		gl.CanvasExtents.Y - h,
		-gl.CanvasExtents.X}

	//setup a starter terminal window
	Terms.Add()
}

func (ts *TerminalStack) Add() msg.TerminalId {
	println("<TerminalStack>.Add()")

	ts.nextDepth += ts.nextOffset.X / 10 // done first, cuz desktop is at 0

	tid := msg.RandTerminalId() //terminal id
	ts.Terms[tid] = &Terminal{
		Depth: ts.nextDepth,
		Bounds: &app.Rectangle{
			ts.nextRect.Top,
			ts.nextRect.Right,
			ts.nextRect.Bottom,
			ts.nextRect.Left}}
	ts.Terms[tid].Init()
	ts.FocusedId = tid
	ts.Focused = ts.Terms[tid]

	ts.nextRect.Top -= ts.nextOffset.Y
	ts.nextRect.Right += ts.nextOffset.X
	ts.nextRect.Bottom -= ts.nextOffset.Y
	ts.nextRect.Left += ts.nextOffset.X

	ts.SetupTerminal(tid)
	return tid
}

func (ts *TerminalStack) RemoveTerminal(id msg.TerminalId) {
	println("<TerminalStack>.RemoveTerminal() ---------------------------- FIXME/TODO")
	// TODO: FIXME: what should happen here after deleting terminal from the stack?
	// delete(ts.Terms, id)
}

func (ts *TerminalStack) Tick() {
	for _, term := range ts.Terms {
		term.Tick()
	}
}

func (ts *TerminalStack) MoveFocusedTerminal(hiResDelta app.Vec2F, mouseDeltaSinceClick *app.Vec2F) {
	d := mouseDeltaSinceClick
	cs := ts.Focused.CharSize
	fb := ts.Focused.Bounds

	if keyboard.ControlKeyIsDown { //smooth, high resolution
		fb.MoveBy(hiResDelta)
	} else { //snap movement to char size
		if d.X > cs.X {
			d.X -= cs.X
			fb.MoveBy(app.Vec2F{cs.X, 0})
		} else if d.X < -cs.X {
			d.X += cs.X
			fb.MoveBy(app.Vec2F{-cs.X, 0})
		}

		if d.Y > cs.Y {
			d.Y -= cs.Y
			fb.MoveBy(app.Vec2F{0, cs.Y})
		} else if d.Y < -cs.Y {
			d.Y += cs.Y
			fb.MoveBy(app.Vec2F{0, -cs.Y})
		}
	}
}

func (ts *TerminalStack) SetupTerminal(termId msg.TerminalId) {
	//make it's task
	task := termTask.MakeNewTask()
	tskIF := msg.ProcessInterface(task)
	tskId := hypervisor.AddProcess(tskIF)

	task.State.VisualInfo = ts.Terms[termId].GetVisualInfo()

	/* the rest is all DBUS related */

	//terminal
	rid1 := fmt.Sprintf("dbus.pubsub.terminal-%d", int(termId)) //ResourceIdentifier
	tcid := hypervisor.DbusGlobal.CreatePubsubChannel(          //terminal channel id
		dbus.ResourceId(termId),   //owner id
		dbus.ResourceTypeTerminal, //owner type
		rid1)

	//process
	rid2 := fmt.Sprintf("dbus.pubsub.process-%d", int(tskId)) //ResourceIdentifier
	pcid := hypervisor.DbusGlobal.CreatePubsubChannel(        //process channel id
		dbus.ResourceId(tskId),   //owner id
		dbus.ResourceTypeProcess, //owner type
		rid2)

	task.OutChannelId = uint32(tcid)
	ts.Terms[termId].OutChannelId = uint32(pcid)
	ts.Terms[termId].AttachedProcess = tskId

	//subscribe process to the terminal id
	hypervisor.DbusGlobal.AddPubsubChannelSubscriber(
		tcid,
		dbus.ResourceId(tskId),
		dbus.ResourceTypeProcess,
		ts.Terms[termId].InChannel)

	//subscribe terminal to the process id
	hypervisor.DbusGlobal.AddPubsubChannelSubscriber(
		pcid,
		dbus.ResourceId(termId),
		dbus.ResourceTypeTerminal,
		tskIF.GetIncomingChannel())
}

func (ts *TerminalStack) SetFocused(topmostId msg.TerminalId) {
	//store which is focused and bring it to top
	newZ := float32(9.9) //FIXME (for all uses of this var, IF you ever want more than (about) 50 terms)
	ts.FocusedId = topmostId
	ts.Focused = ts.Terms[topmostId]
	ts.Focused.Depth = newZ

	//store the REST of the terms
	theRest := []*Terminal{}

	for id, t := range ts.Terms {
		if id != ts.FocusedId {
			theRest = append(theRest, t)
		}
	}

	//sort them (top/closest at the start of list)
	fullySorted := false
	for !fullySorted {
		fullySorted = true

		for i := 0; i < len(theRest)-1; i++ {
			if theRest[i].Depth < theRest[i+1].Depth {
				theNext := theRest[i+1]
				theRest[i+1] = theRest[i]
				theRest[i] = theNext
				fullySorted = false
			}
		}
	}

	//assign receding z/depth values
	for _, t := range theRest {
		newZ -= 0.2
		t.Depth = newZ
	}
}

func (ts *TerminalStack) Defocus() {
	ts.FocusedId = 0
	ts.Focused = nil
}
