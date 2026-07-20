package main

import (
	"fmt"
	"os"

	"github.com/Gerrit91/cli-helper/pkg/battery"
	"github.com/Gerrit91/cli-helper/pkg/jwt"
	"github.com/Gerrit91/cli-helper/pkg/kubernetes"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name: "decode-secret",
				Action: func(c *cli.Context) error {
					if !c.Args().Present() {
						return kubernetes.DecodeSecret(c)
					}

					return kubernetes.DecodeSecretKey(c.Args().First())
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "entire-secret",
						DefaultText: "prints out the entire secret instead of the data section only",
						Value:       false,
					},
				},
			},
			{
				Name: "decode-jwt",
				Action: func(c *cli.Context) error {
					return jwt.DecodeJWT()
				},
			},
			{
				Name:        "battery-daemon",
				Description: "runs a daemon that listens on dbus for upower events and sends notifications and sounds",
				Action: func(c *cli.Context) error {
					return battery.New(c.Bool("daemonize")).Run(c.Context)
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "daemonize",
						Aliases:     []string{"d"},
						DefaultText: "runs a background daemon that watches for upower dbus events",
						Value:       false,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
