package models

import (
	partnerProfile "git.matador.ais.co.th/cncm/nmgw-partner-profile-management"
)

type Configuration struct {
	Cold ColdConfiguration
	Warm WarmConfiguration
}

type ColdConfiguration struct {
	Kafka KafkaConfiguration
	Mongo partnerProfile.MongoConfig
}

type WarmConfiguration struct {
	Log LogConfiguration
	Dr  DrConfiguration
	Mt  MtConfiguration
}

type KafkaConfiguration struct {
	Consumer KafkaConsumerConfiguration
	Producer KafkaProducerConfiguration
}

type KafkaConsumerConfiguration struct {
	Brokers    []string
	GroupName  string
	Topics     Topics
	OffsetPath string
}
type Topics struct {
	DrReport []string
	MtReport []string
}
type KafkaProducerConfiguration struct {
	Brokers     []string
	RefundTopic string
	FailMessage FailMessageConfiguration
}

type FailMessageConfiguration struct {
	Path   string
	Format string
	Rotate string
}

type LogConfiguration struct {
	Enable bool
	Path   string
	Format string
	Level  string
	Rotate string
}
type DrConfiguration struct {
	Path   string
	Format string
	Rotate string
}
type MtConfiguration struct {
	Path       string
	MaxRecords int
	Rotate     string
}

var (
	configuration Configuration
)

func GetConfiguration() Configuration {
	return configuration
}

func SetConfiguration(config Configuration) {
	configuration = config
}
