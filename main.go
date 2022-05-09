package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/nvkalinin/business-calendar/cmd"
	"github.com/nvkalinin/business-calendar/log"
	"os"
)

type CLI struct {
	Debug bool `short:"d" long:"debug" env:"DEBUG" description:"Выводить отладочные сообщения в лог."`

	Server cmd.Server `command:"server" description:"Запустить сервер (rest + периодическая синхронизация)."`
	Sync   cmd.Sync   `command:"sync" description:"Синхронизировать календарь за указанный год."`
	Backup cmd.Backup `command:"backup" description:"Сделать резервную копию хранилища bolt."`
}

func main() {
	cli := &CLI{}
	parser := flags.NewParser(cli, flags.Default)
	parser.CommandHandler = func(cmd flags.Commander, args []string) error {
		log.AllowDebug = cli.Debug

		if cmd != nil {
			return cmd.Execute(args)
		}
		return nil
	}

	if _, err := parser.Parse(); err != nil {
		flagsErr, isFlagsErr := err.(flags.ErrorType)
		if isFlagsErr && flagsErr == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
