package main

import (
	"os"

	"github.com/UFFeScience/akoflow/internal/cli/commands/cli_service"
)

func main() {

	command := os.Args[1]

	cliService := cli_service.New(command)
	cliService.Run()

}
