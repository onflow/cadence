package common

import (
	"github.com/dapperlabs/cadence/runtime/errors"
)

//go:generate stringer -type=PathDomain

type PathDomain int

const (
	PathDomainUnknown PathDomain = iota
	PathDomainStorage
	PathDomainPrivate
	PathDomainPublic
)

var AllPathDomainsByIdentifier = map[string]PathDomain{}

func init() {
	for _, pathDomain := range []PathDomain{
		PathDomainStorage,
		PathDomainPrivate,
		PathDomainPublic,
	} {
		identifier := pathDomain.Identifier()
		AllPathDomainsByIdentifier[identifier] = pathDomain
	}
}

func PathDomainFromIdentifier(domain string) PathDomain {
	result, ok := AllPathDomainsByIdentifier[domain]
	if !ok {
		return PathDomainUnknown
	}
	return result
}

func (i PathDomain) Identifier() string {
	switch i {
	case PathDomainStorage:
		return "storage"

	case PathDomainPrivate:
		return "private"

	case PathDomainPublic:
		return "public"
	}

	panic(errors.NewUnreachableError())
}

func (i PathDomain) Name() string {
	switch i {
	case PathDomainStorage:
		return "storage"

	case PathDomainPrivate:
		return "private"

	case PathDomainPublic:
		return "public"
	}

	panic(errors.NewUnreachableError())
}
