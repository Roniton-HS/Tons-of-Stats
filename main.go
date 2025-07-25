package main

import (
	"time"
	"tons-of-stats/db"
	sess "tons-of-stats/session"

	_ "github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

// ACCENT denotes the color used for message accents.
var ACCENT = int(0xd6aa38)

var dal *DAL
var env *Env
var session *sess.Session
var leaderboard *Leaderboard

func main() {
	log.SetDefault(
		log.NewWithOptions(nil, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.Kitchen,
			Level:           log.DebugLevel,
		}),
	)

	err := godotenv.Load()
	if err != nil {
		log.Warn("Failed to load .env", "err", err)
	}

	env = NewEnv()
	if env.IsProd {
		log.SetLevel(log.InfoLevel)
	}

	// Database configuration
	db, err := db.NewDB("tons_of_stats.sqlite")
	if err != nil {
		log.Fatal("Could not open database", "err", err)
	}
	defer db.Close()

	dal = NewDAL(db)

	// Discord session configuration
	session = sess.NewSession(env.Token, env.ServerID)
	if err := session.Open(cmds); err != nil {
		log.Fatal("Failed to open session", "err", err)
	}

	session.HandlerAdd("record-stats", RecordStats)

	// Stat display and scheduling
	l, err := NewLeaderboard(dal, env, session)
	if err != nil {
		log.Fatal("Failed to initialize leaderboard", "err", err)
	}
	leaderboard = l

	now := time.Now()
	midnight := time.Date(
		now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location(),
	)
	go schedule(midnight, 24*time.Hour, dailyReset)

	log.Info("Running...")
	select {}
}
