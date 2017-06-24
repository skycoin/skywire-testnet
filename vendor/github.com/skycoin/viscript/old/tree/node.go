package tree

import (
//"fmt"
)

type Node struct {
	Text     string
	ChildIdL int // branching to left
	ChildIdR int // branching to right
	ParentId int
}
