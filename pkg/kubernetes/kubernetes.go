package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
)

func DecodeSecret(ctx context.Context, c *cli.Command) error {
	var raw []byte

	if secretName := c.StringArg("secret-name"); secretName != "" {
		var (
			err  error
			args = []string{"get", "secret", secretName, "-o", "yaml"}
		)

		cmd := exec.CommandContext(ctx, "kubectl", args...)
		cmd.Env = os.Environ()

		raw, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error retrieving secret (%w): %s", err, string(raw))
		}

	} else {
		var err error

		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		if len(raw) == 0 {
			return fmt.Errorf("stdin was empty")
		}
	}

	type secret map[string]any
	s := secret{}

	err := yaml.Unmarshal(raw, &s)
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
