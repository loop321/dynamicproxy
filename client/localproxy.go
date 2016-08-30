package main

import (
	"log"
	"net"
	"os"
	"strings"
	"time"
	"util"
)

type LocalProxy struct {
	ID            string
	conn          net.Conn
	MaxRetries    int
	retrytimes    int
	Name          string
	Inner         string
	Outer         string
	Maxconn       int
	User          string
	Pwd           string
	LocalConnPool chan *LocalConn
	MapInfo       map[string]interface{}
}

func NewLocalProxy(inner, outer, user, pwd string) *LocalProxy {
	if strings.Index(inner, ":") < 1 {
		inner += ":80"
	}
	lp := &LocalProxy{
		Inner:         inner,
		Outer:         outer,
		Maxconn:       100,
		MaxRetries:    3,
		User:          user,
		Pwd:           pwd,
		LocalConnPool: make(chan *LocalConn, 150),
		Name:          "localproxy_" + inner + "_" + outer,
	}
	lp.MapInfo = map[string]interface{}{
		"inner": inner,
		"outer": outer,
		"user":  user,
		"pwd":   pwd,
	}
	return lp
}

func (loc *LocalProxy) Connect() bool {
	defer util.Catch()
	if loc.retrytimes > loc.MaxRetries {
		log.Println("重连超时")
		return false
	}
	bconnect := false
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Println("建立-主-连接-出错:", err)
		return false
	}
	log.Println(loc.MapInfo)
	_, err = conn.Write(util.Enpacket(util.EVENT_PROXY_CONNECT, loc.MapInfo, nil))
	if err != nil {
		log.Println("连接-发送-出错:", err)
	}
	ret, bcon := util.Unpacket(conn)
	if !bcon || ret.Event != util.EVENT_PING || ret.Header == nil {
		log.Println("连接失败...", ret.Event, ret)
		if ret.Event == util.EVENT_ERR_PROXY_CONNECT {
			log.Println(ret.Header["msg"])
			os.Exit(1)
		}
	} else {
		//赋新ID
		loc.ID = ret.Header["s_id"].(string)
		loc.MapInfo["s_id"] = loc.ID
		bconnect = true
	}
	if !bconnect {
		loc.retrytimes++
		return loc.Connect()
	} else {
		loc.conn = conn
		return bconnect
	}
}

func (loc *LocalProxy) Start() {
	defer util.Catch()
	log.Println("启动代理客户端:", loc.Name)
	for {
		bconnect := loc.Connect()
		if bconnect {
			for {
				p, bFull := util.Unpacket(loc.conn)
				if bFull {
					switch p.Event {
					case util.EVENT_PING:
						//log.Println("proxy-ping:", loc.Name, "ping...")
						//loc.conn.Write(util.Enpacket(util.EVENT_PING, loc.MapInfo, nil))
					case util.EVENT_REQ_TRANSPORT_CONNECT:
						for i := 0; i < 6; i++ {
							lc := &LocalConn{
								LP: loc,
							}
							if lc.Connect() {
								go lc.Start()
							}
						}
					}
				} else {
					break
				}
			}
		}
		time.Sleep(60 * time.Second)
	}
}
