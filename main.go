package main

import (
	"bufio"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
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

func main() {
	b, err := ioutil.ReadFile("config.yml")
	if err != nil && strings.Contains(err.Error(), "no such file or directory"){
		password := askCtrlPanelPassword()
		createConfigFile(password)
		os.Exit(0)
	} else if err != nil {
		log.Fatal(err)
	}

	config := Config{}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatal(err)
	}
	discordChannelID = config.DiscordOptions.ChannelID

	terrariaPty, err := NewTerrariaPty(TerrariaPtyArgs{
		TerrariaServerPort: config.TerrariaServerPort,
		TerrariaBinaryPath: config.TerrariaServerBinaryPath,
		TerrariaWorldPath:  config.TerrariaWorldPath,
	})
	if err != nil {
		log.Fatal("failed starting terraria pty", err)
	}
	go startDiscordConsoleRelay(terrariaPty.tty)

	discordSession, err = discordgo.New("Bot " + config.DiscordOptions.BotToken)
	if err != nil {
		log.Fatal("failed getting bot session", err)
	}

	gin.SetMode(gin.ReleaseMode)
	go startWebServer(config.WebServerPort, terrariaPty, config.TLSOptions)

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
	for scanner.Scan() { // breaks out when hard resetting
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// todo: register this handler and implement consoleOutput
func handlerConsoleOutput(c *gin.Context, consoleOutput chan string) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}
	for {
		consoleText, ok := <- consoleOutput
		if !ok {
			conn.Close()
			break
		}
		err := conn.WriteMessage(websocket.TextMessage, []byte(consoleText))
		if err != nil {
			log.Println("err writing message to client", err)
			break
		}
	}
}

func notifyServerChannel(msg string) {
	_, err := discordSession.ChannelMessageSend(discordChannelID, msg)
	if err != nil {
		log.Println("err sending message to discord server", err)
	}
}

func startWebServer(webServerPort string, terrariaPty *TerrariaPty, tlsOptions TLSOptions ) {
	r := gin.Default()
	r.POST("/cmd", func(c *gin.Context) {
		handlerCmd(c, terrariaPty)
	})
	r.StaticFS("/", http.Dir("./static"))
	if tlsOptions.UseTLS {
		r.RunTLS(":" + webServerPort, tlsOptions.CertFile, tlsOptions.KeyFile)
	} else {
		r.Run(":" + webServerPort)
	}

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
