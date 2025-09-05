package commons

import "sort"

type GlobalCategory int

const (
	ContractGlobal GlobalCategory = iota
	VariableGlobal
	FunctionGlobal
)

type SortableGlobal interface {
	GetCategory() GlobalCategory
	GetName() string
	SetIndex(index uint16)
}

func SortGlobals(globals []SortableGlobal) {
	sort.Slice(globals, func(i, j int) bool {
		if globals[i].GetCategory() != globals[j].GetCategory() {
			return globals[i].GetCategory() < globals[j].GetCategory()
		}
		return globals[i].GetName() < globals[j].GetName()
	})
}
