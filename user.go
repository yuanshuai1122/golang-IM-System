package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	server *Server
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C: make(chan string),
		conn: conn,
		server: server,
	}

	// 启动监听user channel 的 goroutine
	go user.ListenMessage()

	return user
}

// 用户上线业务
func (this *User) Online() {
	// 用户上线，将用户加入到OnlineMap中
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()
	// 广播当前用户上线消息
	this.server.Broadcast(this, "已上线")
}

// 用户下线业务
func (this *User) Offline() {
	// 用户下线，将用户从OnlineMap移除
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()
	// 广播当前用户上线消息
	this.server.Broadcast(this, "下线")
}

// 给当前user对应的客户端发送消息
func (this *User) SendMessage(msg string) {
	this.conn.Write([]byte(msg))
}

// 用户处理消息业务
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户都有哪些
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap{
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线...\n"
			this.SendMessage(onlineMsg)

		}
		this.server.mapLock.Unlock()
	}else if len(msg) > 7 && msg[:7] == "rename|"{
		// 消息格式： rename|张三
		newName := strings.Split(msg, "|")[1]
		// 判断name是否存在
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMessage("当前用户名被使用")
		}else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()

			this.Name = newName
			this.SendMessage("您已更新用户名：" + this.Name + "\n")
		}
	}else if len(msg) > 4 && msg[:3] == "to|" {
		// 消息格式 to|张三|消息内容
		// 1.获取对方的用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.SendMessage("消息格式不正确，请使用格式为： \"to|张三｜消息内容\"\n")
			return
		}

		// 2.根据对方用户名，得到user对象
		remoteUser, ok := this.server.OnlineMap[remoteName]
		if !ok {
			this.SendMessage("该用户名不存在")
			return
		}

		// 3.获取消息内容，通过user对象，将消息发过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.SendMessage("无消息内容，清重发\n")
			return
		}
		remoteUser.SendMessage(this.Name + " 对您说： " + content)
	}else {
		this.server.Broadcast(this, msg)
	}
}


// 监听当前User channel的方法, 一旦有消息就发给客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C

		this.conn.Write([]byte(msg + "\n"))
	}
}
