package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"util"

	"github.com/p/mahonia"
	"github.com/wuxicn/pipeline"
)

var (
	//WinPort   = `netstat -ano|grep LISTENING |awk '{match($2,'.*[:]([0-9]+)',a);print a[1]}'`
	WinPort   = `netstat -ano|grep LISTENING |awk '{print $2}'`
	LinuxPort = `netstat -lnt | awk '{if(NR>2) { match($4,".*[:]([0-9]+)",a) ;print a[1] }}'`
)

func GetPort() string {
	defer util.Catch()
	//fmt.Sprintf("%.16f", rand.Float64())[4:8]
	PortMap := map[string]bool{}
	if strings.ToLower(runtime.GOOS) == "windows" {
		_, res := execWin(exec.Command("cmd.exe", "/c", WinPort))
		ps := strings.Split(res, "\n")
		for _, line := range ps {
			pos := strings.LastIndex(line, ":")
			if pos > 1 {
				PortMap[line[+1:]] = true
			}
		}
	} else {
		res := execLinux(LinuxPort)
		ps := strings.Split(res, "\n")
		for _, line := range ps {
			line = strings.TrimSpace(line)
			if line != "" {
				PortMap[line] = true
			}
		}
	}
	port := ""
	for i := 0; i < 30; i++ {
		port = Rand()
		if PortMap[port] {
			continue
		} else {
			break
		}
	}
	return port
}

func Rand() string {
	return "1" + fmt.Sprintf("%.16f", rand.Float64())[4:8]
}

//执行管道命令
func execWin(pipe ...*exec.Cmd) (errstr, res string) {
	defer util.Catch()
	stdout, serr, err := pipeline.Run(pipe...)
	util.CheckError(err)
	//log.Println("pipecmd-err-log:", err, string(mahonia.NewDecoder("GBK").ConvertString(serr.String())))
	return string(mahonia.NewDecoder("GBK").ConvertString(serr.String())), strings.TrimRight(string(mahonia.NewDecoder("GBK").ConvertString(stdout.String())), "\r\n")
}

func execLinux(c string) (str string) {
	defer util.Catch()
	cmd := exec.Command("/bin/bash", "-c", c)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		str = "StdoutPipe: " + err.Error()
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		str = "StderrPipe: " + err.Error()
		return
	}
	if err := cmd.Start(); err != nil {
		str = "Start: " + err.Error()
		return
	}
	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		str = "ReadAll stderr: " + err.Error()
		return
	}
	if len(bytesErr) != 0 {
		str = fmt.Sprintf("stderr is not nil: %s", bytesErr)
		return
	}
	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		str = "ReadAll stdout: " + err.Error()
		return
	}
	if err := cmd.Wait(); err != nil {
		str = "Wait: " + err.Error()
		return
	}
	tmpStr := string(bytes)
	str = strings.Trim(tmpStr, "\n")
	return
}

//写配置
func WriteToFile(set map[string]string, filepath string) bool {
	defer util.Catch()
	log.Println(set)
	f, _ := os.OpenFile(filepath+"/conf/hp.conf", os.O_RDWR|os.O_TRUNC|os.O_SYNC, 777)
	defer f.Close()
	bw := bufio.NewWriter(f)
	resStr := ""
	for domain, port := range set {
		if domain != "" && port != "" {
			resStr += fmt.Sprintf("%s %s;\n", domain, port)
		}
	}
	if resStr == "" {
		return true
	}
	_, e := bw.WriteString(resStr)
	log.Println("---------------", resStr)
	bw.Flush()
	if e == nil {
		return true
	}
	return false
}

//读取配置
func ReadFromFile(filepath string) map[string]string {
	setMap := map[string]string{}
	defer util.Catch()
	f, err := os.OpenFile(filepath+"/conf/hp.conf", os.O_RDWR, os.ModeType)
	defer f.Close()
	if err != nil {
		log.Println(err)
	}
	bn := bufio.NewReader(f)
	var str []byte
	for str, _, err = bn.ReadLine(); err != io.EOF; str, _, err = bn.ReadLine() {
		res := strings.Split(strings.TrimRight(string(str), ";"), " ")
		if len(res) == 2 {
			setMap[res[0]] = res[1]
		}
	}
	return setMap
}

func NginxReload(path string) (b bool) {
	defer util.Catch()
	if strings.ToLower(runtime.GOOS) == "windows" {
		res, _ := execWin(exec.Command("cmd.exe", "/c", "cd /d "+path+" && nginx.exe -t"))
		if strings.Index(res, "successful") > 0 {
			log.Println(execWin(exec.Command("cmd.exe", "/c", "cd /d "+path+" && nginx.exe -s reload")))
			b = true
		}
	} else {
		if strings.Index(execLinux(path+"/sbin/nginx -t"), "successful") > 0 {
			log.Println(execLinux(path + "/sbin/nginx -s reload"))
			b = true
		}
	}
	if !b {
		log.Println("语法出错了...")
	}
	return
}

//
func AddToNgin(domain, port string) {
	defer util.Catch()
	path := util.Sysconfig["nginxDir"].(string)
	m := ReadFromFile(path)
	m[domain] = port
	log.Println(m)
	if WriteToFile(m, path) {
		log.Println("重新加载nginx:", NginxReload(path))
	}
}

//
func DelFromNgin(domain string) {
	defer util.Catch()
	path := util.Sysconfig["nginxDir"].(string)
	m := ReadFromFile(path)
	if m[domain] != "" {
		delete(m, domain)
		log.Println(m)
		if WriteToFile(m, path) {
			log.Println("重新加载nginx:", NginxReload(path))
		}
	}
}
