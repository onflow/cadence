package sema

// NeverType represents the bottom type
var NeverType = &NominalType{
	Name:                 "Never",
	QualifiedName:        "Never",
	TypeID:               "Never",
	IsInvalid:            false,
	IsResource:           false,
	Storable:             false,
	Equatable:            false,
	ExternallyReturnable: false,
	IsSuperTypeOf:        nil,
}
