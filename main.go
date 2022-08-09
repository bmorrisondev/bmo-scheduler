package main

import (
	"time"

	"github.com/bmorrisondev/bmo-scheduler/scripts"

	gocron "github.com/go-co-op/gocron"
)

func main() {
	// Init the scheduler
	s := gocron.NewScheduler(time.UTC)

	// Load up scripts
	scripts.SyncNotionContentTasks(s)

	// Start it up!
	s.StartBlocking()
}
