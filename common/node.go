package common

import "sync/atomic"

type NodeID uint32

var nodeIDCounter uint32

func NextNodeID() NodeID {
	return NodeID(atomic.AddUint32(&nodeIDCounter, 1))
}

type NodeIDHolder struct{ id NodeID }

func NewNodeIDHolder() NodeIDHolder { return NodeIDHolder{id: NextNodeID()} }
func (n NodeIDHolder) ID() NodeID   { return n.id }
