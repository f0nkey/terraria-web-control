package main

import (
	"bufio"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

var ipPersons = make(map[string]string) // ip, person
var discordSession *discordgo.Session
var config Config

type Config struct {
	ChannelID string `yaml:"channelID"`
	BotToken  string `yaml:"botToken"`
	TerrariaServerPort string `yaml:"terrariaServerPort"`
	TerrariaBinaryPath string `yaml:"terrariaBinaryPath"`
	TerrariaWorldPath string `yaml:"terrariaWorldPath"`
	WebServerPort string `yaml:"webServerPort"`
}

func main() {
	b, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("running")
	cmdName := fmt.Sprintf("%s -world %s -port %s", config.TerrariaBinaryPath, config.TerrariaWorldPath, config.TerrariaServerPort)
	cmdArgs := strings.Fields(cmdName)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:len(cmdArgs)]...)
	gin.SetMode(gin.ReleaseMode)

	tty, err := pty.Start(cmd)
	if err != nil {
		log.Println("err!!", err)
		return
	}

	discordSession, err = discordgo.New("Bot " + config.BotToken)
	if err != nil {
		log.Fatal("failed getting bot session", err)
	}
	go startWebServer(tty)

	scanner := bufio.NewScanner(tty)

	lastIP := "NA"
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, "is connecting") {
			lastIP = text[:strings.Index(text, " is connecting")]
			lastIP = strings.Split(lastIP, ":")[0]
		}
		if strings.Contains(text, " has joined.") {
			person := text[:strings.Index(text, " has joined.")]
			notifyServerChannel(person + " has joined!")
			ipPersons[lastIP] = person
		}
		if strings.Contains(text, " has left.") {
			person := text[:strings.Index(text, " has left.")]
			notifyServerChannel(person + " has left.")
			ipPersons[lastIP] = person
		}
		log.Println(text)
	}
}

func handlerCmd(c *gin.Context, stdin io.WriteCloser) {
	body, _ := ioutil.ReadAll(c.Request.Body)
	bStr := string(body)
	if bStr != "dusk" && bStr != "dawn" && bStr != "noon" && bStr != "midnight" {
		c.JSON(400, gin.H{
			"msg":   "error",
			"error": "command not allowed",
		})
		return
	}

	_, err := io.WriteString(stdin, string(body)+"\n")
	if err != nil {
		c.JSON(400, gin.H{
			"msg":   "error",
			"error": err.Error(),
		})
		return
	}

	ip := strings.Split(c.ClientIP(), ":")[0]
	if p, exists := ipPersons[ip]; exists {
		notifyServerChannel(p + " issued command: " + bStr)
	} else {
		notifyServerChannel("Someone issued command: " + bStr)
	}
	c.JSON(200, gin.H{"msg": "passed"})
}

func notifyServerChannel(msg string) {
	_, err := discordSession.ChannelMessageSend(config.ChannelID, msg)
	log.Println("err sending message to discord server", err)
}

func startWebServer(stdin io.WriteCloser) {
	r := gin.Default()
	r.POST("/cmd", func(c *gin.Context) {
		handlerCmd(c, stdin)
	})
	r.StaticFS("/", http.Dir("./static"))
	r.Run(":" + config.WebServerPort)
}
