package compiler

// Special wildcard op for use in pattern matching
var Op_ = DefOp("_")

type Pattern struct {
	Op      Op
	Args    []*Pattern
	ArgsDDD bool
	// Bind    int
}

// func (p *Pattern) Match(v *Value) {}
