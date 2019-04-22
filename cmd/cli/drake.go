package main

import (
	"fmt"
	"os"

	"github.com/lovethedrake/prototype/pkg/version"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "drake"
	app.Usage = "unified, container-aware task execution"
	app.Version = version.GetVersion()
	if version.GetCommit() != "" {
		app.Version = fmt.Sprintf("%s+%s", app.Version, version.GetCommit())
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagsFile,
			Usage: "specify the location of drake configuration",
			Value: "Drakefile.yaml",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "run",
			Usage:     "execute drake target(s) or pipeline(s)",
			UsageText: "drake run name... [options]",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  flagsPipeline,
					Usage: "execute a pipeline instead of a target",
				},
				cli.BoolFlag{
					Name:  flagsDebug,
					Usage: "display debug info",
				},
			},
			Action: run,
		},
	}
	fmt.Println()
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("%s\n\n", err)
		os.Exit(1)
	}
	fmt.Println()
}
