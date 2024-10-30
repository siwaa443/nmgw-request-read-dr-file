package controller

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"time"

	utils "git.matador.ais.co.th/cncm/nmgw-utililty-library/v2"
	"github.com/siwaa443/nmgw-request-read-dr-file/models"
)

var httpClientDR = make(map[string]map[string]*utils.FastHTTPClient)
var httpClientMT = make(map[string]map[string]*utils.FastHTTPClient)
type Request struct {
	Method      string
	ContentType string
	Header      map[string]string
	Url         string
	Timeout     int
	Tid         string
	Body        []byte
}

func DRRequest(profileId string, data Request, tid string) (resp string, status string, detail string, err error) {
	var jsonstruct models.JSONformat
	var xmlstruct models.XMLformat
	var res []byte
	data.Header["X-Tid"] = tid
	urlParsed, _ := url.Parse(data.Url)
	if httpClientDR[profileId] != nil && httpClientDR[profileId][urlParsed.Host] != nil {
		httpClientDR[profileId][urlParsed.Host].ReadTimeout = time.Duration(data.Timeout)
		httpClientDR[profileId][urlParsed.Host].WriteTimeout = time.Duration(data.Timeout)
		res, err = httpClientDR[profileId][urlParsed.Host].SendRquest(&utils.HttpRequest{
			Url:         &data.Url,
			Headers:     data.Header,
			Method:      data.Method,
			ContentType: data.ContentType,
			Body:        data.Body,
		})
		if err != nil {
			return "", "", "", err
		}
	} else {
		return "", "", "", fmt.Errorf("httpClient is nil")
	}

	// Handle body response
	resp = string(res)
	if err = json.Unmarshal(res, &jsonstruct); err != nil {
		err = xml.Unmarshal(res, &xmlstruct)
		if err != nil {
			return resp, status, detail, fmt.Errorf("Invalid Response Format")
		}
		status = xmlstruct.Status
		detail = xmlstruct.Detail
	} else {
		status = jsonstruct.Status
		detail = jsonstruct.Detail
	}

	return resp, status, detail, err
}
