package utils

import (
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var (
	DataDirFlag = PathFlag{
		Name:  "dataDir",
		Usage: "Data directory for the databases ",
		Value: PathString{"./"},
	}

	KeyDirFlag = PathFlag{
		Name:  "keyDir",
		Usage: "Directory for the wallet key (default = inside the dataDir)",
	}

	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}

	RPCApiFlag = cli.StringFlag{
		Name:  "rpcApi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
	}

	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}

	P2PPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 30303,
	}

	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "cfg.toml path to change the default path",
		Value: "",
	}

	TypeFlag = cli.StringFlag{
		Name:  "type",
		Usage: "launcher type",
		Value: "",
	}

	CountFlag = cli.IntFlag{
		Name:  "count",
		Usage: "launcher count",
		Value: 1,
	}

	IPFlag = cli.StringFlag{
		Name:  "IP",
		Usage: "launcher IP",
		Value: "",
	}
)

// MigrateFlags sets the global flag from a local flag when it's set.
// This is a temporary function used for migrating old command/flags to the new format.
func MigrateFlags(action func(ctx *cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		for _, name := range ctx.FlagNames() {
			if ctx.IsSet(name) {
				ctx.GlobalSet(name, ctx.String(name))
			}
		}
		return action(ctx)
	}
}

func MakeDataDir(ctx *cli.Context) string {
	if path := ctx.GlobalString(DataDirFlag.Name); path != "" {
		return path
	}
	return ""
}

func GetConfigDir(ctx *cli.Context) string {
	if path := ctx.GlobalString(ConfigFileFlag.Name); path != "" {
		return path
	}
	return ""
}

func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}
