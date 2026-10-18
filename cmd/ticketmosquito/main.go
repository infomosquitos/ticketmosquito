package main

import (
	"os"

	"github.com/infomosquitos/ticketmosquito/internal/cliapp"
)

func main() {
	os.Exit(cliapp.Run(os.Args))
}
