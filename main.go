package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Gerrit91/cli-helper/pkg/battery"
	"github.com/Gerrit91/cli-helper/pkg/jwt"
	"github.com/Gerrit91/cli-helper/pkg/kubernetes"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name: "decode-secret",
				Action: func(ctx context.Context, c *cli.Command) error {
					return kubernetes.DecodeSecret(ctx, c)
				},
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "secret-name",
					},
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
				Action: func(ctx context.Context, c *cli.Command) error {
					return jwt.DecodeJWT()
				},
			},
			{
				Name:        "battery-daemon",
				Description: "runs a daemon that listens on dbus for upower events and sends notifications and sounds",
				Action: func(ctx context.Context, c *cli.Command) error {
					return battery.New(c.Bool("daemonize")).Run(ctx)
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

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
