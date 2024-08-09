package kubernetes

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"
)

func DecodeSecret(c *cli.Context) error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	type secret map[string]any
	s := secret{}

	err = yaml.Unmarshal(raw, &s)
	if err != nil {
		return err
	}

	data, ok := s["data"]
	if !ok {
		return fmt.Errorf("secret does not contain data field")
	}

	d := data.(map[string]any)
	for k, v := range d {
		v := v.(string)

		value, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}

		d[k] = string(value)
	}

	s["data"] = d

	var output []byte

	if c.Bool("entire-secret") {
		output, err = yaml.Marshal(s)
		if err != nil {
			return err
		}
	} else {
		output, err = yaml.Marshal(d)
		if err != nil {
			return err
		}
	}

	fmt.Println(string(output))

	return nil
}

func DecodeSecretKey(key string) error {
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
