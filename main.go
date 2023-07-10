package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var config Configuration

type Configuration struct {
	MonitorDelay            int     `json:"monitorDelay"`
	Webhook                 string  `json:"webhook"`
	SkinportUsername        string  `json:"skinportUsername"`
	SkinportPassword        string  `json:"skinportPassword"`
	TwoCaptchaKey           string  `json:"twoCaptchaKey"`
	BotToken                string  `json:"botToken"`
	DmarketPublicKey        string  `json:"dmarketPublicKey"`
	DmarketPrivateKey       string  `json:"dmarketPrivateKey"`
	MinimumProfitPercentage float64 `json:"minimumProfitPercentage"`
	MinimumPrice            float64 `json:"minimumPrice"`
	MaximumPrice            float64 `json:"maximumPrice"`
	InputChannel            string  `json:"inputChannel"`
}

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

func init() {
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	configFile, err := os.Open("config.json")
	defer configFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	CreateWebhookClient(config.Webhook)
}

func main() {
	UpdateAvailableBalance()
	login()
	fetchPrices()

	go RunDmarket(config.MonitorDelay, "p2p")
	go RunDmarket(config.MonitorDelay, "dmarket")
	go RunSkinport()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
