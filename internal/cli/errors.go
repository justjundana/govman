package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	cobra "github.com/spf13/cobra"
)

type commandError struct {
	cause   error
	help    string
	usage   bool
	command *cobra.Command
}

func (e *commandError) Error() string { return e.cause.Error() }
func (e *commandError) Unwrap() error { return e.cause }

func withHelp(err error, help string) error {
	if err == nil {
		return nil
	}
	return &commandError{cause: err, help: help}
}

func withUsageHelp(command *cobra.Command, err error, help string) error {
	if err == nil {
		return nil
	}
	return &commandError{cause: err, help: help, usage: true, command: command}
}

func usageArgs(validator cobra.PositionalArgs) cobra.PositionalArgs {
	return func(command *cobra.Command, args []string) error {
		if err := validator(command, args); err != nil {
			return &commandError{cause: err, usage: true, command: command}
		}
		return nil
	}
}

func flagUsageError(command *cobra.Command, err error) error {
	return &commandError{cause: err, usage: true, command: command}
}

func renderCommandError(writer io.Writer, root *cobra.Command, err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, "Error: %v\n", err)

	var typed *commandError
	if errors.As(err, &typed) {
		if typed.help != "" {
			_, _ = fmt.Fprintf(writer, "Help: %s\n", typed.help)
		}
		if typed.usage {
			command := typed.command
			if command == nil {
				command = root
			}
			_, _ = fmt.Fprintln(writer)
			_, _ = fmt.Fprint(writer, command.UsageString())
		}
		return
	}

	if strings.HasPrefix(err.Error(), "unknown command") {
		_, _ = fmt.Fprintln(writer)
		_, _ = fmt.Fprint(writer, root.UsageString())
	}
}
