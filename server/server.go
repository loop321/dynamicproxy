package main

import (
	"time"
	"util"
)

var (
	M = util.NewEncrypt("renproxy")
)

func main() {
	util.InitConfig()
	go Start()
	time.Sleep(999999 * time.Hour)
}
