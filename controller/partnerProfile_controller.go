package controller

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	partnerProfile "git.matador.ais.co.th/cncm/nmgw-partner-profile-management"
	utils "git.matador.ais.co.th/cncm/nmgw-utililty-library/v2"
	"github.com/siwaa443/nmgw-request-read-dr-file/models"
)

func InitPartnerProfile() {
	partnerProfileNotifyCh := make(chan interface{})

	partnerProfile.Init(false, models.GetConfiguration().Cold.Mongo, Log, partnerProfileNotifyCh, nil, nil, nil) //conect mongo and working about change steam
	manageHttpClientForSendDR()
	manageHttpClientForSendMT()
	go func(changeStream <-chan interface{}) {
		for {
			<-changeStream
			ReloadDr("Reload_Dr&MT_Offline")
			manageHttpClientForSendDR()
			manageHttpClientForSendMT()
		}
	}(partnerProfileNotifyCh)
}

func manageHttpClientForSendDR() {
	for profileId, service := range partnerProfile.GetPartnerServiceProfile().ServiceOfProfileId {
		var allHost string
		for _, v := range service.OnlineReport.Endpoint.Url {
			urlParsed, _ := url.Parse(v)
			if _, ok := httpClientDR[profileId][urlParsed.Host]; ok {
				clientConfig := utils.FastHTTPClient{
					MaxConnsPerHost:    *service.OnlineReport.Endpoint.MaxConcurrent,
					MaxConnWaitTimeout: time.Duration(*service.OnlineReport.Endpoint.ConcurrentTimeout) * time.Millisecond,
					ReadTimeout:        time.Duration(*service.OnlineReport.Endpoint.Timeout) * time.Millisecond,
					WriteTimeout:       time.Duration(*service.OnlineReport.Endpoint.Timeout) * time.Millisecond,
				}

				if urlParsed.Scheme == "https" {
					clientConfig.InsecureSkipVerify = *service.OnlineReport.Endpoint.InsecureSkipVerify
					if (service.OnlineReport.Endpoint.Cert != nil && *service.OnlineReport.Endpoint.Cert != "") && !clientConfig.InsecureSkipVerify {
						clientConfig.Cert = *service.OnlineReport.Endpoint.Cert
					}
				}
				httpClientDR[profileId][urlParsed.Host], _ = utils.InitHTTPClient(clientConfig)
			} else {
				if len(httpClientDR[profileId]) == 0 {
					httpClientDR[profileId] = make(map[string]*utils.FastHTTPClient)
				}
				clientConfig := utils.FastHTTPClient{
					MaxConnsPerHost:    *service.OnlineReport.Endpoint.MaxConcurrent,
					MaxConnWaitTimeout: time.Duration(*service.OnlineReport.Endpoint.ConcurrentTimeout) * time.Millisecond,
					ReadTimeout:        time.Duration(*service.OnlineReport.Endpoint.Timeout) * time.Millisecond,
					WriteTimeout:       time.Duration(*service.OnlineReport.Endpoint.Timeout) * time.Millisecond,
				}

				if urlParsed.Scheme == "https" {
					clientConfig.InsecureSkipVerify = *service.OnlineReport.Endpoint.InsecureSkipVerify
					if (service.OnlineReport.Endpoint.Cert != nil && *service.OnlineReport.Endpoint.Cert != "") && !clientConfig.InsecureSkipVerify {
						clientConfig.Cert = *service.OnlineReport.Endpoint.Cert
					}
				}
				httpClientDR[profileId][urlParsed.Host], _ = utils.InitHTTPClient(clientConfig)

			}
			allHost += fmt.Sprintf("%s ", urlParsed.Host)
		}
		// Clear host
		for hostName := range httpClientDR[profileId] {
			if !strings.Contains(allHost, hostName) {
				delete(httpClientDR[profileId], hostName)
			}
		}
	}
}
func manageHttpClientForSendMT() {
	for profileId, service := range partnerProfile.GetPartnerServiceProfile().ServiceOfProfileId {
		var allHost string
		for _, v := range service.MtOnlineReport.Endpoint.Url {
			urlParsed, _ := url.Parse(v)
			if _, ok := httpClientMT[profileId][urlParsed.Host]; ok {
				clientConfig := utils.FastHTTPClient{
					MaxConnsPerHost:    *service.MtOnlineReport.Endpoint.MaxConcurrent,
					MaxConnWaitTimeout: time.Duration(*service.MtOnlineReport.Endpoint.ConcurrentTimeout) * time.Millisecond,
					ReadTimeout:        time.Duration(*service.MtOnlineReport.Endpoint.Timeout) * time.Millisecond,
					WriteTimeout:       time.Duration(*service.MtOnlineReport.Endpoint.Timeout) * time.Millisecond,
				}

				if urlParsed.Scheme == "https" {
					clientConfig.InsecureSkipVerify = *service.MtOnlineReport.Endpoint.InsecureSkipVerify
					if (service.MtOnlineReport.Endpoint.Cert != nil && *service.MtOnlineReport.Endpoint.Cert != "") && !clientConfig.InsecureSkipVerify {
						clientConfig.Cert = *service.MtOnlineReport.Endpoint.Cert
					}
				}
				httpClientMT[profileId][urlParsed.Host], _ = utils.InitHTTPClient(clientConfig)
			} else {
				if len(httpClientMT[profileId]) == 0 {
					httpClientMT[profileId] = make(map[string]*utils.FastHTTPClient)
				}
				clientConfig := utils.FastHTTPClient{
					MaxConnsPerHost:    *service.MtOnlineReport.Endpoint.MaxConcurrent,
					MaxConnWaitTimeout: time.Duration(*service.MtOnlineReport.Endpoint.ConcurrentTimeout) * time.Millisecond,
					ReadTimeout:        time.Duration(*service.MtOnlineReport.Endpoint.Timeout) * time.Millisecond,
					WriteTimeout:       time.Duration(*service.MtOnlineReport.Endpoint.Timeout) * time.Millisecond,
				}

				if urlParsed.Scheme == "https" {
					clientConfig.InsecureSkipVerify = *service.MtOnlineReport.Endpoint.InsecureSkipVerify
					if (service.MtOnlineReport.Endpoint.Cert != nil && *service.MtOnlineReport.Endpoint.Cert != "") && !clientConfig.InsecureSkipVerify {
						clientConfig.Cert = *service.MtOnlineReport.Endpoint.Cert
					}
				}
				httpClientMT[profileId][urlParsed.Host], _ = utils.InitHTTPClient(clientConfig)

			}
			allHost += fmt.Sprintf("%s ", urlParsed.Host)
		}
		// Clear host
		for hostName := range httpClientMT[profileId] {
			if !strings.Contains(allHost, hostName) {
				delete(httpClientMT[profileId], hostName)
			}
		}
	}
}