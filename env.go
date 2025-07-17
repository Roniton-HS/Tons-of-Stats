package main

import (
	"os"

	_ "github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

type Env struct {
	// IsProd reports whether the bot is running in a production environment. This
	// is true if, and only if, the environment variable "PROD" is equal to "1".
	IsProd bool

	// Bot token for the discord api WITHOUT a "Bot "-prefix.
	//
	// Read from DISCORD_BOT_TOKEN.
	Token string

	// Server ID to try and connect to (see [discordgo.Guild]).
	//
	// Read from SERVER_ID.
	ServerID string

	// Name of the channel to listen for results (see [LoldleStats]) in.
	//
	// Read from RESULT_CHANNEL.
	ResultsCh string

	// Name of the channel to use for posting daily results and leaderboards.
	//
	// Read from STATS_CHANNEL.
	StatsCh string
}

// NewEnv creates a new [*Env], reading required values from the environment.
// Where possible, this function provides more-or-less (probably less) sensible
// defaults.
func NewEnv() *Env {
	log.Info("Setting up environment")

	env := &Env{
		ResultsCh: "result-spam",
		StatsCh:   "daily-stats",
	}

	if v, ok := os.LookupEnv("PROD"); ok && v == "1" {
		env.IsProd = true
	}

	if v, ok := os.LookupEnv("DISCORD_BOT_TOKEN"); ok {
		env.Token = v
	} else {
		log.Fatal("DISCORD_BOT_TOKEN not set")
	}
	if v, ok := os.LookupEnv("SERVER_ID"); ok {
		env.ServerID = v
	} else {
		log.Fatal("SERVER_ID not set")
	}

	if v, ok := os.LookupEnv("RESULT_CHANNEL"); ok {
		env.ResultsCh = v
	}
	if v, ok := os.LookupEnv("STATS_CHANNEL"); ok {
		env.StatsCh = v
	}

	return env
}
