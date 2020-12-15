package main

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
)

func createConfigFile(password string) {
	pHash, err := hashPassword(password)
	if err != nil {
		log.Fatal(err)
	}
	dConf := defaultConfig(pHash)
	confBytes, err := yaml.Marshal(&dConf)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("config.yml", confBytes, 0644)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Config generated at config.yml. Edit this file, then re-run the program.")
}

func askCtrlPanelPassword() string {
	r := bufio.NewReader(os.Stdin)
	fmt.Print("Enter a new web control panel password: ")
	text, err := r.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return text
}

func defaultConfig(panelPasswordHash string) Config {
	return Config{
		DiscordOptions: struct {
			UseDiscordBot bool `yaml:"useDiscordBot"`
			ChannelID     string `yaml:"channelID"`
			BotToken      string `yaml:"botToken"`
		}{UseDiscordBot: false, ChannelID: "xxx", BotToken: "xxx"},
		ControlPanelHeader: "Terraria Control Panel",
		TerrariaServerPort:       "7777",
		TerrariaServerBinaryPath: "./server/TerrariaServer.bin.x86_64",
		TerrariaWorldPath:        "./server/Texas.wld",
		WebServerPort:            "80",
		TLSOptions: TLSOptions{UseTLS: false, CertFile: "", KeyFile: ""},
		ControlPanelPassHash: panelPasswordHash,
	}
}

type Config struct {
	DiscordOptions struct {
		UseDiscordBot bool `yaml:"useDiscordBot"`
		ChannelID     string `yaml:"channelID"`
		BotToken      string `yaml:"botToken"`
	} `yaml:"discordOptions"`
	ControlPanelHeader 		string `yaml:"controlPanelHeader"`
	TerrariaServerPort       string `yaml:"terrariaServerPort"`
	WebServerPort            string `yaml:"webServerPort"`
	TerrariaServerBinaryPath string `yaml:"terrariaServerBinaryPath"`
	TerrariaWorldPath        string `yaml:"terrariaWorldPath"`
	TLSOptions TLSOptions `yaml:"tlsOptions"`
	ControlPanelPassHash string `yaml:"controlPanelPassHash"`
}

type TLSOptions struct {
	UseTLS   bool `yaml:"useTLS"`
	CertFile string `yaml:"certFilePath"`
	KeyFile  string `yaml:"keyFilePath"`
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}