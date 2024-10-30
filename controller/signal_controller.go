package controller

import (
	"os"
	"os/signal"
	"syscall"

	utils "git.matador.ais.co.th/cncm/nmgw-utililty-library/v2"

)

func Channel() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGTERM)
	for {
		s := <-signalChan
		switch s {
		case syscall.SIGHUP:
			Log.Info(&utils.EventLog{
				Event: utils.Events.Receive_Signal,
				Description: utils.Description{
					Method: "Reload",
				},
			})
			ReloadConfig()
		case syscall.SIGTERM:
			Log.Info(&utils.EventLog{
				Event: utils.Events.Receive_Signal,
				Description: utils.Description{
					Method: "Shutdown",
				},
			})
			if err := ShutDownConsumeToKafka(); err != nil {
				Log.Error(&utils.EventLog{
					Event: utils.Events.Shutdown_Application,
					Description: utils.Description{
						Method: "Disconnect_Kafka",
						Status: "Error",
						Detail: err.Error(),
					},
				})
				os.Exit(1)
			} else {
				Log.Info(&utils.EventLog{
					Event: utils.Events.Shutdown_Application,
					Description: utils.Description{
						Method: "Disconnect_Kafka",
						Status: "Success",
					},
				})
				os.Exit(0)
			}
		}
	}
}
