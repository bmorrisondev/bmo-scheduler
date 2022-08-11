package models

import (
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

type JobWrapper struct {
	Name string
	Run  func(*logrus.Logger) error
	Cron string
}

func (w *JobWrapper) Register(s *gocron.Scheduler) {
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	log.Infof("Scheduling Job '%v' with cron expression '%v'", w.Name, w.Cron)

	s.Cron(w.Cron).Do(func() {
		log.Infof("Starting job: %v", w.Name)
		err := w.Run(log)
		if err != nil {
			log.Errorf("Job %v failed!", w.Name)
			log.Error(err)
		}
		log.Infof("Finishing job: %v", w.Name)
	})
	log.Infof("Job '%v' scheduled", w.Name)
}
