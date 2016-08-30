package main

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"strings"
	"time"
	"util"
)

var (
	Content_Length = "Content-Length: "
	bufsize        = 64 * 1024
)

type LocalConn struct {
	LP   *LocalProxy
	Conn net.Conn
}

func (lc *LocalConn) Connect() (res bool) {
	defer util.Catch()
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Println("建立-子-连接-出错-con:", err)
		return
	}
	_, err = conn.Write(util.Enpacket(util.EVENT_TRANSPORT_CONNECT, map[string]interface{}{
		"s_id": lc.LP.ID,
		"user": lc.LP.User,
	}, nil))
	if err != nil {
		log.Println("建立-子-连接-出错-write:", err)
		return
	}
	ret, bcon := util.Unpacket(conn)
	if !bcon || ret.Event != util.EVENT_PING {
		log.Println("连接失败...")
		return
	}
	lc.Conn = conn
	return true
}

func (lc *LocalConn) Start() {
	defer util.Catch()
	defer lc.Conn.Close()
OVER:
	for {
		p, bFull := util.Unpacket(lc.Conn)
		if bFull {
			switch p.Event {
			case util.EVENT_PING:
				//log.Println("localconn-ping:", lc.LP.Name, "ping...")
			case util.EVENT_TRANSPORT_START:
				pc, err := net.Dial("tcp", lc.LP.Inner)
				if err != nil {
					log.Println("连接端口|服务失败", lc.LP.Inner)
				} else {
					//log.Printf("[+]: (%v)\n", pc.LocalAddr())
					if !lc.Transport(pc) {
						log.Println("关闭连接...")
						break OVER
					}
					//log.Printf("[-]: (%v)\n", pc.LocalAddr())
				}
			}
		} else {
			break OVER
		}
	}
}

func (lc *LocalConn) Transport(httpcon net.Conn) bool {
	go func() {
		defer util.Catch()
		//只读一次！！！不支持上传文件 ！！--服务端也一样
		res, isFull := util.Unpacket(lc.Conn)
		if !isFull {
			return
		}
		if res.Event == util.EVENT_TRANSPORT_DATA && len(res.Raw) > 0 {
			nw, ew := httpcon.Write(res.Raw)
			if ew != nil {
				log.Println("ew", ew)
				return
			}
			if len(res.Raw) != nw {
				log.Println("nr!=nw")
				return
			}
		}
	}()
	func() {
		defer util.Catch()
		buf := make([]byte, bufsize)
		nr, er := httpcon.Read(buf)
		if nr < 1 || er != nil {
			log.Println(nr, er)
			return
		}
		itype := 0
		consize := 0
		readn := 0
		hpos := 0
		bread := bufio.NewReader(bytes.NewReader(buf[:nr]))
		for i := 0; i < 12; i++ {
			res, _, err := bread.ReadLine()
			hpos += len(res) + 2
			resStr := string(res)
			if err != nil {
				itype = -1
				break
			}
			if itype == 0 {
				ipos := strings.Index(resStr, Content_Length)
				if ipos > -1 {
					itype = 1
					consize = util.IntAll(resStr[ipos+len(Content_Length):])
					continue
				} else if strings.Index(resStr, "Transfer-Encoding: chunked") > -1 {
					itype = 2
					break
				}
			}
			if resStr == "" {
				if itype == 1 {
					header := hpos
					readn = consize - (nr - header)
					//log.Println("=====header=============", header, strings.Index(string(buf[:nr]), "\r\n\r\n")+4, readn, consize)
				}
				break
			}
		}
		nw, ew := lc.Conn.Write(util.Enpacket(util.EVENT_TRANSPORT_DATA, nil, buf[0:nr]))
		if ew != nil {
			log.Println("ew2", ew)
			return
		}
		if nr+util.HLEN != nw {
			log.Println("nr!=nw2")
			return
		}
	M:
		switch itype {
		case -1:
			break
		case 0:
			httpcon.SetDeadline(time.Now().Add(20 * time.Second))
			for {
				nr, er = httpcon.Read(buf)
				if nr > 0 {
					nw, ew := lc.Conn.Write(util.Enpacket(util.EVENT_TRANSPORT_DATA, nil, buf[0:nr]))
					if ew != nil {
						log.Println("ew2", ew)
						break M
					}
					if nr+util.HLEN != nw {
						log.Println("nr!=nw2")
						break M
					}
				} else {
					break M
				}
				if er != nil {
					log.Println("er2", er)
					break M
				}
			}
		case 1:
			for {
				if readn > 0 {
					nr, er := httpcon.Read(buf)
					if nr > 0 {
						nw, ew := lc.Conn.Write(util.Enpacket(util.EVENT_TRANSPORT_DATA, nil, buf[0:nr]))
						if ew != nil {
							log.Println("ew2", ew)
							break M
						}
						if nr+util.HLEN != nw {
							log.Println("nr!=nw2")
							break M
						}
					} else {
						break M
					}
					if er != nil {
						log.Println("er2", er)
						break M
					}
					readn -= nr
				} else {
					break M
				}
			}
		case 2:
			for {
				if nr > 6 {
					if string(buf[nr-7:nr]) == "\r\n0\r\n\r\n" {
						break
					}
				}
				nr, er = httpcon.Read(buf)
				if nr > 0 {
					nw, ew := lc.Conn.Write(util.Enpacket(util.EVENT_TRANSPORT_DATA, nil, buf[0:nr]))
					if ew != nil {
						log.Println("ew2", ew)
						break M
					}
					if nr+util.HLEN != nw {
						log.Println("nr!=nw2")
						break M
					}
				} else {
					break M
				}
				if er != nil {
					log.Println("er2", er)
					break M
				}
			}
		}
	}()
	httpcon.Close()
	_, er := lc.Conn.Write(util.Enpacket(util.EVENT_TRANSPORT_DATA_END, nil, nil))
	if er != nil {
		log.Println("结束请求出错..", er)
		return false
	}
	return true
}
