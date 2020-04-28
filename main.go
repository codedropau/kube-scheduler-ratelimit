package main

import (
	scheduler "k8s.io/kubernetes/cmd/kube-scheduler/app"

	"github.com/codedropau/kube-scheduler-ratelimit/internal/plugins/ratelimit"
)

func main() {
	command := scheduler.NewSchedulerCommand(
		scheduler.WithPlugin(ratelimit.Name, ratelimit.New),
	)
	if err := command.Execute(); err != nil {
		panic(err)
	}
}
