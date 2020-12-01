package main

import (
	"errors"
	"fmt"
	"github.com/creack/pty"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// TerrariaPty holds the pseudo-terminal and process for the Terraria server.
type TerrariaPty struct {
	ptyArgs          TerrariaPtyArgs
	cmd              *exec.Cmd
	tty              *os.File
	consoleRelayChan chan string
}

type TerrariaPtyArgs struct {
	TerrariaServerPort string
	TerrariaBinaryPath string
	TerrariaWorldPath  string
}

// NewTerrariaPty uses the config argument to run a Terraria server inside a pseudo-terminal.
func NewTerrariaPty(args TerrariaPtyArgs) (*TerrariaPty, error) {
	cmd, tty, err := createCmdTty(args)
	if err != nil {
		return nil, err
	}
	return &TerrariaPty{
		ptyArgs: args,
		cmd:     cmd,
		tty:     tty,
	}, nil
}

// WriteConsole writes a string to the Terraria console. Should be a command.
func (tp TerrariaPty) WriteConsole(s string) error {
	_, err := io.WriteString(tp.tty, s+"\n")
	if err != nil {
		return err
	}
	return nil
}

// HardReboot reboots the server without saving.
func (tp *TerrariaPty) HardReboot() error {
	err := tp.cmd.Process.Signal(syscall.SIGINT)
	if err != nil {
		return err
	}
	pState, err := tp.cmd.Process.Wait()
	if err != nil {
		return errors.New(err.Error() + " " + pState.String())
	}

	cmd, tty, err := createCmdTty(tp.ptyArgs)
	if err != nil {
		return err
	}

	tp.cmd = cmd
	tp.tty = tty

	go startDiscordConsoleRelay(tp.tty)
	return nil
}

func createCmdTty(args TerrariaPtyArgs) (*exec.Cmd, *os.File, error) {
	cmdName := fmt.Sprintf("%s -world %s -port %s", args.TerrariaBinaryPath, args.TerrariaWorldPath, args.TerrariaServerPort)
	cmdArgs := strings.Fields(cmdName)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:len(cmdArgs)]...)
	tty, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	return cmd, tty, nil
}
