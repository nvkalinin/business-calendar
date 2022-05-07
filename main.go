package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/nvkalinin/business-calendar/cmd"
	"os"
)

type CLI struct {
	Server cmd.Server `command:"server" description:"Запустить сервер (rest + периодическая синхронизация)."`
	Sync   cmd.Sync   `command:"sync" description:"Синхронизировать календарь за указанный год."`
	Backup cmd.Backup `command:"backup" description:"Сделать резервную копию хранилища bolt."`
}

func main() {
	cli := &CLI{}
	parser := flags.NewParser(cli, flags.Default)

	if _, err := parser.Parse(); err != nil {
		flagsErr, isFlagsErr := err.(flags.ErrorType)
		if isFlagsErr && flagsErr == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
