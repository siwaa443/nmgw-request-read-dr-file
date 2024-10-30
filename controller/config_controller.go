package controller

import (
	"log"
	"os"

	utils "git.matador.ais.co.th/cncm/nmgw-utililty-library/v2"
	"github.com/siwaa443/nmgw-request-read-dr-file/models"

)

func InitConfig() {
	var config models.Configuration
	err := utils.InitConfig(&config)
	if err != nil {
		log.Panic("InitConfig err:", err)
		os.Exit(1)
	}
	models.SetConfiguration(config)
}

func ReloadConfig() {
	var newConfig models.Configuration
	err := utils.ReloadConfig(&newConfig)
	if err != nil {
		Log.Error(&utils.EventLog{
			Event: utils.Events.Reload_Application,
			Description: utils.Description{
				Method: "Set_Config",
				Detail: err.Error(),
			},
		})
		return
	}
	config := models.GetConfiguration()
	config.Warm = newConfig.Warm
	models.SetConfiguration(config)
	manageHttpClientForSendDR()
	ReloadLog()
	Log.Info(&utils.EventLog{
		Event: utils.Events.Reload_Application,
		Description: utils.Description{
			Method: "Set_Config",
			Detail: config,
		},
	})
}
