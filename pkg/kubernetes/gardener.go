package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"sigs.k8s.io/yaml"
)

func VirtualGardenKubeconfig(ctx context.Context) error {
	type (
		secret struct {
			Metadata struct {
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Data map[string]string `json:"data"`
		}
		garden struct {
			Spec struct {
				VirtualCluster struct {
					Dns struct {
						Domains []struct {
							Name string `json:"name"`
						} `json:"domains"`
					} `json:"dns"`
				} `json:"virtualCluster"`
			} `json:"spec"`
		}

		list[A any] struct {
			Items []A `json:"items"`
		}

		kubeconfig struct {
			Clusters []struct {
				Cluster struct {
					CA     string `json:"certificate-authority-data"`
					Server string `json:"server"`
				} `json:"cluster"`
				Name string `json:"name"`
			} `json:"clusters"`
			Users []struct {
				User struct {
					Token string `json:"token"`
				} `json:"user"`
			} `json:"users"`
		}
	)

	cmd := exec.CommandContext(ctx, "kubectl", "get", "secret", "-n", "garden", "shoot-access-virtual-garden", "-o", "json")
	cmd.Env = os.Environ()

	raw, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error retrieving shoot-access-virtual-garden secret (%w): %s", err, string(raw))
	}

	var shootAccessSecret secret
	if err := json.Unmarshal(raw, &shootAccessSecret); err != nil {
		return err
	}

	token, err := base64.StdEncoding.DecodeString(shootAccessSecret.Data["token"])
	if err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "kubectl", "get", "secret", "-n", "garden", "-l", "managed-by=secrets-manager,manager-identity=gardener-operator,name=generic-token-kubeconfig", "-o", "json")
	cmd.Env = os.Environ()

	raw, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error retrieving generic-token-kubeconfig secrets (%w): %s", err, string(raw))
	}

	var genericSecrets list[secret]
	if err := json.Unmarshal(raw, &genericSecrets); err != nil {
		return fmt.Errorf("unable to unmarshal generic secrets: %w", err)
	}

	var genericSecret *secret
	for _, secret := range genericSecrets.Items {
		if genericSecret == nil {
			genericSecret = &secret
		} else if genericSecret.Metadata.Labels["issued-at-time"] < secret.Metadata.Labels["issued-at-time"] {
			genericSecret = &secret
		}
	}

	if genericSecret == nil {
		return fmt.Errorf("no generic kubeconfig secret found for accessing virtual garden: %w", err)
	}

	genericKubeconfigRaw, err := base64.StdEncoding.DecodeString(genericSecret.Data["kubeconfig"])
	if err != nil {
		return fmt.Errorf("unable to decode generic garden kubeconfig: %w", err)
	}

	var genericKubeconfig kubeconfig
	if err := yaml.Unmarshal(genericKubeconfigRaw, &genericKubeconfig); err != nil {
		return fmt.Errorf("unable to unmarshal generic garden kubeconfig: %w", err)
	}

	cmd = exec.CommandContext(ctx, "kubectl", "get", "garden", "-o", "json")
	cmd.Env = os.Environ()

	raw, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error retrieving generic-token-kubeconfig secrets (%w): %s", err, string(raw))
	}

	var gardens list[garden]
	if err := json.Unmarshal(raw, &gardens); err != nil {
		return fmt.Errorf("unable to unmarshal garden: %w", err)
	}

	if len(gardens.Items) == 0 {
		return fmt.Errorf("no garden resource found")
	}

	gardenResource := gardens.Items[0]

	f, err := os.CreateTemp(os.TempDir(), "ch")
	if err != nil {
		return fmt.Errorf("unable to create tmp file: %w", err)
	}
	defer func() {
		_ = os.Remove(f.Name())
		_ = f.Close()
	}()

	decodedCA, err := base64.StdEncoding.DecodeString(genericKubeconfig.Clusters[0].Cluster.CA)
	if err != nil {
		return fmt.Errorf("unable to decode ca: %w", err)
	}

	_, err = f.Write(decodedCA)
	if err != nil {
		return fmt.Errorf("unable to write ca data to tmp file: %w", err)
	}

	for _, cmd := range []*exec.Cmd{
		exec.CommandContext(ctx, "kubectl", "config", "set-cluster", "garden", "--server", "https://api."+gardenResource.Spec.VirtualCluster.Dns.Domains[0].Name),
		exec.CommandContext(ctx, "kubectl", "config", "set-cluster", "garden", "--embed-certs", "--certificate-authority", f.Name()),
		exec.CommandContext(ctx, "kubectl", "config", "set-credentials", "garden", "--token", string(token)),
		exec.CommandContext(ctx, "kubectl", "config", "set-context", "garden", "--cluster", "garden", "--user", "garden", "--namespace", "garden"),
	} {
		cmd.Env = os.Environ()

		raw, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error setting kubeconfig entry %s: %v", cmd.String(), string(raw))
		}
	}

	return nil
}
