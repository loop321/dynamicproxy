# dynamicproxy
动态代理，类似花生壳功能，实现内网应用发布到外网。配合Nginx，实现多域名转发到内网
# 使用方法
1.server 端，编译部署在外网服务器
修改配置文件 server/config.json
{
	"version":"0.1", 
	"managerPort":"8086", #web页面管理，暂未开发
	"connPort":"11179",#客户端连接端口，固定
	"nginxDir":"D:/soft/nginx-1.10.1", #nginx目录，默认sbin在此目录内，否则不能使用
	"userSetting":{  #用户配置
		"loop321,pwdpwd":[]  #loop321用户，密码为pwdpwd,在服务端代理配置为空	
	}
}

2.client 端,编译部署在内网
修改 client/config.json
{
	"addr":"127.0.0.1:11179", #连接服务端
	"user":"loop321", #用户
	"pwd":"pwdpwd",   #密码
	"inner":"127.0.0.1:8080", #代理内网地址（目前支持http转发） 多个用|分开
	"outer":"www.test.cn"#外网访问域名，与inner一一对应， 多个用|分开
}
3.nginx 
修改nginx.conf
http{
...
map $http_host $pp {
    include hp.conf;
}
server {
	listen 80;
	server_name $http_host;
	location / {
		proxy_pass http://127.0.0.1:$pp;
	}
}
...
}
增加空hp.conf文件
