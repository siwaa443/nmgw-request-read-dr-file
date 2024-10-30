package controller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	partnerProfile "git.matador.ais.co.th/cncm/nmgw-partner-profile-management"

	utils "git.matador.ais.co.th/cncm/nmgw-utililty-library/v2"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/robfig/cron/v3"
	"github.com/siwaa443/nmgw-request-read-dr-file/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Dr struct {
	Log     *zap.Logger
	Crontab *cron.Cron
}

var (
	dr    = make(map[string]*Dr)
	drSeq = make(map[string]*DrSeq)
	mtSeq = make(map[string]*MtSeq)
)

type DrSeq struct {
	drmutex *sync.Mutex
	drseq   int
}
type MtSeq struct {
	mtmutex *sync.Mutex
	mtseq   int
}

var mutex = sync.RWMutex{}

const keywordConfProfileId = "${profileId}"

func InitDr() {
	var err error
	config := models.GetConfiguration()
	partnerServiceProfile = partnerProfile.GetPartnerServiceProfile()
	mutex.Lock()
	defer mutex.Unlock()
	dr = make(map[string]*Dr)

	for profileid := range partnerServiceProfile.CodeOfProfileId {
		path := strings.Replace(config.Warm.Dr.Path, keywordConfProfileId, profileid, -1)
		formatFile := strings.Replace(config.Warm.Dr.Format, keywordConfProfileId, profileid, -1)
		dr[profileid], err = InitialDr(models.DrConfiguration{
			Path:   path,
			Format: formatFile,
			Rotate: config.Warm.Dr.Rotate,
		})
		if err != nil {
			Log.Error(&utils.EventLog{
				Event: utils.Events.Reload_Application,
				Description: utils.Description{
					Method: "Init_Profile",
					Detail: err.Error(),
				},
			})
			return
		}
		drSeq[profileid] = &DrSeq{
			drmutex: &sync.Mutex{},
			drseq:   0,
		}

	}
	Log.Info(&utils.EventLog{
		Event: utils.Events.Initial_Application,
		Description: utils.Description{
			Method: "Init_DR",
			Status: "Success",
		},
	})
}

func ReloadDr(event string) {
	for _, oldDr := range dr {
		oldDr.Crontab.Stop()
	}
	InitDr()
}

func InitialDr(config models.DrConfiguration) (*Dr, error) {
	drFile := config.Path + "/" + config.Format
	rotate := config.Rotate
	var (
		rotatetime time.Duration
		rotatecron string
		encCfg     zapcore.EncoderConfig
	)
	if index := strings.Index(strings.ToLower(rotate), "m"); index != -1 {
		tmp, err := strconv.Atoi(rotate[:index])
		if err != nil {
			return &Dr{}, err
		}
		rotatetime = time.Duration(tmp) * time.Minute
		rotatecron = fmt.Sprintf("*/%s * * * *", rotate[:index])
	} else if index = strings.Index(strings.ToLower(rotate), "h"); index != -1 {
		tmp, err := strconv.Atoi(rotate[:index])
		if err != nil {
			return &Dr{}, err
		}
		rotatetime = time.Duration(tmp) * time.Hour
		rotatecron = fmt.Sprintf("0 */%s * * *", rotate[:index])
	} else if index = strings.Index(strings.ToLower(rotate), "d"); index != -1 {
		rotatetime = 24 * time.Hour
		rotatecron = "0 0 */1 * *"
	} else {
		//fmt.Println("default 1 hour")
		rotatetime = 1 * time.Hour
		rotatecron = "0 */1 * * *"
	}
	rotator, err := rotatelogs.New(
		drFile,
		rotatelogs.WithRotationTime(rotatetime))
	if err != nil {
		return &Dr{}, err
	}
	encCfg.MessageKey = "msg"
	// add the encoder config and rotator to create a new zap logger
	w := zapcore.AddSync(rotator)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encCfg),
		w,
		zap.InfoLevel)
	tempDr := Dr{
		Log:     zap.New(core),
		Crontab: cron.New(),
	}
	tempDr.Crontab.AddFunc(rotatecron, func() { rotator.Rotate() })
	tempDr.Crontab.Start()
	return &tempDr, nil
}

func DR(tid, profileId string, consumeData models.ConsumeData, requireOnline bool, requireOffline bool, transProcessTime time.Time) (error, error) {
	var drOnlineErr error = fmt.Errorf("none")
	var drOfflineErr error = fmt.Errorf("none")
	if requireOnline {
		drOnlineErr = SendDROnline(tid, profileId, consumeData, transProcessTime)
		if drOnlineErr != nil {
			drOfflineErr = WriteDrOffline(tid, profileId, consumeData)
			return drOnlineErr, drOfflineErr
		}
	}
	if requireOffline {
		drOfflineErr = WriteDrOffline(tid, profileId, consumeData)
	}
	return drOnlineErr, drOfflineErr
}

func WriteDrOffline(tid, profileId string, consumeData models.ConsumeData) error {
	var context string
	logMsg := utils.TransLog{Tid: &tid, Event: utils.Events.Write_File}
	description := utils.Description{Method: "DR"}

	if consumeData.Cmd == "DLVRREP" {
		offlineReportField := partnerServiceProfile.ServiceOfProfileId[profileId].OfflineReport.Field
		shortMessageSlice := strings.Split(consumeData.ShortMsg, " ")
		var doneDateValue string
		for i := 0; i < len(shortMessageSlice); i++ {
			if shortMessageSlice[i] == "done" && i+1 < len(shortMessageSlice) && strings.HasPrefix(shortMessageSlice[i+1], "date:") {
				doneDateValue = strings.TrimPrefix(shortMessageSlice[i+1], "date:")
				break
			}
		}

		field := map[string]string{
			"command":           consumeData.Cmd,
			"senderName":        consumeData.To,
			"destinationNumber": consumeData.From,
			"serviceNumber":     consumeData.Code,
			"smid":              consumeData.Vsmid,
			"campaignName":      consumeData.Campaign,
			"transactionId":     consumeData.TransactionId,

			"CMD":      consumeData.Cmd,
			"SENDER":   consumeData.To,
			"NTYPE":    "REP",
			"FROM":     consumeData.From,
			"CODE":     consumeData.Code,
			"SMID":     consumeData.Vsmid,
			"STATUS":   consumeData.Status,
			"DETAIL":   consumeData.Detail,
			"TO":       consumeData.To,
			"TOTALMSG": strconv.Itoa(consumeData.TotalMsg),
			"TIME":     doneDateValue,
			"CAMPAIGN": consumeData.Campaign,
		}
		if consumeData.Status == "ERR" {
			field["DETAIL"] = consumeData.ErrCode
		}
		for count, val := range offlineReportField {
			context = context + val + "=" + field[val] + "&"
			if count == len(offlineReportField)-1 {
				context = context[:len(context)-1]
			}
		}
		mutex.RLock()
		if dr[profileId].Log == nil {
			errMsg := "profileId not match in Dr Offline"
			description.Status = "Error"
			description.Detail = map[string]interface{}{"Message": errMsg, "ProfileId": profileId, "DrMessage": context}
			logMsg.Description = description
			Log.Error(&logMsg)
			return fmt.Errorf(errMsg)
		}
		dr[profileId].Log.Info(context)
		description.Status = "Success"
		//description.Detail = map[string]interface{}{"Message": context}
		logMsg.Description = description
		Log.Info(&logMsg)
		mutex.RUnlock()
	} else if consumeData.Cmd == "DLVRMSG" {
		msg := fmt.Sprintf("TRANSID=%s&CMD=DLVRMSG&FET=%s&NTYPE=%s&FROM=%s&TO=%s&CODE=%s&CTYPE=%s&CONTENT=%s&CAMPAIGN=%s", consumeData.Tid, consumeData.Fet, "REP", consumeData.From, consumeData.To, consumeData.Code, consumeData.Ctype, consumeData.Content, consumeData.Campaign)
		mutex.RLock()
		if dr[profileId].Log == nil {
			errMsg := "profileId not match in Dr Offline"
			description.Status = "Error"
			description.Detail = map[string]interface{}{"Message": errMsg, "ProfileId": profileId, "DrMessage": context}
			logMsg.Description = description
			Log.Error(&logMsg)
			return fmt.Errorf(errMsg)
		}
		dr[profileId].Log.Info(msg)
		description.Status = "Success"
		//description.Detail = map[string]interface{}{"Message": msg}
		logMsg.Description = description
		Log.Info(&logMsg)
		mutex.RLock()
	}
	return nil
}

func SendDROnline(tid, profileId string, consumeData models.ConsumeData, transProcessTime time.Time) error { //send http request to end point
	var body []byte
	onlineReportConfig := partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport
	logMsg := utils.TransLog{Event: utils.Events.Send_Request_To_Server, Tid: &tid, StartTimeTrans: transProcessTime}
	description := utils.Description{Method: "Sent_DR_Online", Status: "Success"}
	shortMessageSlice := strings.Split(consumeData.ShortMsg, " ")
	var doneDateValue string
	for i := 0; i < len(shortMessageSlice); i++ {
		if shortMessageSlice[i] == "done" && i+1 < len(shortMessageSlice) && strings.HasPrefix(shortMessageSlice[i+1], "date:") {
			doneDateValue = strings.TrimPrefix(shortMessageSlice[i+1], "date:")
			break
		}
	}
	if len(partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Endpoint.Url) == 0 {
		errMsg := "destination URL of profile is empty"
		description.Status = "Error"
		description.Detail = map[string]interface{}{"ProfileId": profileId, "Message": errMsg}
		logMsg.Description = description
		logMsg.StartTimeTrans = transProcessTime
		Log.Error(&logMsg)
		return fmt.Errorf(errMsg)
	}
	if consumeData.Cmd == "DLVRREP" {
		onlineReportField := partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Endpoint.Field
		field := map[string]string{"CODE": consumeData.Code,
			"SENDER":        consumeData.To,
			"TIME":          doneDateValue,
			"CMD":           consumeData.Cmd,
			"NTYPE":         "REP",
			"FROM":          consumeData.From,
			"SMID":          consumeData.Vsmid,
			"STATUS":        consumeData.Status,
			"DETAIL":        consumeData.Detail,
			"TO":            consumeData.To,
			"TOTALMSG":      strconv.Itoa(consumeData.TotalMsg),
			"CAMPAIGN":      consumeData.Campaign,
			"campaignName":  consumeData.Campaign,
			"transactionId": consumeData.TransactionId}
		if consumeData.Status == "ERR" {
			field["DETAIL"] = consumeData.ErrCode
		}
		switch onlineReportConfig.Type {
		case "xml":
			var xmlBody string
			for count, val := range onlineReportField {
				xmlBody = xmlBody + val + "=" + field[val] + "&"
				if count == len(onlineReportField)-1 {
					xmlBody = xmlBody[:len(xmlBody)-1]
				}
			}
			body = []byte(xmlBody)
		case "smpp":
			smppReport := models.SMPPFormat{
				Seq:        consumeData.Seq,
				Tid:        consumeData.Tid,
				Code:       consumeData.Code,
				To:         consumeData.To,
				SenderName: consumeData.From,
				ProcessTime: models.ProcessTimeDR{
					InitProcessTime: consumeData.InitProcessTime.Format(time.RFC3339Nano),
					FlowProcessTime: consumeData.FlowProcessTime.Format(time.RFC3339Nano),
				},
			}
			if consumeData.ShortMsg == "" {
				submitDate := consumeData.InitProcessTime.Format("0601021504")
				doneDate := time.Now().Format("0601021504")
				smppReport.ShortMessage = fmt.Sprintf("id:%s submit date:%s done date:%s stat:UNDELIV err:008 text:", consumeData.Vsmid, submitDate, doneDate)

			} else {
				shortMessageSlice := strings.Split(consumeData.ShortMsg, " ")
				for index, value := range shortMessageSlice {
					tmp := strings.Split(value, ":")
					if tmp[0] == "id" {
						shortMessageSlice[index] = fmt.Sprintf("id:%s", consumeData.Vsmid)
						break
					}
				}
				smppReport.ShortMessage = strings.Join(shortMessageSlice, " ")

			}
			body, _ = json.Marshal(smppReport)
		default:
			var jsonBody string
			for count, val := range onlineReportField {
				jsonBody = jsonBody + fmt.Sprintf("\"%s\":\"%s\",", val, field[val])
				if count == len(onlineReportField)-1 {
					jsonBody = jsonBody[:len(jsonBody)-1]
					jsonBody = "{" + jsonBody + "}"
				}
			}
			body = []byte(jsonBody)
		}
	} else if consumeData.Cmd == "DLVRMSG" {
		if onlineReportConfig.Type == "xml" {
			body = []byte(fmt.Sprintf("TRANSID=%s&CMD=DLVRMSG&FET=%s&NTYPE=%s&FROM=%s&TO=%s&CODE=%s&CTYPE=%s&CONTENT=%s&CAMPAIGN=%s", consumeData.Tid, consumeData.Fet, consumeData.Ntype, consumeData.From, consumeData.To, partnerServiceProfile.CodeOfProfileId[profileId], consumeData.Ctype, consumeData.Content, consumeData.Campaign))
		} else {
			bodyJson := models.BodyJsonFormat{
				Transid: consumeData.Tid,
				Cmd:     consumeData.Cmd,
				Fet:     consumeData.Fet,
				Ntype:   consumeData.Ntype,
				From:    consumeData.From,
				To:      consumeData.To,
				Code:    partnerServiceProfile.CodeOfProfileId[profileId],
				Ctype:   consumeData.Ctype,
				Content: consumeData.Content,
			}
			bodyByte, err := json.Marshal(bodyJson)
			if err != nil {
				description.Status = "Error"
				description.Detail = map[string]interface{}{"Data": bodyJson, "Message": err.Error()}
				logMsg.Description = description
				Log.Error(&logMsg)
				return err
			}
			body = bodyByte
		}
	}
	var header = make(map[string]string)
	if len(onlineReportConfig.Endpoint.Header) == 0 {
		onlineReportConfig.Endpoint.Header = []string{}
	} else {
		for index := range onlineReportConfig.Endpoint.Header {
			str := strings.SplitN(onlineReportConfig.Endpoint.Header[index], ":", 2)
			key := strings.TrimSpace(str[0])
			value := strings.TrimSpace(str[1])
			header[key] = value

		}
	}
	requestConfig := Request{
		Method:      "POST",
		ContentType: "text/plain",
		Header:      header,
		Timeout:     *onlineReportConfig.Endpoint.Timeout,
		Tid:         *logMsg.Tid,
		Body:        body,
	}

	drSeq := manageDRSeq(profileId)
	partnerServiceProfile = partnerProfile.GetPartnerServiceProfile()
	requestConfig.Url = partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Endpoint.Url[drSeq]
	description.Detail = map[string]interface{}{"Body": string(body)}
	description.IP = requestConfig.Url
	logMsg.Description = description
	logMsg.StartTimeTrans = transProcessTime
	Log.Info(&logMsg)
	logMsg.Event = utils.Events.Receive_Response_From_Server
	description.Method = ""
	resp, status, _, err := DRRequest(profileId, requestConfig, consumeData.Tid)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			description.Status = "Error"
			description.Detail = map[string]interface{}{"Message": err.Error(), "responseBody": resp}
			logMsg.Description = description
			if onlineReportConfig.Type == "smpp" {
				_, _ = partnerProfile.PartnerProfileUpdateDRURL(consumeData.Code, requestConfig.Url, "delete")
			}
			Log.Error(&logMsg)
			return err
		} else if strings.Contains(err.Error(), "Invalid Response Format") {
			description.Status = "Error"
			description.Detail = map[string]interface{}{"Message": err.Error(), "responseBody": resp}
			logMsg.Description = description
			Log.Error(&logMsg)
			return err
		} else {
			description.Status = "Error"
			description.Detail = map[string]interface{}{"Message": err.Error(), "responseBody": resp}
			logMsg.Description = description
			Log.Error(&logMsg)
			return err
		}
	}
	if status != "OK" {
		description.Status = "Error"
		description.Detail = map[string]interface{}{"Response": resp, "Message": "Response status is not OK"}
		logMsg.Description = description
		Log.Error(&logMsg)
		return fmt.Errorf(resp)
	} else {
		description.Status = "Success"
		description.Detail = map[string]interface{}{"Response": resp}
		logMsg.Description = description
		Log.Info(&logMsg)
		return nil
	}
}

func manageDRSeq(profileId string) int {
	defer drSeq[profileId].drmutex.Unlock()
	drSeq[profileId].drmutex.Lock()
	drSeq[profileId].drseq++
	if drSeq[profileId].drseq >= len(partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Endpoint.Url) {
		drSeq[profileId].drseq = 0
	}
	return drSeq[profileId].drseq
}


func getStatusFromDRStatus(requireOnline bool, requireOffline bool, drOnlineStatus, drOfflineStatus error) (string, map[string]string) {
	status := "Success"
	detail := map[string]string{
		"DrOnlineStatus":  "Success",
		"DrOfflineStatus": "Success",
	}
	if drOnlineStatus != nil {
		detail["DrOnlineStatus"] = drOnlineStatus.Error()
		if requireOnline {
			status = "Error"
		}
	}
	if drOfflineStatus != nil {
		detail["DrOfflineStatus"] = drOfflineStatus.Error()
		if requireOffline {
			status = "Error"
		}
	}
	return status, detail
}
