package main

import (
	"os"

	"github.com/charmbracelet/log"
)

type Env struct {
	IsProd   bool
	Token    string
	ServerID string

	ResultsCh string
	StatsCh   string
}

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
