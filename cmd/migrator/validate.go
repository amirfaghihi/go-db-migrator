package main

import (
	"fmt"

	"github.com/amirfaghihi/migrator/jobspec"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var (
		file               string
		allowInlineSecrets bool
		resolve            bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check a job spec without connecting to anything",
		Long: "validate parses a job spec, rejects unknown keys and literal credentials,\n" +
			"and reports what the effective settings would be. It touches no network.\n\n" +
			"By default it does not resolve ${env:...} references either, so it is safe\n" +
			"to run in CI where no credentials are present. Pass --resolve to check that\n" +
			"the references themselves point at something.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			spec, resolved, err := jobspec.Load(file, jobspec.LoadOptions{
				AllowInlineSecrets: allowInlineSecrets,
				SkipResolve:        !resolve,
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s: ok\n\n", file)
			if spec.Name != "" {
				fmt.Fprintf(out, "  job               %s\n", spec.Name)
			}
			if resolve {
				fmt.Fprintf(out, "  source            %s\n", resolved.Source)
				fmt.Fprintf(out, "  sink              %s\n", resolved.Sink)
			} else {
				fmt.Fprintf(out, "  source            %s (unresolved)\n", spec.Source.DSN)
				fmt.Fprintf(out, "  sink              %s (unresolved)\n", spec.Sink.DSN)
			}
			fmt.Fprintf(out, "  objects           %d\n", len(spec.Objects))
			fmt.Fprintf(out, "  mode              %s\n", spec.Defaults.Mode)
			fmt.Fprintf(out, "  batch size        %s\n", spec.Defaults.BatchBytes)
			fmt.Fprintf(out, "  read partitions   %d\n", spec.Defaults.ReadPartitions)
			fmt.Fprintf(out, "  write concurrency %d\n", spec.Defaults.WriteConcurrency)
			fmt.Fprintf(out, "  memory budget     %s\n", spec.Defaults.MemoryBudget)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&file, "file", "f", "", "path to the job spec (required)")
	f.BoolVar(&allowInlineSecrets, "allow-inline-secrets", false,
		"permit credentials written directly into the spec")
	f.BoolVar(&resolve, "resolve", false,
		"also resolve ${env:...} and ${file:...} references")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
