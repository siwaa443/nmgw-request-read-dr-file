package main

import "github.com/siwaa443/nmgw-request-read-dr-file/controller"

func init() {
	controller.InitConfig()
	controller.InitLog()
	controller.InitPartnerProfile()
	controller.InitDr()
	controller.ConnectToKafka()
}

func main(){
	controller.Channel()
}