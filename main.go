package main

import (
	"fmt"
	"os"

	"github.com/Gerrit91/cli-helper/pkg/jwt"
	"github.com/Gerrit91/cli-helper/pkg/kubernetes"
	"github.com/Gerrit91/cli-helper/pkg/weather"

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
				Name: "weather",
				Action: func(c *cli.Context) error {
					w, err := weather.New(c.String("cache-path"), c.String("location"), c.String("api-token-path"))
					if err != nil {
						return err
					}

					return w.PrintForWaybar()
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "api-token-path",
						DefaultText: "open weather api token",
						Required:    true,
						EnvVars:     []string{"OPEN_WEATHER_API_TOKEN"},
					},
					&cli.StringFlag{
						Name:        "location",
						DefaultText: "name of the location to query",
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "cache-path",
						DefaultText: "the path where to store the cached weather data",
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
