package controller

import (
	"fmt"


	utils "git.matador.ais.co.th/cncm/nmgw-utililty-library/v2"
	"github.com/siwaa443/nmgw-request-read-dr-file/models"
)

// var Log *utils.Logger
var Log *utils.Logger

func InitLog() {
	var err error
	config := models.GetConfiguration()
	Log, err = utils.InitialLog(utils.LogConfiguration{
		Enable: config.Warm.Log.Enable,
		LogApp: utils.LogAppConfig{
			DisableLogApp: true,
			Rename: utils.RenameConfig{
				Enable: false,
			},
		},
		LogDetail: utils.LogDetailConfig{
			Format: config.Warm.Log.Format,
			Level:  config.Warm.Log.Level,
			Rotate: config.Warm.Log.Rotate,
		},
		Path: config.Warm.Log.Path,
	})
	if err != nil {
		fmt.Println("Initial log error :", err)
	}
	Log.Info(&utils.EventLog{
		Event: utils.Events.Initial_Application,
		Description: utils.Description{
			Method: "Set_Config",
			Detail: config,
		},
	})
}

func ReloadLog() {
	var err error
	config := models.GetConfiguration()
	newConfigLog := utils.LogConfiguration{
		LogApp: utils.LogAppConfig{},
		LogDetail: utils.LogDetailConfig{
			Format: config.Warm.Log.Format,
			Level:  config.Warm.Log.Level,
			Rotate: config.Warm.Log.Rotate,
		},
		Path: config.Warm.Log.Path,
	}
	Log, err = Log.ReloadLog(newConfigLog)
	if err != nil {
		Log.Error(&utils.EventLog{
			Event: utils.Events.Reload_Application,
			Description: utils.Description{
				Method: "Set_Log_Config",
				Detail: err.Error(),
			},
		})
	}
}

func WriteSummary(logBuf *utils.Buffer, logMsg utils.TransLog, description utils.Description) {
	description.Method = ""
	logMsg.Event = utils.Events.Summary_log
	logMsg.Description = description
	logBuf.Summary(logMsg).Append()
	logBuf.Flush()
}