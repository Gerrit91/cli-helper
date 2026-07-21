package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Gerrit91/cli-helper/pkg/battery"
	"github.com/Gerrit91/cli-helper/pkg/jwt"
	"github.com/Gerrit91/cli-helper/pkg/kubernetes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	var (
		rootCmd = &cobra.Command{
			Use:   "ch",
			Short: "a small cli-helper tool",
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				must(viper.BindPFlags(cmd.Flags()))
				must(viper.BindPFlags(cmd.PersistentFlags()))
			},
		}

		decodeSecretCmd = &cobra.Command{
			Use:               "decode-secret",
			Short:             "decodes a b64-encoded kubernetes secret",
			ValidArgsFunction: secretNameCompletion,
			RunE: func(cmd *cobra.Command, args []string) error {
				var secretName string
				if len(args) == 1 {
					secretName = args[0]
				}

				return kubernetes.DecodeSecret(cmd.Context(),
					viper.GetString("context"),
					viper.GetString("namespace"),
					secretName,
					viper.GetBool("entire-secret"),
				)
			},
		}

		decodeJwtCmd = &cobra.Command{
			Use:   "decode-jwt",
			Short: "decodes a jwt",
			RunE: func(cmd *cobra.Command, args []string) error {
				return jwt.DecodeJWT()
			},
		}

		batteryDaemonCmd = &cobra.Command{
			Use:   "battery-daemon",
			Short: "runs a daemon that listens on dbus for upower events and sends notifications and sounds",
			RunE: func(cmd *cobra.Command, args []string) error {
				return battery.New(viper.GetBool("daemonize")).Run(cmd.Context())
			},
		}

		gardenerReconcile = &cobra.Command{
			Use:               "reconcile",
			Short:             "adds the gardener.cloud/operation=reconcile annotation to the given resource",
			ValidArgsFunction: resourceCompletion,
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) != 2 {
					return errors.New("arguments must be <resource> <name>")
				}

				return kubernetes.AnnotateResource(
					cmd.Context(),
					viper.GetString("context"),
					args[0],
					viper.GetString("namespace"),
					args[1],
					"gardener.cloud/operation=reconcile",
				)
			},
		}
		gardenerMaintain = &cobra.Command{
			Use:               "maintain",
			Short:             "adds the gardener.cloud/operation=maintain annotation to the given resource",
			ValidArgsFunction: resourceCompletion,
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) != 2 {
					return errors.New("arguments must be <resource> <name>")
				}

				return kubernetes.AnnotateResource(
					cmd.Context(),
					viper.GetString("context"),
					args[0],
					viper.GetString("namespace"),
					args[1],
					"gardener.cloud/operation=maintain",
				)
			},
		}
	)

	decodeSecretCmd.Flags().StringP("namespace", "n", "", "the namespace of the secret")
	decodeSecretCmd.Flags().Bool("entire-secret", false, "prints out the entire secret instead of the data section only")

	must(decodeSecretCmd.RegisterFlagCompletionFunc("namespace", namespaceCompletion))

	batteryDaemonCmd.Flags().BoolP("daemonize", "d", false, "runs a background daemon that watches for upower dbus events")

	gardenerReconcile.Flags().String("context", "", "the kubeconfig context")
	gardenerReconcile.Flags().StringP("namespace", "n", "", "the namespace of the resource")
	must(gardenerReconcile.RegisterFlagCompletionFunc("context", contextCompletion))
	must(gardenerReconcile.RegisterFlagCompletionFunc("namespace", namespaceCompletion))

	gardenerMaintain.Flags().String("context", "", "the kubeconfig context")
	gardenerMaintain.Flags().StringP("namespace", "n", "", "the namespace of the resource")
	must(gardenerMaintain.RegisterFlagCompletionFunc("context", contextCompletion))
	must(gardenerMaintain.RegisterFlagCompletionFunc("namespace", namespaceCompletion))

	rootCmd.AddCommand(
		decodeSecretCmd,
		decodeJwtCmd,
		batteryDaemonCmd,
		gardenerReconcile,
		gardenerMaintain,
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func secretNameCompletion(c *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	args := []string{"get", "secrets", "--no-headers", "-o", "custom-columns=:metadata.name"}
	if ns, _ := c.Flags().GetString("namespace"); ns != "" {
		args = append(args, "-n", ns)
	}
	if context, _ := c.Flags().GetString("context"); context != "" {
		args = append(args, "--context", context)
	}

	cmd := exec.CommandContext(c.Context(), "kubectl", args...)
	cmd.Env = os.Environ()

	raw, err := cmd.Output()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return strings.Split(string(raw), "\n"), cobra.ShellCompDirectiveNoFileComp
}

func namespaceCompletion(c *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	args := []string{"get", "ns", "--no-headers", "-o", "custom-columns=:metadata.name"}
	if context, _ := c.Flags().GetString("context"); context != "" {
		args = append(args, "--context", context)
	}

	cmd := exec.CommandContext(c.Context(), "kubectl", args...)
	cmd.Env = os.Environ()

	raw, err := cmd.Output()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return strings.Split(string(raw), "\n"), cobra.ShellCompDirectiveNoFileComp
}

func contextCompletion(c *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cmd := exec.CommandContext(c.Context(), "kubectl", "config", "get-contexts", "-o", "name")
	cmd.Env = os.Environ()

	raw, err := cmd.Output()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return strings.Split(string(raw), "\n"), cobra.ShellCompDirectiveNoFileComp
}

func resourceCompletion(c *cobra.Command, cArgs []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(cArgs) {
	case 0:
		args := []string{"api-resources", "-o", "json"}
		if context, _ := c.Flags().GetString("context"); context != "" {
			args = append(args, "--context", context)
		}

		cmd := exec.CommandContext(c.Context(), "kubectl", args...)
		cmd.Env = os.Environ()

		raw, err := cmd.Output()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		type api struct {
			Resources []struct {
				Name       string   `json:"name"`
				ShortNames []string `json:"shortNames"`
			} `json:"resources"`
		}

		var apiResources api

		if err = json.Unmarshal(raw, &apiResources); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var res []string

		for _, resource := range apiResources.Resources {
			res = append(res, resource.Name)
			res = append(res, resource.ShortNames...)
		}

		return res, cobra.ShellCompDirectiveNoFileComp
	case 1:
		args := []string{"get", cArgs[0], "--no-headers", "-o", "custom-columns=:metadata.name"}
		if ns, _ := c.Flags().GetString("namespace"); ns != "" {
			args = append(args, "-n", ns)
		}
		if context, _ := c.Flags().GetString("context"); context != "" {
			args = append(args, "--context", context)
		}

		cmd := exec.CommandContext(c.Context(), "kubectl", args...)
		cmd.Env = os.Environ()

		raw, err := cmd.Output()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return strings.Split(string(raw), "\n"), cobra.ShellCompDirectiveNoFileComp
	default:
		return nil, cobra.ShellCompDirectiveError
	}
}
