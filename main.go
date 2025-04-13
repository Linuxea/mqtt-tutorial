package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTConfig 存储MQTT配置
type MQTTConfig struct {
	Broker   string
	ClientID string
	Topic    string
	QoS      byte
}

// 消息处理函数，当从订阅的主题收到消息时会被调用
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message from topic: %s\n", msg.Topic())
	log.Printf("Message: %s\n", string(msg.Payload()))
}

// 连接建立成功时的处理函数
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

// 连接丢失时的处理函数
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v\n", err)
}

func main() {
	// 初始化日志
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// MQTT配置
	config := MQTTConfig{
		Broker:   "tcp://127.0.0.1:1883",
		ClientID: "simple-go-mqtt-client",
		Topic:    "test/topic",
		QoS:      1,
	}

	// 创建客户端配置
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.SetDefaultPublishHandler(messagePubHandler)

	// 创建客户端并连接
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	// 订阅主题
	token := client.Subscribe(config.Topic, config.QoS, nil)
	if token.Wait() && token.Error() != nil {
		log.Printf("Failed to subscribe: %v", token.Error())
		client.Disconnect(250)
		return
	}
	log.Printf("Subscribed to topic: %s\n", config.Topic)

	// 创建上下文用于优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在后台发送消息
	go func() {
		for i := 1; i <= 10; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				text := fmt.Sprintf("Hello MQTT from Go! Message %d", i)
				token = client.Publish(config.Topic, config.QoS, false, text)
				if token.Wait() && token.Error() != nil {
					log.Printf("Failed to publish message %d: %v", i, token.Error())
					continue
				}
				log.Printf("Published message: %s\n", text)
				time.Sleep(time.Second)
			}
		}
	}()

	// 等待中断信号
	<-sigChan
	log.Println("Shutting down...")

	// 清理订阅
	if token := client.Unsubscribe(config.Topic); token.Wait() && token.Error() != nil {
		log.Printf("Failed to unsubscribe: %v", token.Error())
	}

	// 断开连接
	client.Disconnect(250)
	log.Println("Disconnected from MQTT broker")
}
