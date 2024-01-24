package main

import (
	"log"
	"os"

	"github.com/b1naryth1ef/carto"
	"github.com/b1naryth1ef/carto/build"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:        "carto",
		Description: "minecraft web-based map generator",
		Commands: []*cli.Command{

			{
				Name:   "build",
				Action: commandBuild,
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:  "config",
						Usage: "path to the configuration file",
						Value: "config.hcl",
					},
					&cli.BoolFlag{
						Name:  "clean",
						Usage: "force a clean build ignoring chunk modification time data",
						Value: false,
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func commandBuild(ctx *cli.Context) error {
	config, err := carto.LoadConfig(ctx.Path("config"))
	if err != nil {
		return err
	}

	return build.Build(config, build.BuildOpts{
		ForceClean: ctx.Bool("clean"),
	})
}
