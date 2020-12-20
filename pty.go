package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/creack/pty"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
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
	cmd, tty, ch, err := createCmdTtyCh(args)
	if err != nil {
		return nil, err
	}
	go relayConsoleOutput(tty, ch)
	go relayChanToDiscord(ch)
	go relayChanToLog(ch)
	return &TerrariaPty{
		ptyArgs: args,
		cmd:     cmd,
		tty:     tty,
		consoleRelayChan: ch,
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

// WriteConsole writes a string to the Terraria console. Should be a command.
func (tp TerrariaPty) WriteDiscordChannel(s string) error {
	speakDiscord(s)
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

	cmd, tty, ch, err := createCmdTtyCh(tp.ptyArgs)
	if err != nil {
		return err
	}

	tp.cmd = cmd
	tp.tty = tty
	tp.consoleRelayChan = ch

	go relayConsoleOutput(tp.tty, tp.consoleRelayChan)
	go relayChanToLog(tp.consoleRelayChan)
	go relayChanToDiscord(tp.consoleRelayChan)

	return nil
}

func createCmdTtyCh(args TerrariaPtyArgs) (*exec.Cmd, *os.File, chan string, error) {
	cmdName := fmt.Sprintf("%s -world %s -port %s", args.TerrariaBinaryPath, args.TerrariaWorldPath, args.TerrariaServerPort)
	cmdArgs := strings.Fields(cmdName)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:len(cmdArgs)]...)
	tty, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	ch := make(chan string)
	return cmd, tty, ch, nil
}

func relayConsoleOutput(tty io.Reader, output chan string) {
	scanner := bufio.NewScanner(tty)
	for scanner.Scan() { // breaks out when hard resetting
		text := scanner.Text()
		output <- text
	}
}

func relayChanToDiscord(ch chan string) {
	lastIP := "NA"
	for {
		msg, ok := <-ch
		if !ok {
			break
		}

		if strings.Contains(msg, "is connecting") {
			lastIP = msg[:strings.Index(msg, " is connecting")]
			lastIP = strings.Split(lastIP, ":")[0]
		} else if strings.Contains(msg, " has joined.") {
			personName := msg[:strings.Index(msg, " has joined.")]
			speakDiscord(personName + " has joined!")
			ipPersons[lastIP] = Person{
				Name:     personName,
				JoinTime: time.Now(),
			}
		} else if strings.Contains(msg, " has left.") {
			personName := msg[:strings.Index(msg, " has left.")]
			person := Person{}
			for key, p := range ipPersons {
				if p.Name == personName {
					person = p
					delete(ipPersons, key)
				}
			}
			speakDiscord(personName + " has left. They played for " + humanizedDuration(time.Since(person.JoinTime)))
		}
	}
}

func speakDiscord(msg string) {
	_, err := discordSession.ChannelMessageSend(discordChannelID, msg)
	if err != nil {
		log.Println("err sending message to discord server", err)
	}
}

func relayChanToLog(msg chan string) {
	for {
		chData, ok := <-msg
		if !ok {
			break
		}
		log.Println(chData)
	}
}
