package compiler

type RewriteRule struct {
	Name    string
	Pattern *Pattern
	Replace func(*Builder, *Value) *Value
}
