package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"sigs.k8s.io/yaml"
)

func DecodeSecret(ctx context.Context, context, namespace, name string, entireSecret bool) error {
	var raw []byte

	if name != "" {
		var (
			err  error
			args = []string{"get", "secret", name, "-o", "yaml"}
		)

		if context != "" {
			args = append(args, "--context", context)
		}
		if namespace != "" {
			args = append(args, "-n", namespace)
		}

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

	if entireSecret {
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

func AnnotateResource(ctx context.Context, context, resource, namespace, name, annotation string) error {
	var (
		err  error
		args = []string{"annotate", resource, name}
	)

	if context != "" {
		args = append(args, "--context", context)
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	args = append(args, annotation)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = os.Environ()

	raw, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error annotating resource (%w): %s", err, string(raw))
	}

	fmt.Println(strings.TrimSpace(string(raw)))

	return nil
}
