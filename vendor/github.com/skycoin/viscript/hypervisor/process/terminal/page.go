package process

import (
	//"github.com/skycoin/viscript/hypervisor"
	"github.com/skycoin/viscript/msg"
)

func (st *State) makePageOfLog(m msg.MessageVisualInfo) {
	//app.At("process/terminal/msg_action", "makePageOfLog")

	//called by
	//* viewport/term.setupNewGrid()
	//* backscrolling actions

	st.VisualInfo = m
	println("st.VisualInfo.NumRows:", st.VisualInfo.NumRows)

	if //...there's not a full screenful in log yet
	st.VisualInfo.CurrRow <
		st.VisualInfo.NumRows-
			st.VisualInfo.PromptRows {

		//don't allow a setting that can't be used yet.
		//it would give no visual feedback anyways.
		//later it might (in a buggy way),
		//once some random state changes.
		//by that time, the user forgets they
		//might have backscrolled (they saw no scrolling)

		//st.Cli.BackscrollAmount = 0
	}

	ei /* entry index (of log) */ := len(st.Cli.Log) - 1
	page := []string{} //(screenful of visible text)

	//build a page (or less if term hasn't scrolled yet)
	usableRows := int(m.NumRows - m.PromptRows)
	for /* page isn't full & more entries */ len(page) < usableRows && ei >= 0 {
		tl /* temporary line to dissect */ := st.Cli.Log[ei]

		lineSections := []string{} //pieces of broken/divided-up lines

		x := int(m.NumColumns)
		for /* line needs breaking up */ len(tl) > int(m.NumColumns) {
			/* decrement towards start of word */
			for string(tl[x]) != " " &&
				/* still fits on 1 line */ (len(tl)-x) < int(m.NumColumns) {
				x--
			}

			lineSections = append(lineSections, tl[:x])
			tl = tl[x+1:]
		}

		//the last section, if anything remains
		if len(tl) > 0 {
			lineSections = append(lineSections, tl)
		}

		//add line or line sections to page
		for i := len(lineSections) - 1; i >= 0; i-- {
			page = append(page, lineSections[i])
		}

		ei--
	}

	if st.Cli.BackscrollAmount > len(page) {
		st.Cli.BackscrollAmount = len(page)
	}

	for i := len(page) - 1; i >= /*0*/ st.Cli.BackscrollAmount; i-- {
		st.printLnAndMAYBELogIt(page[i], false)
	}

	//it's nice to be able to see/interact with the command prompt even while
	//backscrolled
	st.Cli.EchoWholeCommand(st.proc.OutChannelId)
}
