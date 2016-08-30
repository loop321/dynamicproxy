package main

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
	"util"
)

var UserMap = map[string]*ProxyUser{}
var IDMAPLoack = sync.Mutex{}
var IDMAP = map[string]bool{}

func GetID() string {
	uuid := util.UUID(8)
	//防止重复
	for {
		if !IDMAP[uuid] {
			break
		} else {
			uuid = util.UUID(8)
		}
	}
	IDMAP[uuid] = true
	return uuid
}

//开启代理服务
func Start() {
	//加载各代理
	LoadUserSetting()
	//接收客户端-新连接
	log.Println("配置加载完成,服务启动。")
	serCon, err := net.Listen("tcp", ":"+util.ObjToString(util.Sysconfig["connPort"]))
	util.CheckError(err)
	for {
		cliCon, err := serCon.Accept()
		go func() {
			defer util.Catch()
			util.CheckError(err)
			p, isFull := util.Unpacket(cliCon)
			if !isFull {
				cliCon.Close()
				return
			}
			switch p.Event {
			case util.EVENT_PROXY_CONNECT: //是建立主连接请求
				log.Println("new main client...")
				s_id := util.ObjToString(p.Header["s_id"])
				user := util.ObjToString(p.Header["user"])
				proxy := UserMap[user]
				//判断密码
				bconnect := false
				if proxy != nil {
					var sp *SerProxy
					if s_id == "" { //新请求
						pwd := p.Header["pwd"].(string)
						outer := p.Header["outer"].(string)
						inner := p.Header["inner"].(string)
						if outer != "" && M.DecodeString(pwd) == proxy.Pwd {
							if regexp.MustCompile("^[0-9]+$").MatchString(outer) {
								sp = proxy.AddServe(inner, outer, "")
							} else {
								sp = proxy.AddServe(inner, "80", outer)
							}
						}
					} else {
						sp = proxy.Setting[s_id]
					}
					if sp != nil {
						sp.MainConn = cliCon
						bconnect = true
						sp.MainConn.Write(util.Enpacket(util.EVENT_PING, map[string]interface{}{
							"s_id": sp.ID,
						}, nil))
					}
				}

				if !bconnect {
					cliCon.Write(util.Enpacket(util.EVENT_ERR_PROXY_CONNECT, map[string]interface{}{
						"msg": "用户|密码|端口等信息不存在" + fmt.Sprintf("%v", p.Header),
					}, nil))
					cliCon.Close()
				}
			case util.EVENT_TRANSPORT_CONNECT: //建立子连接,连续访问会出粘包的问题
				//此info是之前服务端发送过去又返回来的
				user := util.ObjToString(p.Header["user"])
				s_id := util.ObjToString(p.Header["s_id"])
				proxy := UserMap[user]
				if proxy != nil {
					serp := proxy.Setting[s_id]
					if serp != nil {
						//只在存在有用户下确认连接，有问题
						_, err := cliCon.Write(util.Enpacket(util.EVENT_PING, nil, nil))
						if err != nil {
							log.Println(err)
						} else {
							tcp := NewTcp(serp, cliCon)
							serp.ClientSess <- tcp
							serp.Timestamp = time.Now().Unix()
						}
					}
				}
			}
		}()
	}
}

type ProxyUser struct {
	Setting map[string]*SerProxy
	Lock    sync.Mutex
	User    string
	Pwd     string
}

//加载用户设置,暂不允许共用端口
func LoadUserSetting() {
	sets := util.Sysconfig["userSetting"].(map[string]interface{})
	for key, set := range sets {
		keys := strings.Split(key, ",")
		pu := &ProxyUser{
			Setting: map[string]*SerProxy{},
			User:    keys[0],
			Pwd:     keys[1],
		}
		UserMap[pu.User] = pu
		usets := set.([]interface{})
		for _, s := range usets {
			obj := s.(map[string]interface{})
			psp := NewSerProxy(obj["innerport"].(string), obj["outerport"].(string), "", pu)
			pu.Setting[psp.ID] = psp
		}
		//启动每个用户的服务
		go pu.OuterServe()
	}
}

//建立服务
func (p *ProxyUser) OuterServe() {
	for _, set := range p.Setting {
		go set.Deal(p)
	}
}

//添加监听//outer 80端口特殊处理
func (p *ProxyUser) AddServe(inner, outer, doamin string) *SerProxy {
	defer util.Catch()
	p.Lock.Lock()
	defer p.Lock.Unlock()
	psp := NewSerProxy(inner, outer, doamin, p)
	if psp != nil {
		p.Setting[psp.ID] = psp
		go psp.Deal(p)
		go psp.GC()
	}
	return psp
}

func GetSerKey(user, inner, outer, outer80 string) string {
	return fmt.Sprintf("%s_%s_%s_%s", user, inner, outer, outer80)
}
