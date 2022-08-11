package main

import (
	"time"

	"github.com/bmorrisondev/bmo-scheduler/scripts"
	"github.com/joho/godotenv"

	gocron "github.com/go-co-op/gocron"
)

func main() {
	// Load .env file
	godotenv.Load()

	// Init the scheduler
	s := gocron.NewScheduler(time.UTC)

	// Load up scripts
	scripts.SyncNotionContentTasks.Register(s)

	// Start it up!
	s.StartBlocking()
}
