package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"

	"github.com/DoomSentinel/scheduler/client"
)

const (
	tasksCount       = 15000
	remoteTasksCount = 10
)

func main() {
	//Opinionated defaults
	host := pflag.String("host", "localhost", "scheduler address")
	port := pflag.Int("port", 8245, "connection TCP port")
	pflag.Parse()

	scheduler, err := client.NewDefaultClient(*host, *port)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = scheduler.Close()
	}()

	//Lets catch when os is no longer can tolerate us
	appDone := make(chan os.Signal, 1)
	signal.Notify(appDone,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Subscribe for task execution notifications
	notifications, errors, err := scheduler.ExecutionNotifications(ctx)
	if err != nil {
		log.Fatal("Execution notification failed: " + err.Error())
	}

	go func() {
		for {
			select {
			case notification, ok := <-notifications:
				if !ok {
					return
				}
				log.Println(notification)
			case err, ok := <-errors:
				if !ok {
					return
				}
				log.Println("notifications error: " + err.Error())
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	//Lest seed scheduler with a large number of tasks
	go func() {
		rand.Seed(time.Now().Unix())
		go func() {
			for i := 0; i < tasksCount; i++ {
				rnd := rand.Intn(15-10) + 10

				err := scheduler.ScheduleDummyTask(ctx, uuid.New().String(), time.Now().Add(time.Duration(rnd)*time.Second))
				if err != nil {
					log.Println("unable to schedule task: ", err.Error())
				}
			}
		}()

		go func() {
			for i := 0; i < tasksCount/4; i++ {
				rnd := rand.Intn(15-10) + 10

				err := scheduler.ScheduleCommandTask(
					ctx, uuid.New().String(), time.Now().Add(time.Duration(rnd)*time.Second), 5*time.Second,
					"echo", "my text", strconv.Itoa(rnd))
				if err != nil {
					log.Println("unable to schedule task: ", err.Error())
				}
			}
		}()

		go func() {
			for i := 0; i < remoteTasksCount; i++ {
				rnd := rand.Intn(15-10) + 10

				err := scheduler.ScheduleRemoteTask(
					ctx, uuid.New().String(), time.Now().Add(time.Duration(rnd)*time.Second), 3,
					client.RemoteConfig{
						Method:        client.MethodGet,
						URL:           "http://worldclockapi.com/api/json/est/now",
						ExpectedCodes: []uint32{http.StatusOK},
						Timeout:       5,
					},
				)
				if err != nil {
					log.Println("unable to schedule task: ", err.Error())
				}
			}
		}()
	}()

	<-appDone
}
