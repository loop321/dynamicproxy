package util

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"log"
	"runtime"
	"strconv"
)

//出错拦截
func Catch() {
	if r := recover(); r != nil {
		log.Println(r)
		for skip := 0; ; skip++ {
			_, file, line, ok := runtime.Caller(skip)
			if !ok {
				break
			}
			go log.Printf("%v,%v\n", file, line)
		}
	}
}

func ObjToString(old interface{}) string {
	if nil == old {
		return ""
	} else {
		return old.(string)
	}
}

func IntAll(num interface{}) int {
	if i, ok := num.(int); ok {
		return int(i)
	} else if i0, ok0 := num.(int32); ok0 {
		return int(i0)
	} else if i1, ok1 := num.(float64); ok1 {
		return int(i1)
	} else if i2, ok2 := num.(int64); ok2 {
		return int(i2)
	} else if i3, ok3 := num.(float32); ok3 {
		return int(i3)
	} else if i4, ok4 := num.(string); ok4 {
		in, _ := strconv.Atoi(i4)
		return int(in)
	} else if i5, ok5 := num.(int16); ok5 {
		return int(i5)
	} else if i6, ok6 := num.(int8); ok6 {
		return int(i6)
	} else {
		return 0
	}
}

func CheckError(err error, info ...interface{}) {
	if err != nil {
		go log.Println(info, err.Error())
	}
}

//
func Byte2Int(src []byte) int32 {
	var ret int32
	binary.Read(bytes.NewReader(src), binary.BigEndian, &ret)
	return ret
}

//
func Int2Byte(src int32) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, src)
	return buf.Bytes()
}

type SimpleEncrypt struct {
	Key string
}

func NewEncrypt(key string) *SimpleEncrypt {
	return &SimpleEncrypt{key}
}

func (s *SimpleEncrypt) EncodeString(str string) string {
	bs := []byte(str)
	s.doEncode(&bs)
	return base64.StdEncoding.EncodeToString(bs)
}

//解密String
func (s *SimpleEncrypt) DecodeString(str string) string {
	bs, _ := base64.StdEncoding.DecodeString(str)
	s.doEncode(&bs)
	return string(bs)
}

//加密
func (s *SimpleEncrypt) Encode(data *[]byte) {
	s.doEncode(data)

}

//解密
func (s *SimpleEncrypt) Decode(data *[]byte) {
	s.doEncode(data)
}

func (s *SimpleEncrypt) doEncode(bs *[]byte) {
	tmp := []byte(s.Key)
THEFOR:
	for i := 0; i < len(*bs); {
		for j := 0; j < len(tmp); j, i = j+1, i+1 {
			if i >= len(*bs) {
				break THEFOR
			}
			(*bs)[i] = (*bs)[i] ^ tmp[j]
		}
	}
}

func UUID(length int) string {
	tmp := make([]byte, length>>1)
	rand.Read(tmp)
	return hex.EncodeToString(tmp)
}
