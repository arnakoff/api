// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// The actions of the command; see the package doc.
const (
	actionMigrate = "migrate"
	actionCreate  = "key create"
	actionList    = "key list"
	actionRevoke  = "key revoke"
)

// command is the action asked on the command line and its flags. An empty
// env means the configured ARNAKOFF_API_KEY_ENV; an empty listen, a
// migration that exits once done.
type command struct {
	action string
	label  string
	env    string
	listen string
	id     int64
}

// errUsage is a misuse of the arguments.
var errUsage = errors.New("usage")

// usage is printed with every misuse.
const usage = `usage:
  admin migrate [-listen <address>]
  admin key create -label <holder> [-env live|test]
  admin key list
  admin key revoke -id <id>`

// parseArgs reads args, writing usage and errors to w. It returns
// flag.ErrHelp for -h.
func parseArgs(args []string, w io.Writer) (command, error) {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "-help" || args[0] == "--help") {
		_, _ = fmt.Fprintln(w, usage)
		return command{}, flag.ErrHelp
	}
	name, rest := actionOf(args)
	cmd := command{action: name}
	fs := flag.NewFlagSet("admin "+name, flag.ContinueOnError)
	fs.SetOutput(w)
	switch name {
	case actionCreate:
		fs.StringVar(&cmd.label, "label", "", "whom the key is issued to, 1 to 100 characters")
		fs.StringVar(&cmd.env, "env", "", "live or test; ARNAKOFF_API_KEY_ENV by default")
	case actionRevoke:
		fs.Int64Var(&cmd.id, "id", 0, "identifier of the key, from key list")
	case actionMigrate:
		fs.StringVar(&cmd.listen, "listen", "", "serve /healthz and /readyz on this address until stopped, /readyz answering 200 once the migrations are applied")
	case actionList:
	default:
		return command{}, misuse(w, "unknown action")
	}
	if err := fs.Parse(rest); err != nil {
		return command{}, err
	}
	if err := validate(cmd, fs.NArg()); err != nil {
		return command{}, misuse(w, err.Error())
	}
	return cmd, nil
}

// actionOf returns the action args start with and the arguments after it.
func actionOf(args []string) (string, []string) {
	switch {
	case len(args) >= 1 && args[0] == actionMigrate:
		return actionMigrate, args[1:]
	case len(args) >= 2 && args[0] == "key":
		return "key " + args[1], args[2:]
	default:
		return "", nil
	}
}

// validate returns what is wrong with cmd, given extra positional
// arguments.
func validate(cmd command, extra int) error {
	switch {
	case extra > 0:
		return errors.New("no argument expected after the flags")
	case cmd.action == actionCreate && cmd.label == "":
		return errors.New("-label is required")
	case cmd.action == actionRevoke && cmd.id <= 0:
		return errors.New("-id must be a key identifier, from key list")
	}
	return nil
}

// misuse writes msg and the usage to w and returns errUsage.
func misuse(w io.Writer, msg string) error {
	_, _ = fmt.Fprintf(w, "%s\n%s\n", msg, usage)
	return fmt.Errorf("%w: %s", errUsage, msg)
}
