package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"
	"util"
)

var (
	serverAddr  = "127.0.0.1:11179"
	settingConf = flag.String("f", "", "配置文件路径")
	addr        = flag.String("addr", "127.0.0.1:11179", "指定服务地址，如：127.0.0.1:11179")
	user        = flag.String("user", "ren", "指定用户")
	pwd         = flag.String("pwd", "", "指定用户密码")
	inner       = flag.String("inner", "", "内网代理应用(如127.0.0.1:99,多个用|分开)")
	outer       = flag.String("outer", "", "外网代理端口(可以是端口/域名,多个用|分开)")
	M           = util.NewEncrypt("renproxy")
)

func main() {
	flag.PrintDefaults()
	flag.Parse()
	if len(os.Args) == 1 {
		*settingConf = "./config.json"
	}
	if *settingConf != "" {
		log.Println(*settingConf)
		util.ReadConfig(*settingConf, &util.Sysconfig)
		*addr = util.Sysconfig["addr"].(string)
		*user = util.Sysconfig["user"].(string)
		*pwd = util.Sysconfig["pwd"].(string)
		*inner = util.Sysconfig["inner"].(string)
		*outer = util.Sysconfig["outer"].(string)
	}
	serverAddr = *addr
	inners := strings.Split(*inner, "|")
	outers := strings.Split(*outer, "|")
	if len(inners) == len(outers) {
		for i := 0; i < len(inners); i++ {
			LP := NewLocalProxy(inners[i], outers[i], *user, M.EncodeString(*pwd))
			go LP.Start()
		}
	}
	time.Sleep(999999 * time.Hour)
}
