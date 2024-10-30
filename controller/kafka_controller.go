package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	connector "git.matador.ais.co.th/cncm/nmgw-connector/v2"
	partnerProfile "git.matador.ais.co.th/cncm/nmgw-partner-profile-management"
	"github.com/siwaa443/nmgw-request-read-dr-file/models"

	utils "git.matador.ais.co.th/cncm/nmgw-utililty-library/v2"
)

var (
	kafkaConfig           models.KafkaConfiguration
	ConsumerShutDown      bool
	tpsLock               sync.Mutex
	startTime             = time.Now()
	countReq              int
	KafkaConsumer         connector.ConsumerConfig
	KafkaProducer         connector.ProducerConfig
	partnerServiceProfile partnerProfile.PartnerServiceProfile
)

type KafkaData struct {
	Tid         string
	Topic       string
	Body        map[string]string
	ProduceTime time.Time
}

func ConnectToKafka() {
	var err error
	kafkaConfig = models.GetConfiguration().Cold.Kafka
	Topics := append(kafkaConfig.Consumer.Topics.DrReport, kafkaConfig.Consumer.Topics.MtReport...)
	KafkaConsumer = connector.ConsumerConfig{
		Brokers:              kafkaConfig.Consumer.Brokers,
		Group:                kafkaConfig.Consumer.GroupName,
		Topics:               Topics,
		Offset:               connector.Offset{LocalOffsetPath: kafkaConfig.Consumer.OffsetPath},
		ReturnError:          true,
		GroupSessionTimeout:  30,
		Log:                  Log.Rotator,
		HandleConsumeMessage: handleMessage,
	}
	err = KafkaConsumer.Connect()

	if err != nil {
		Log.Error(&utils.EventLog{Event: utils.Events.Connect_To_Kafka, Description: utils.Description{Method: "Consumer_Connect", Status: "Error", IP: KafkaConsumer.Brokers, Detail: err.Error()}})
		return
	}
	Log.Info(&utils.EventLog{Event: utils.Events.Connect_To_Kafka, Description: utils.Description{Method: "Consumer_Connect", Status: "Success", IP: KafkaConsumer.Brokers, Detail: map[string]interface{}{
		"Group":  kafkaConfig.Consumer.GroupName,
		"Topics": kafkaConfig.Consumer.Topics,
	}}})
	KafkaProducer = connector.ProducerConfig{
		Brokers:       kafkaConfig.Producer.Brokers,
		Partitioner:   connector.RoundRobinPartitioner,
		RetryMax:      5,
		ReturnSuccess: true,
		Topic:         []string{kafkaConfig.Producer.RefundTopic},
		FailMessage: &connector.FailMessage{
			Path:   kafkaConfig.Producer.FailMessage.Path,
			Format: kafkaConfig.Producer.FailMessage.Format,
			Rotate: kafkaConfig.Producer.FailMessage.Rotate,
		},
	}
	err = KafkaProducer.Connect()
	if err != nil {
		Log.Error(&utils.EventLog{Event: utils.Events.Connect_To_Kafka, Description: utils.Description{Method: "Producer_Connect", Status: "Error", IP: KafkaProducer.Brokers, Detail: err.Error()}})
		return
	}
	Log.Info(&utils.EventLog{Event: utils.Events.Connect_To_Kafka, Description: utils.Description{Method: "Producer_Connect", Status: "Success", IP: KafkaProducer.Brokers}})
}

func ShutDownConsumeToKafka() error {
	return KafkaConsumer.ShutDown()
}
func containsTopic(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
func handleMessage(consume connector.ConsumeData) bool {
	msg := consume.Value
	fmt.Println(kafkaConfig.Consumer.Topics.DrReport)
	fmt.Println(consume.Topic)

	switch {
	case containsTopic(kafkaConfig.Consumer.Topics.DrReport, consume.Topic):

		var consumeData models.ConsumeData
		var profileId, logTid string
		transProcessTime := time.Now()
		logBuf := Log.NewLogBuffer()
		logMsg := utils.TransLog{Event: utils.Events.Receive_Request_From_Kafka}
		description := utils.Description{Method: "Consume_Message", Status: "Success", Detail: map[string]interface{}{"Topic": consume.Topic, "Partition": consume.Partition, "Offset": consume.Offset, "Value": consume.Value}}
		err := json.Unmarshal(msg, &consumeData)

		if err != nil {
			description.Status = "Error"
			description.Detail.(map[string]interface{})["Message"] = err.Error()
			logMsg.Description = description
			Log.Error(&logMsg)
			description.Detail = err.Error()
			logMsg.StartTimeAll = consumeData.InitProcessTime
			logMsg.StartTimeFlow = consumeData.FlowProcessTime
			logMsg.StartTimeTrans = transProcessTime
			WriteSummary(logBuf, logMsg, description)
			return true
		}
		partnerServiceProfile = partnerProfile.GetPartnerServiceProfile()

		switch consumeData.Cmd {
		case "DLVRREP", "SENDMSG":
			consumeData.Cmd = "DLVRREP"
			profileId = consumeData.Cmd + "_" + consumeData.Code
			logTid = consumeData.Tid
			logMsg.Tid = &logTid
			if _, found := partnerServiceProfile.CodeOfProfileId[profileId]; !found {
				description.Status = "Error"
				description.Method = "Get_Profile_ID"
				description.Detail = map[string]interface{}{"ProfileId": profileId, "Message": "ProfileId does not exist"}
				logMsg.Description = description
				Log.Error(&logMsg)
				description.Detail = "ProfileId does not exist"
				logMsg.StartTimeAll = consumeData.InitProcessTime
				logMsg.StartTimeFlow = consumeData.FlowProcessTime
				logMsg.StartTimeTrans = transProcessTime
				WriteSummary(logBuf, logMsg, description)
				return true
			}
		case "DLVRMSG":
			profileId = partnerServiceProfile.ProfileIdOfShortcode[consumeData.To]
			logTid = consumeData.Tid
			logMsg.Tid = &logTid
			if _, found := partnerServiceProfile.ProfileIdOfShortcode[consumeData.To]; !found {
				description.Status = "Error"
				description.Method = "Get_Profile_ID"
				description.Detail = map[string]interface{}{"Message": "ProfileId does not exist"}
				Log.Error(&logMsg)
				description.Detail = "ProfileId does not exist"
				logMsg.StartTimeAll = consumeData.InitProcessTime
				logMsg.StartTimeFlow = consumeData.FlowProcessTime
				logMsg.StartTimeTrans = transProcessTime
				WriteSummary(logBuf, logMsg, description)
				return true
			}
		}
		logMsg.Description = description
		Log.Debug(&logMsg)
		delete(description.Detail.(map[string]interface{}), "Value")
		logMsg.Description = description
		//logMsg.StartTimeTrans = transProcessTime
		Log.Info(&logMsg)

		// Send DR Online and Offline
		//drOnlineError, drOfflineError := DR(logTid, profileId, consumeData, partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Require, partnerServiceProfile.ServiceOfProfileId[profileId].OfflineReport.Require, transProcessTime)
		DR(logTid, profileId, consumeData, *partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Require, *partnerServiceProfile.ServiceOfProfileId[profileId].OfflineReport.Require, transProcessTime)		
		if consumeData.Status == "ERR" {
			if strings.Contains(partnerServiceProfile.ServiceOfProfileId[profileId].Topic, "-SCF") && consumeData.ShortMsg != "" {
				var produceTopic string
				var produceMsg []byte
				produceTopic = models.GetConfiguration().Cold.Kafka.Producer.RefundTopic
				scfData := models.SCFData{Tid: logTid, Body: map[string]string{}}
				scfData.Body["CMD"] = "REFUND"
				scfData.Body["CODE"] = consumeData.Code
				scfData.Body["CHARGE"] = partnerServiceProfile.ServiceOfProfileId[profileId].ChargeNumber
				scfData.Body["VSMID"] = consumeData.Vsmid
				produceMsg, _ = json.Marshal(scfData)
				fmt.Println(produceMsg)
				if err = produceDataToKafka(logTid, produceTopic, string(produceMsg)); err != nil {
					description.Method = "Error"
					description.Detail = err.Error()
					logMsg.StartTimeAll = consumeData.InitProcessTime
					logMsg.StartTimeFlow = consumeData.FlowProcessTime
					logMsg.StartTimeTrans = transProcessTime
					WriteSummary(logBuf, logMsg, description)
					return true
				}
			}
		}

		description.Detail = map[string]interface{}{"Message": "Success"}
		logMsg.StartTimeAll = consumeData.InitProcessTime
		logMsg.StartTimeFlow = consumeData.FlowProcessTime
		logMsg.StartTimeTrans = transProcessTime

		WriteSummary(logBuf, logMsg, description)
		if partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Tps > 0 {
			tpsLock.Lock()
			startTime, countReq = utils.LimitTPS(startTime, countReq, partnerServiceProfile.ServiceOfProfileId[profileId].OnlineReport.Tps)
			tpsLock.Unlock()
		}
	//Consumeing topic [1]
	case containsTopic(kafkaConfig.Consumer.Topics.MtReport, consume.Topic):
		var profileId, logTid string
		var consumeData models.MtReport
		transProcessTime := time.Now()
		logBuf := Log.NewLogBuffer()
		logMsg := utils.TransLog{Event: utils.Events.Receive_Request_From_Kafka}
		description := utils.Description{Method: "Consume_Message", Status: "Success", Detail: map[string]interface{}{"Topic": consume.Topic, "Partition": consume.Partition, "Offset": consume.Offset, "Value": consume.Value}}
		err := json.Unmarshal(msg, &consumeData)
		if err != nil {
			description.Status = "Error"
			description.Detail.(map[string]interface{})["Message"] = err.Error()
			logMsg.Description = description
			Log.Error(&logMsg)
			description.Detail = err.Error()
			logMsg.StartTimeAll = consumeData.InitProcessTime
			logMsg.StartTimeFlow = consumeData.FlowProcessTime
			logMsg.StartTimeTrans = transProcessTime
			WriteSummary(logBuf, logMsg, description)
			return true
		}
		//Set value
		profileId = consumeData.Body["CMD"] + "_" + consumeData.Body["CODE"]
		logTid = consumeData.Tid
		logMsg.Tid = &logTid

		logMsg.Description = description
		Log.Debug(&logMsg)
		delete(description.Detail.(map[string]interface{}), "Value")
		logMsg.Description = description
		Log.Info(&logMsg)
		if !*partnerServiceProfile.ServiceOfProfileId[profileId].MtOfflineReport.Require {
			description.Method = "Skip_Wirte_MT_Report"
			description.Detail.(map[string]interface{})["Message"] = "profile's MT Report value is false"
			logMsg.Description = description
			Log.Info(&logMsg)
		}
		if consumeData.Status == "ERR" {
			if strings.Contains(partnerServiceProfile.ServiceOfProfileId[profileId].Topic, "-SCF") {
				var produceTopic string
				var produceMsg []byte
				produceTopic = models.GetConfiguration().Cold.Kafka.Producer.RefundTopic
				scfData := models.SCFData{Tid: logTid, Body: map[string]string{}}
				scfData.Body["CMD"] = "REFUND"
				scfData.Body["CODE"] = consumeData.Body["CODE"]
				scfData.Body["CHARGE"] = partnerServiceProfile.ServiceOfProfileId[profileId].ChargeNumber
				scfData.Body["VSMID"] = consumeData.Body["VSMID"]
				produceMsg, _ = json.Marshal(scfData)
				if err = produceDataToKafka(logTid, produceTopic, string(produceMsg)); err != nil {
					description.Method = "Error"
					description.Detail = err.Error()
					logMsg.StartTimeAll = consumeData.InitProcessTime
					logMsg.StartTimeFlow = consumeData.FlowProcessTime
					logMsg.StartTimeTrans = transProcessTime
					WriteSummary(logBuf, logMsg, description)
					return true
				}
			}
		}
		description.Detail = map[string]interface{}{"Message": "Success"}
		logMsg.StartTimeAll = consumeData.InitProcessTime
		logMsg.StartTimeFlow = consumeData.FlowProcessTime
		logMsg.StartTimeTrans = transProcessTime
		WriteSummary(logBuf, logMsg, description)
	}
	return true
}

func produceDataToKafka(tid, topic, produceMsg string) error {
	kafkaProcessTime := time.Now()
	logMsg := utils.TransLog{Tid: &tid, Event: utils.Events.Send_Request_To_Kafka}
	description := utils.Description{Method: "Produce_Message", Detail: map[string]interface{}{"Topic": topic}}

	partition, offset, err := KafkaProducer.Produce(topic, "", produceMsg)
	if err != nil {
		description.Status = "Error"
		description.Detail.(map[string]interface{})["Message"] = err.Error()
		logMsg.Description = description
		logMsg.StartTimeTrans = kafkaProcessTime
		Log.Error(&logMsg)
		KafkaProducer.WriteFailMessage(topic, int32(partition), produceMsg)
		return err
	}
	tmpDes := description
	description.Status = "Success"
	description.Detail.(map[string]interface{})["Partition"] = partition
	description.Detail.(map[string]interface{})["Offset"] = offset
	logMsg.Description = description
	logMsg.StartTimeTrans = kafkaProcessTime
	Log.Info(&logMsg)
	description = tmpDes
	return nil
}
