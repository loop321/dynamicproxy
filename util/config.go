package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

var Sysconfig map[string]interface{}

func InitConfig() {
	ReadConfig(&Sysconfig)
}

//读取配置文件
func ReadConfig(config ...interface{}) {
	var r *os.File
	filepath := "./config.json"
	pos := 0
	if len(config) > 1 {
		filepath, _ = config[pos].(string)
		pos++
	}
	r, _ = os.Open(filepath)
	defer r.Close()
	bs, _ := ioutil.ReadAll(r)
	json.Unmarshal(bs, config[pos])
}
