package main

import (
	"fmt"
	"log"
	"encoding/json"
	"os"
	"os/signal"
	"math/rand/v2"
	"github.com/bwmarrin/discordgo"
)

var AppId string

func initGuild(s *discordgo.Session, appId, guildId string) {
	s.ApplicationCommandCreate(appId, guildId, &discordgo.ApplicationCommand{
		Name: "roll",
		Description: "Roll the dice",
		Options: []*discordgo.ApplicationCommandOption {
			{
				Type: discordgo.ApplicationCommandOptionInteger,
				Name: "modifier",
				Description: "Stat modifier",
			},
			{
				Type: discordgo.ApplicationCommandOptionBoolean,
				Name: "d20",
				Description: "Roll d20 instead",
			},
		},
	})
}

type CommandHandler func(options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionResponse

var handlers = map[string]CommandHandler {
	"roll": func(options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionResponse {
		var mod int64
		var d20 bool

		for _, opt := range options {
			switch opt.Name {
			case "modifier":
				mod = opt.IntValue()
			case "d20":
				d20 = opt.BoolValue()
			}
		}
		var response string

		if d20 {
			res := rand.Int64N(20) + 1
			sum := res + mod
			response = fmt.Sprintf("You rolled **%d** (%d)+%d", sum, res, mod)
		} else {
			h := rand.Int64N(12) + 1
			f := rand.Int64N(12) + 1
			sum := h + f + mod

			if h > f {
				response = fmt.Sprintf("You rolled **%d** with **Hope** (%d+%d)+%d", sum, h, f, mod)
			} else if (f > h) {
				response = fmt.Sprintf("You rolled **%d** with **Fear** (%d+%d)+%d", sum, h, f, mod)
			} else {
				response = fmt.Sprintf("**CRIT!** (%d %d)", h, f)
			}
		}

		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		}
	},
}

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	options := data.Options

	var response *discordgo.InteractionResponse

	h, ok := handlers[data.Name]

	if(!ok) {
		fmt.Fprintf(os.Stderr, "Command %s not found\n", data.Name)
		response = &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error",
			},
		}
	} else {
		response = h(options)
	}

	err := s.InteractionRespond(i.Interaction, response)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error responding: %v\n", err)
	}
}

func newGuildHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	initGuild(s, AppId, g.ID)
}

func readyHandler(s *discordgo.Session, m *discordgo.Ready) {
	fmt.Printf("Bot Ready\n")

	AppId = m.Application.ID
}

type Config struct {
	Token string `json:"token"`
}

func readConfig() (*Config, error) {
	f, err := os.Open("./.config.json")
	if err != nil {
		return nil, fmt.Errorf("While opening config file: %w", err)
	}

	dec := json.NewDecoder(f)
	var config Config
	err = dec.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("While reading config: %w", err)
	}
	return &config, nil
}

func main() {
	config, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	auth := "Bot " + config.Token 
	discord, err := discordgo.New(auth)

	discord.Identify.Intents |= discordgo.IntentsAllWithoutPrivileged

	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(newGuildHandler)
	discord.AddHandler(readyHandler)
	discord.AddHandler(interactionHandler)

	err = discord.Open()
	if err != nil {
		log.Fatal(err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = discord.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}

