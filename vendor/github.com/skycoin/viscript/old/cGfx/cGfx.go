package cGfx

/*
------------- STARTING UP A NEW *CLEAN* VERSION OF GFX -------------

.....which (ATM) has only "app" as a dependency
("app" has NONE)
..... [UPDATE]: added ui for now, but it only has app & "math"

migrate things over here until we can delete/replace
the current "gfx" package with THIS

--------------------------------------------------------------------
*/

// rectangle data soup
// var Rects []*app.PicRectangle

// func SetRect(r *app.PicRectangle) { // will add one if it doesn't exist yet
// 	if len(Rects) < 1 {
// 		// prepare for appending new rects
// 		// (won't use 0, leaving it as an id code for element being uninitialized)
// 		Rects = append(Rects, &app.PicRectangle{})
// 	}

// 	if r.Id == 0 {
// 		// TODO IF A RECYCLABLE POOL IS DESIRED, scan through and recycle
// 		// a RecState_Dead element rather than appending (if possible)
// 		r.Id = int32(len(Rects))
// 		Rects = append(Rects, r)
// 	} else {
// 		Rects[r.Id] = r
// 	}
// }
