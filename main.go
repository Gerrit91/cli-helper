package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name: "decode-secret",
				Action: func(c *cli.Context) error {
					if !c.Args().Present() {
						return fmt.Errorf("no key arg provided")
					}

					return decodeSecret(c.Args().First())
				},
			},
			{
				Name: "decode-jwt",
				Action: func(c *cli.Context) error {
					return decodeJWT()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

}

func decodeSecret(key string) error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	type secret struct {
		Data map[string]string `json:"data"`
	}

	s := &secret{}

	err = yaml.Unmarshal(raw, s)
	if err != nil {
		return err
	}

	encoded, ok := s.Data[key]
	if !ok {
		return fmt.Errorf("secret does not contain data beneath key %q", key)
	}

	value, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	fmt.Println(string(value))

	return nil
}

func decodeJWT() error {
	var (
		decodeJSON = func(raw string) (string, error) {
			decoded, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				return "", err
			}

			parsed := map[string]any{}

			err = json.Unmarshal(decoded, &parsed)
			if err != nil {
				return "", err
			}

			formatted, err := json.MarshalIndent(parsed, "", "    ")
			if err != nil {
				return "", err
			}

			return string(formatted), nil
		}
	)

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	parts := strings.Split(string(input), ".")

	for i := range parts {
		switch i {
		case 0:
			data, err := decodeJSON(parts[i])
			if err != nil {
				return err
			}
			fmt.Println("Header (Algorithm & Token Type):")

			fmt.Println(data)
		case 1:
			data, err := decodeJSON(parts[i])
			if err != nil {
				return err
			}
			fmt.Println("Payload:")
			fmt.Println(data)
		case 2:
			// what to do with signature?
		}
	}

	return nil
}
