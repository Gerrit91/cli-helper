package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func DecodeJWT(t string) error {
	var (
		decodeJSON = func(raw string) (string, error) {
			decoded, err := base64.RawStdEncoding.DecodeString(raw)
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

	if t == "" {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		t = string(input)
	}

	parts := strings.Split(t, ".")

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

	var (
		parser = jwt.NewParser()
		claims = jwt.MapClaims{}
	)

	token, _, err := parser.ParseUnverified(t, claims)
	if err != nil {
		return err
	}

	exp, err := token.Claims.GetExpirationTime()
	if err == nil {
		fmt.Println("Expires at: " + exp.String())
	}

	return nil
}
