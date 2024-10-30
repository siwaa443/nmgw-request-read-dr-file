package models

import (
	"encoding/xml"
)

// Request JSON
type BodyJsonFormat struct {
	Transid string `json:"TRANSID"`
	Cmd     string `json:"CMD"`
	Fet     string `json:"FET"`
	Ntype   string `json:"NTYPE"`
	From    string `json:"FROM"`
	To      string `json:"TO"`
	Code    string `json:"CODE"`
	Ctype   string `json:"CTYPE"`
	Content string `json:"CONTENT"`
}
type BodyJsonFormatMt struct {
	Tid           string
	From          string
	Charge        string
	To            string
	Vsmid         string
	Submitdate    string
	Content       string
	ExpiryDate    string
	CountyrCode string
	CampaignName  string
	TransactionId string
}

// Response XML
type XMLformat struct {
	XMLName xml.Name `xml:"XML"`
	Status  string   `xml:"STATUS"`
	Detail  string   `xml:"DETAIL"`
}

// Response JSON
type JSONformat struct {
	Status string `json:"Status,omitempty"`
	Detail string `json:"Detail,omitempty"`
}

type SMPPFormat struct {
	Seq          uint32        `json:"seq"`
	Tid          string        `json:"tid"`
	Code         string        `json:"code"`
	To           string        `json:"to"`
	SenderName   string        `json:"senderName"`
	ProcessTime  ProcessTimeDR `json:"processTime"`
	ShortMessage string        `json:"shortmessage"`
}

type ProcessTimeDR struct {
	InitProcessTime string `json:"initProcessTime"`
	FlowProcessTime string `json:"flowProcessTime"`
}
