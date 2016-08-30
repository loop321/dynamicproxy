package main

import (
	"fmt"
	"log"
	"net"
	"time"
	"util"
)

type SerProxy struct {
	MainConn    net.Conn //主连接
	OuterPort   string   //外网端口
	OuterPort80 string
	OuterDomain string
	InnerPort   string        //内网应用地址
	ClientSess  chan *TCPConn //代理端连接池
	Name        string        //代理名称
	Listen      net.Listener
	Timestamp   int64
	P           *ProxyUser
	ID          string
}

var AcceptNum = 350

func (s *SerProxy) GetOuterPort() string {
	if s.OuterPort == "80" {
		return s.OuterPort80
	}
	return s.OuterPort
}

//没有更新及时关闭
func (s *SerProxy) GC() {
	defer util.Catch()
	for {
		time.Sleep(3 * time.Second)
		now := time.Now().Unix()
		_, err := s.MainConn.Write(util.Enpacket(util.EVENT_PING, nil, nil))
		if err != nil {
			if now-s.Timestamp > 12 {
				go s.Destory()
				break
			}
		} else {
			s.Timestamp = now
		}
	}
}

func (s *SerProxy) GetClient() *TCPConn {
	var responesConn *TCPConn
	select {
	case responesConn = <-s.ClientSess:
		go fmt.Sprint("_.")
		break
	case <-time.After(1 * time.Millisecond):
		go fmt.Sprint("_,")
		//一次取多个连接备用
		_, err := s.MainConn.Write(util.Enpacket(util.EVENT_REQ_TRANSPORT_CONNECT, map[string]interface{}{
			"s_id": s.ID,
		}, nil))
		util.CheckError(err)
		//s.Destory()
		//s.MainConn.Close()
		select {
		case responesConn = <-s.ClientSess:
			//log.Println("新建连接.成功.")
			break
		case <-time.After(2 * time.Second):
			//log.Println("新建连接.超时.")
			break
		}
	}
	return responesConn
}

//放回连接
func (s *SerProxy) RecycleClient(t *TCPConn) {
	_, err := t.Conn.Write(util.Enpacket(util.EVENT_PING, map[string]interface{}{
		"s_id": s.ID,
	}, nil))
	if err == nil {
		select {
		case s.ClientSess <- t:
			//log.Println("释放连接.....success")
			break
		case <-time.After(3 * time.Second):
			t.Conn.Close()
			break
		}
	} else {
		log.Println(err)
		t.Conn.Close()
	}
}

func (s *SerProxy) Destory() {
	defer util.Catch()
	defer s.MainConn.Close()
	for i := 0; i < AcceptNum; i++ {
		b := true
		select {
		case t := <-s.ClientSess:
			t.Conn.Close()
			close(s.ClientSess)
		case <-time.After(5 * time.Second):
			b = false
		}
		if !b {
			break
		}
	}
	delete(IDMAP, s.ID)
	s.Listen.Close()
	s = nil
}

func (s *SerProxy) Deal(p *ProxyUser) {
	defer util.Catch()
	defer s.Destory()
	log.Println("启动端口", s.GetOuterPort())
	ln, err := net.Listen("tcp", ":"+s.GetOuterPort())
	s.Listen = ln
	if err != nil {
		log.Println(err, s, s.GetOuterPort())
		return
	}
	fmt.Printf("启动服务:\n\t%s-%s\n\t外网端口:%s\n\t内网代理:%s\n", p.User, s.Name, s.OuterPort, s.InnerPort)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err, "出错,停止监听...")
			break
		}
		go func() {
			defer util.Catch()
			defer conn.Close()
			//defer log.Println("[-]:", conn.RemoteAddr())
			//log.Println(s.Name, "accept new requset ", s.OuterPort)
			//log.Println("[+]:", conn.RemoteAddr())
			//请求新连接
			responesTcp := s.GetClient()
			if responesTcp != nil {
				responesTcp.TransportCon(conn)
			}
		}()
	}
}

func NewSerProxy(inner, outer, doamin string, p *ProxyUser) *SerProxy {
	psp := &SerProxy{
		OuterPort:  outer,
		InnerPort:  inner,
		ID:         GetID(),
		Name:       "proxy_" + outer,
		ClientSess: make(chan *TCPConn, AcceptNum),
		Timestamp:  time.Now().Unix(),
		P:          p,
	}
	if outer == "80" {
		pt := GetPort()
		if pt != "" && doamin != "" {
			psp.OuterPort80 = pt
			psp.OuterDomain = doamin
			AddToNgin(doamin, pt)
		} else {
			//应该给对方回复
			log.Println("未找到合适端口，添加失败...")
			return nil
		}
	} else {
		psp.OuterPort80 = psp.OuterPort
	}

	if psp.OuterDomain == "" {
		psp.OuterDomain = "localhost"
	}
	return psp
}

func NewTcp(s *SerProxy, conn net.Conn) *TCPConn {
	tcp := &TCPConn{
		Conn: conn,
		Psp:  s,
	}

	return tcp
}

type TCPConn struct {
	Conn net.Conn
	Psp  *SerProxy
}

func (t *TCPConn) TransportCon(httpcon net.Conn) {
	httpcon.SetDeadline(time.Now().Add(20 * time.Second))
	defer func() {
		go t.Psp.RecycleClient(t)
	}()
	go func() {
		defer util.Catch()
		_, err := t.Conn.Write(util.Enpacket(util.EVENT_TRANSPORT_START, nil, nil))
		if err != nil {
			log.Println("---err---new request", err)
			return
		}
		buf := make([]byte, 64*1024)
		nr, er := httpcon.Read(buf)
		if nr > 0 {
			nw, ew := t.Conn.Write(util.Enpacket(util.EVENT_TRANSPORT_DATA, nil, buf[0:nr]))
			if ew != nil {
				log.Println("ew", ew)
				return
			}
			if nr+util.HLEN != nw {
				log.Println("nr!=nw", nr+util.HLEN, nw)
				return
			}
		}
		if er != nil {
			log.Println("er", er)
		}
	}()
	func() {
		defer util.Catch()
	L:
		for {
			res, isFull := util.Unpacket(t.Conn)
			if !isFull {
				break L
			}
			switch res.Event {
			case util.EVENT_TRANSPORT_DATA:
				if len(res.Raw) > 0 {
					nw, ew := httpcon.Write(res.Raw)
					if ew != nil {
						log.Println("ew2", ew)
						break L
					}
					if len(res.Raw) != nw {
						log.Println("res!=nw_2", len(res.Raw), nw)
						break L
					}
				} else {
					break L
				}
			default:
				//case protocol.TransportEnd:
				break L
			}
		}
	}()
}
