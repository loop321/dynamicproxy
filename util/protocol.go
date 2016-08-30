package util

import (
	"encoding/json"
	"io"
	"log"
	"net"
)

const (
	EVENT_PING                  byte = iota //ping 连接
	EVENT_PROXY_CONNECT                     //建立代理连接
	EVENT_REQ_TRANSPORT_CONNECT             //请求建立传输连接
	EVENT_TRANSPORT_CONNECT                 //建立传输连接
	EVENT_TRANSPORT_START                   //传输数据
	EVENT_TRANSPORT_DATA                    //传输数据
	EVENT_TRANSPORT_DATA_END                //传输数据结束
	EVENT_ERR_PROXY_CONNECT
)

//包体
type Packet struct {
	PacketLength int32                  //4 bit 包长度
	Event        byte                   //1 bit 事件
	HeaderLength int32                  //4 bit 头长度
	Header       map[string]interface{} //头
	Raw          []byte                 //数据
}

var (
	KB   = int32(1024)
	MB   = int32(1000 * KB)
	HLEN = 9
)

//封包
func Enpacket(event byte, header map[string]interface{}, data interface{}) []byte {
	var ret, dataByte, headerByte []byte
	if data != nil {
		if v, ok := data.([]byte); ok {
			dataByte = v
		} else if v, ok := data.(string); ok {
			dataByte = []byte(v)
		} else {
			dataByte, _ = json.Marshal(data)
		}
	}
	if header != nil {
		headerByte, _ = json.Marshal(header)
	}
	ret = append(Int2Byte(int32(5 + len(dataByte) + len(headerByte)))) //event+headerlength+len(d)+len(h)
	ret = append(ret, event)
	ret = append(ret, Int2Byte(int32(len(headerByte)))...)
	if header != nil {
		ret = append(ret, headerByte...)
	}
	if data != nil {
		ret = append(ret, dataByte...)
	}
	return ret
}

//

//解包
func Unpacket(conn net.Conn) (p *Packet, isFull bool) {
	defer Catch()
	Hbuffer := make([]byte, 4)
	if n, err := io.ReadFull(conn, Hbuffer); err != nil || n != 4 {
		log.Println("Read-Size-Fullerr", err)
		return
	}
	size := Byte2Int(Hbuffer)
	if size < 5 || size > 10*MB {
		log.Println("Size-Error[", size, "]")
		return
	}
	p = &Packet{}
	p.PacketLength = size
	ret := make([]byte, size)
	readlen, err := io.ReadFull(conn, ret)
	if err != nil || int32(readlen) != size {
		log.Println("Read-Packet-Error", err)
		return
	}
	p.Event = ret[0]
	p.HeaderLength = Byte2Int(ret[1:5])
	if p.HeaderLength > 0 {
		if 5+p.HeaderLength > int32(len(ret)) {
			log.Println("---header---length---", p.HeaderLength, len(ret))
		}
		json.Unmarshal(ret[5:5+p.HeaderLength], &(p.Header))
	}
	if size > 5+p.HeaderLength {
		p.Raw = ret[5+p.HeaderLength:]
	}
	isFull = true
	return
}
