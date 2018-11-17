package run

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"github.com/gin-gonic/gin"
)

const ProtocolMQTT = "mqtt"

const ProtocolWebsocket = "websocket"

//Default configration
const (
	DefaultDeliveryRetryInterval = 20
	DefaultQueueQos0Messages     = true
	DefaultMaxInflightMessages   = 20
	DefaultLogging               = false
	DefaultMaxMsgQueueMessages   = 2048
	DefaultHttpAddr              = ":8080"
)

//监听地址,类型：tcp/ssl ws/wss
type Config struct {
	DeliveryRetryInterval int64            `yaml:"delivery_retry_interval"`
	QueueQos0Messages     bool             `yaml:"queue_qos0_messages"`
	MaxInflightMessages   int              `yaml:"max_inflight_messages"`
	MaxMsgQueueMessages   int              `yaml:"max_msgqueue_messages"`
	ProfileConfig         ProfileConfig    `yaml:"profile"`
	Listener              []ListenerConfig `yaml:"listener,flow"`
	Logging               bool             `yaml:"logging"`
	HttpServerConfig              HttpServerConfig           `yaml:"http_server"`
}

type ProfileConfig struct {
	CPUProfile string `yaml:"cpu"`
	MemProfile string `yaml:"mem"`
}

type HttpServerConfig struct {
	Addr string  `yaml:"addr"`
	User gin.Accounts `yaml:"user"`
}

type ListenerConfig struct {
	Protocol string `yaml:"protocol"`
	Addr     string `yaml:"addr"`
	CertFile string `yaml:"certfile"`
	KeyFile  string `yaml:"keyfile"`
}

func (c *Config) Validate() error {
	for _, v := range c.Listener {
		if v.Protocol != ProtocolMQTT && v.Protocol != ProtocolWebsocket {
			return fmt.Errorf("invalid protocol name '%s',expect 'mqtt' or 'websocket'", v.Protocol)
		}
		if v.KeyFile != "" && v.CertFile == "" {
			return fmt.Errorf("invalid tls/ssl configration, 'certfile missing'")
		}
		if v.KeyFile == "" && v.CertFile != "" {
			return fmt.Errorf("invalid tls/ssl configration, 'keyfile' missing")
		}
		if v.Addr == "" {
			return fmt.Errorf("addr missing")
		}
	}
	return nil
}

func NewConfig() *Config {
	return &Config{
		DeliveryRetryInterval: DefaultDeliveryRetryInterval,
		QueueQos0Messages:     DefaultQueueQos0Messages,
		MaxInflightMessages:   DefaultMaxInflightMessages,
		MaxMsgQueueMessages:   DefaultMaxMsgQueueMessages,
		Logging:               DefaultLogging,
	}
}

// loads the config from a yaml config file
func (c *Config) FromConfigFile(fpath string) error {
	bs, err := ioutil.ReadFile(fpath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bs, c)
	if err != nil {
		return err
	}
	if len(c.Listener) == 0 {
		c.Listener = make([]ListenerConfig, 1)
		c.Listener[0].Protocol = ProtocolMQTT
		c.Listener[0].Addr = ":1883"
	}
	return nil
}
