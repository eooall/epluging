package main

import (
	e_rabbit "eplugs/e-rabbit"
	"log"
	"time"
)

func main() {
	//建立rabbit连接并且监听rabbit连接
	cfg := e_rabbit.RabbitMQCfg{
		Url:       "amqp://admin:admin@127.0.0.1:5672/",
		Heartbeat: time.Second * 5, //五秒一次心跳检查
		RetryNum:  10,
		RabbitBlockedLogFun: func(info e_rabbit.RabbitBlockedLog) {
			log.Println("阻塞 --->  ", info) //elk等等
		},
		RabbitCloseLogFun: func(info e_rabbit.RabbitCloseLog) {
			log.Println("关闭 --->   ", info) //elk等等
		},
		RabbitCloseActiveFun: func() {
			log.Println("重连到达上限，不在重连 ---> 发送邮件")
		},
		RabbitRetryInfoFun: func(info e_rabbit.RabbitRetryInfo) {
			log.Println("重连信息 --->   ", info) //elk等等
		},
	}

	_, err := e_rabbit.NewRabbitMQ(cfg)
	if err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Second * 5)
	}
}
