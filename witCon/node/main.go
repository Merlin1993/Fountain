package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"runtime/debug"
	"witCon/common"
	"witCon/console"
	"witCon/log"
	"witCon/node/utils"
	"witCon/stat"
)

var (
	DefaultDataDir = "data"
)

var (
	app = NewApp("wit-consensus command line interface")
)

func init() {
	app.Action = wit
	app.HideVersion = true
	app.Copyright = "Copyright 2023"
	app.Commands = []cli.Command{
		{
			Action:      utils.MigrateFlags(wit),
			Name:        "start",
			Usage:       "Start an interactive JavaScript environment with config",
			Flags:       []cli.Flag{utils.ConfigFileFlag, utils.DataDirFlag},
			Category:    "CONSOLE COMMANDS",
			Description: ``,
		},
	}
	app.Flags = append(app.Flags, utils.ConfigFileFlag, utils.DataDirFlag)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func NewApp(usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = "snz"
	app.Usage = usage
	return app
}

func wit(ctx *cli.Context) error {
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("Invalid command: %q", args[0])
	}
	defer RecoverError()
	console, err := console.NewConsole()
	if err != nil {
		fmt.Println(err)
	}

	dataPath := utils.MakeDataDir(ctx)
	config := MakeConfig(ctx)
	stat.Instance.Init(config)
	name := fmt.Sprintf("%v_len-%v_con-%v_txam-%v_txs-%v", config.Name.String(), len(config.SaintList), config.Consensus, config.TxAmount, config.TxSize)
	fmt.Sprintf(fmt.Sprintf("%v", config))
	log.Create(dataPath, name, int(config.LogLvl))
	log.Info("start", "config", config, "data", dataPath)
	if config != nil {
		n := NewNode(config)
		n.Start(dataPath)
	}
	console.Interactive()
	return nil
}

func RecoverError() {
	if err := recover(); err != nil {
		//输出panic信息
		log.Error(fmt.Sprintf("%v", err))

		//输出堆栈信息
		log.Error(string(debug.Stack()))
	}
}

func MakeConfig(ctx *cli.Context) *common.Config {
	path := utils.GetConfigDir(ctx)
	if len(path) == 0 {
		fmt.Println("find no config path")
		path = common.DefaultConfigPath
	}
	return common.GetConfig(path)
}
