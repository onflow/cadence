package main

import (
	"github.com/dapperlabs/cadence/languageserver/server"
)

func main() {
	server.NewServer().Start()
}
