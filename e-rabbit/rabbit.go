package e_rabbit

import (
	"github.com/streadway/amqp"
	"sync"
	"time"
)

// rabbit配置
type RabbitMQCfg struct {
	Url                  string
	RetryNum             int                    //rabbit连接出现问题重试次数 -1 则不限次数
	RetryPause           time.Duration          //rabbit连接出现问题重试连接间隔时间
	Heartbeat            time.Duration          //rabbit心跳时间	--->	此心跳时间仅仅为通知服务端不要断开连接，因为rabbit长时间不传递数据，服务端会断开与客户端的连接
	RabbitBlockedLogFun  func(RabbitBlockedLog) //rabbit阻塞log，具体操作由用户配置，举例：写入文件，发送email等等
	RabbitCloseLogFun    func(RabbitCloseLog)   //rabbit关闭log，具体操作由用户决定
	RabbitRetryInfoFun   func(RabbitRetryInfo)  //rabbit在重连过程中产生信息
	RabbitCloseActiveFun func()                 //如果rabbit重连次数到达预设次数，则执行函数
}

// rabbit连接发生阻塞
type RabbitBlockedLog struct {
	Time            time.Time //发生阻塞不可用的时间
	Detail          string    //发生阻塞详细信息
	MateInformation string    //发生阻塞的rabbit元信息
}

// rabbit被关闭
type RabbitCloseLog struct {
	Time            time.Time //关闭时间
	Code            int       //rabbit响应码
	Server          bool      //是否从远端启动 ---> 远端启动为true，本地启动为false
	Reason          string    //关闭的详细信息
	Recover         bool      //是否可以通过重试或者改变参数重新启动 ---> 是为true 否为false
	MateInformation string    //rabbit被关闭发生的rabbit元信息
}

// rabbit重连发生错误
type RabbitRetryInfo struct {
	Time            time.Time //重连时间
	Num             int       //重连次数
	Err             error     //重连错误
	RetryServer     bool      //重连成功则为 true
	MateInformation string    //重连的rabbit元信息
}

type rabbitMQ struct {
	conn    *amqp.Connection //rabbit连接
	//channel *amqp.Channel    //rabbit信道
	connMux *sync.Mutex      //连接锁
	userCfg RabbitMQCfg      //rabbit相关配置
}

//暴露
func NewRabbitMQ(cfg RabbitMQCfg) (*rabbitMQ, error) {
	var err error
	conn := &rabbitMQ{}
	conn.conn, err = amqp.DialConfig(cfg.Url, initRabbitMQCfg(&cfg))
	if err == nil {
		go conn.reConnection()
		conn.userCfg = cfg
	}
	return conn, err
}

// 由用户选择配置生成amqp配置与用户配置
func initRabbitMQCfg(userCfg *RabbitMQCfg) amqp.Config {
	amqpCfg := amqp.Config{}
	//设置默认心跳
	if userCfg.Heartbeat == 0 {
		amqpCfg.Heartbeat = time.Second * 5
	}
	//设置默认链接时间
	if userCfg.RetryPause == 0 {
		userCfg.RetryPause = time.Second * 1
	}
	//默认无限制重试次数
	if userCfg.RetryNum == 0 {
		userCfg.RetryNum = -1
	}
	//rabbit重连达到上线默认函数
	if userCfg.RabbitCloseActiveFun == nil {
		userCfg.RabbitCloseActiveFun = func() {}
	}
	//rabbit关闭log默认函数
	if userCfg.RabbitCloseLogFun == nil {
		userCfg.RabbitCloseLogFun = func(RabbitCloseLog) {}
	}
	//rabbit阻塞默认log函数
	if userCfg.RabbitBlockedLogFun == nil {
		userCfg.RabbitBlockedLogFun = func(RabbitBlockedLog) {}
	}
	//rabbit重连产生错误默认函数
	if userCfg.RabbitRetryInfoFun == nil {
		userCfg.RabbitRetryInfoFun = func(RabbitRetryInfo) {}
	}
	return amqpCfg
}

// 监听rabbit是否可用，断线
// rabbit为阻塞则需要关闭
// rabbit关闭则直接放弃
func (r *rabbitMQ) reConnection() {
	for {
		select {
		//读出数据则表示当前rabbit连接出某些原因正在阻塞中，直接放弃该连接
		case err := <-r.conn.NotifyBlocked(make(chan amqp.Blocking, 0)):
			//err.Active 为true 则表示该rabbit为阻塞状态 rabbit连接不可用
			//err.Reason 为rabbit阻塞具体信息	--->	只有Active为true时才会存在具体阻塞信息
			//如果为true，应记录具体阻塞信息，时间等数据
			r.conn.Close()
			if err.Active {
				blockedInfo := RabbitBlockedLog{
					Time:            time.Now(),
					Detail:          err.Reason,
					MateInformation: r.userCfg.Url,
				}
				r.userCfg.RabbitBlockedLogFun(blockedInfo)
			}
		//读出数据则表示当前被关闭，具体原因解析 err
		case err := <-r.conn.NotifyClose(make(chan *amqp.Error, 0)):
			if err != nil {
				closeInfo := RabbitCloseLog{
					Time:            time.Now(),
					Code:            err.Code,
					Server:          err.Server,
					Reason:          err.Reason,
					Recover:         err.Recover,
					MateInformation: r.userCfg.Url,
				}
				//执行rabbit关闭相关相关log函数
				r.userCfg.RabbitCloseLogFun(closeInfo)
			}
		}
		// 开始重连
		var (
			reNum = 0
			err   error
			conn  = &rabbitMQ{}
		)
	beginRetry:
		if reNum == r.userCfg.RetryNum {
			goto activeRetryFunc
		}
		conn, err = NewRabbitMQ(r.userCfg)
		if err != nil {
			//执行rabbit重连相关相关log函数
			r.userCfg.RabbitRetryInfoFun(RabbitRetryInfo{Time: time.Now(), Num: reNum + 1, Err: err, MateInformation: r.userCfg.Url})
			reNum++
			//重连间隔时间
			time.Sleep(r.userCfg.RetryPause)
			goto beginRetry
		}
		//重连成功
		r.conn = conn.conn
		goto endRetry
		//重连次数到达上线执行函数
	activeRetryFunc:
		//执行rabbit重连到达上限关闭rabbit函数
		r.userCfg.RabbitCloseActiveFun()
		return
		//重连结束
	endRetry:
		//执行rabbit链接成功相关log函数
		r.userCfg.RabbitRetryInfoFun(RabbitRetryInfo{Time: time.Now(), Num: reNum + 1, Err: nil, RetryServer: true, MateInformation: r.userCfg.Url})
	}
}
