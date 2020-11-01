package e_rabbit

import (
	"errors"
	"github.com/streadway/amqp"
	"time"
)

//rabbit交换机相关操作

//交换机创建不成功应该让这个项目启动不成功
type ExchangeDeclareCfg struct {
	Name                string                   //交换器名称
	Kind                string                   //交换器类型
	Durable             bool                     //是否持久化
	AutoDelete          bool                     //是否自动删除
	Internal            bool                     //是否为内部使用 --->  内部使用则只有交换器可以发送msg到内部交换器
	NoWait              bool                     //是否非阻塞
	Args                amqp.Table               //相关参数设置，详见源码
	ReTryNum            int                      //创建交换机重试次数
	RetryPause          time.Duration            //重试间隔时间 --->  默认间隔一秒
	ExchangeDeclareFunc func(ExchangeDeclareLog) //重试信息
	RabbitUrl           string                   //rabbit地址
}
type ExchangeDeclareLog struct {
	Time            time.Time          //创建交换机重试发生时间
	MateInformation ExchangeDeclareCfg //创建交换机重试元信息
	Err             error              //创建交换机重试err
}

//创建交换机
func (r *rabbitMQ) ExchangeDeclare(cfg ExchangeDeclareCfg) error {
	err := checkExchangeDeclareCfg(&cfg)
	if err != nil {
		return err
	}
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	for i := 0; i < cfg.ReTryNum; i++ {
		//IsClosed 为 true 则表示该rabbit被关闭
		if !r.conn.IsClosed() {
			err = ch.ExchangeDeclare(cfg.Name, cfg.Kind, cfg.Durable, cfg.AutoDelete, cfg.Internal, cfg.NoWait, cfg.Args)
			if err != nil {
				exchangeDeclareLog := ExchangeDeclareLog{}
				exchangeDeclareLog.Time = time.Now()
				exchangeDeclareLog.MateInformation = cfg
				exchangeDeclareLog.MateInformation.RabbitUrl = r.userCfg.Url
				exchangeDeclareLog.Err = err
				cfg.ExchangeDeclareFunc(exchangeDeclareLog)
			}
		}
		time.Sleep(cfg.RetryPause)
	}
	return err
}

//初始化一些并且校验创建交换机数据
func checkExchangeDeclareCfg(cfg *ExchangeDeclareCfg) error {
	if cfg.Name == "" {
		return errors.New("ExchangeDeclare not null")
	}
	switch cfg.Kind {
	case "direct","topic","fanout","headers":
	default:
		return errors.New("ExchangeDeclare kind error")
	}
	if cfg.ReTryNum == 0 {
		cfg.ReTryNum = 5
	}
	if cfg.RetryPause == 0 {
		cfg.RetryPause = time.Second * 1
	}
	if cfg.ExchangeDeclareFunc == nil {
		cfg.ExchangeDeclareFunc = func(ExchangeDeclareLog) {}
	}
	return nil
}