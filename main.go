package main

import (
	"bufio"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

var ipPersons = make(map[string]Person) // ip, Person
var discordSession *discordgo.Session
var discordChannelID string

type Person struct {
	Name     string    `json:"name"`
	JoinTime time.Time `json:"joinTime"`
}

type Config struct {
	ChannelID          string `yaml:"channelID"`
	BotToken           string `yaml:"botToken"`
	TerrariaServerPort string `yaml:"terrariaServerPort"`
	TerrariaBinaryPath string `yaml:"terrariaBinaryPath"`
	TerrariaWorldPath  string `yaml:"terrariaWorldPath"`
	WebServerPort      string `yaml:"webServerPort"`
}

func main() {
	b, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	config := Config{}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatal(err)
	}
	discordChannelID = config.ChannelID

	terrariaPty, err := NewTerrariaPty(TerrariaPtyArgs{
		TerrariaServerPort: config.TerrariaServerPort,
		TerrariaBinaryPath: config.TerrariaBinaryPath,
		TerrariaWorldPath:  config.TerrariaWorldPath,
	})
	if err != nil {
		log.Fatal("failed starting terraria pty", err)
	}
	go startDiscordConsoleRelay(terrariaPty.tty)

	discordSession, err = discordgo.New("Bot " + config.BotToken)
	if err != nil {
		log.Fatal("failed getting bot session", err)
	}

	gin.SetMode(gin.ReleaseMode)
	go startWebServer(config.WebServerPort, terrariaPty)

	fmt.Println("Running")
	shouldExit := make(chan bool)
	<-shouldExit
}

func startDiscordConsoleRelay(tty io.Reader) {
	scanner := bufio.NewScanner(tty)
	relayConsoleText(scanner)
}

func relayConsoleText(scanner *bufio.Scanner) {
	lastIP := "NA"
	for scanner.Scan() { // breaks when hard resetting
		text := scanner.Text()
		if strings.Contains(text, "is connecting") {
			lastIP = text[:strings.Index(text, " is connecting")]
			lastIP = strings.Split(lastIP, ":")[0]
		}
		if strings.Contains(text, " has joined.") {
			personName := text[:strings.Index(text, " has joined.")]
			notifyServerChannel(personName + " has joined!")
			ipPersons[lastIP] = Person{
				Name:     personName,
				JoinTime: time.Now(),
			}
		}
		if strings.Contains(text, " has left.") {
			personName := text[:strings.Index(text, " has left.")]
			person := Person{}
			for key, p := range ipPersons {
				if p.Name == personName {
					person = p
					delete(ipPersons, key)
				}
			}
			notifyServerChannel(personName + " has left. They played for " + humanizedDuration(time.Since(person.JoinTime)))
		}
		log.Println(text)
	}
}

func handlerCmd(c *gin.Context, terrariaPty *TerrariaPty) {
	body, _ := ioutil.ReadAll(c.Request.Body)
	bStr := string(body)
	if bStr != "dusk" && bStr != "dawn" && bStr != "noon" && bStr != "midnight" && bStr != "hardReset" {
		c.JSON(400, gin.H{"msg": "error", "error": "command not allowed"})
		return
	}

	if bStr == "hardReset" {
		err := terrariaPty.HardReboot()
		if err != nil {
			c.JSON(400, gin.H{"msg": "error", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"msg": "Executed hard reset successfully."})
		return
	}

	err := terrariaPty.WriteConsole(string(body))
	if err != nil {
		c.JSON(400, gin.H{"msg": "error", "error": err.Error()})
		return
	}

	ip := strings.Split(c.ClientIP(), ":")[0]
	if p, exists := ipPersons[ip]; exists {
		notifyServerChannel(p.Name + " issued command: " + bStr)
	} else {
		notifyServerChannel("Someone not playing in the server issued command: " + bStr)
	}
	c.JSON(200, gin.H{"msg": "passed"})
}

func notifyServerChannel(msg string) {
	_, err := discordSession.ChannelMessageSend(discordChannelID, msg)
	if err != nil {
		log.Println("err sending message to discord server", err)
	}
}

func startWebServer(webServerPort string, terrariaPty *TerrariaPty) {
	r := gin.Default()
	r.POST("/cmd", func(c *gin.Context) {
		handlerCmd(c, terrariaPty)
	})
	r.StaticFS("/", http.Dir("./static"))
	r.Run(":" + webServerPort)
}

func humanizedDuration(duration time.Duration) string {
	if duration.Seconds() < 60.0 {
		return fmt.Sprintf("%d seconds", int64(duration.Seconds()))
	}
	if duration.Minutes() < 60.0 {
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d minutes %d seconds", int64(duration.Minutes()), int64(remainingSeconds))
	}
	if duration.Hours() < 24.0 {
		remainingMinutes := math.Mod(duration.Minutes(), 60)
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d hours %d minutes %d seconds",
			int64(duration.Hours()), int64(remainingMinutes), int64(remainingSeconds))
	}
	remainingHours := math.Mod(duration.Hours(), 24)
	remainingMinutes := math.Mod(duration.Minutes(), 60)
	remainingSeconds := math.Mod(duration.Seconds(), 60)
	return fmt.Sprintf("%d hours %d minutes %d seconds",
		int64(remainingHours),
		int64(remainingMinutes), int64(remainingSeconds))
}
