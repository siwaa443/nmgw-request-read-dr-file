package models

import "time"

type ConsumeData struct {
	Tid             string
	From            string
	To              string
	Smid            string
	Vsmid           string
	Code            string
	TotalMsg        int
	Seq             uint32
	Status          string
	Detail          string
	Cmd             string
	ShortMsg        string
	Cdr             bool
	ErrCode         string `json:"ErrCode,omitempty"`
	Fet             string `json:"Fet,omitempty"`
	Ntype           string `json:"Ntype,omitempty"`
	Ctype           string `json:"Ctype,omitempty"`
	Content         string `json:"Content,omitempty"`
	Campaign        string
	TransactionId   string
	InitProcessTime time.Time
	FlowProcessTime time.Time
}

type ChargeRequest struct {
	Action    string
	ShortCode string
}

type SCFData struct {
	Tid  string
	Body map[string]string
}
type MtReport struct {
	Tid             string
	Topic           string
	Body            map[string]string
	Response        map[string]string
	ProduceTime     time.Time
	Smpp            SmppConfig
	Seq             uint32
	InitProcessTime time.Time
	FlowProcessTime time.Time
	Status          string
}
type SmppConfig struct {
	ServiceType        interface{}
	RegisteredDelivery interface{}
	Validity           interface{}
	NumberTon          interface{}
	NumberNpi          interface{}
}
