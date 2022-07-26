/*Package redis init redis connection
 */
package redis

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/sirupsen/logrus"
)

// Interface operate golang struct
type Interface interface {
	SetObj(key string, obj interface{}, expiration time.Duration) error
	GetObj(key string, obj interface{}) error
	Cli() *redis.Client
}

type client struct {
	Client *redis.Client
}

// var client *redis.Client
var c client

// GetClient return redis client
func GetRedis() Interface {
	return &c
}

// RedisConnect init redis connection
func RedisConnect(addr, pass string, dbnum int) {
	logrus.Infof("start connect redis addr=[%s], password=[%s], dbNumber=[%d]",
		addr, pass, dbnum)

	doConnect := func() error {
		newClient := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pass,
			DB:       dbnum,
		})

		_, err := newClient.Ping().Result()
		if err != nil {
			return err
		}
		c.Client = newClient
		return nil
	}
	go func() {
		var count int
	connDB:
		if count > 10 {
			panic("can not connect redis, panic")
		}
		if err := doConnect(); err != nil {
			logrus.Errorf("can't connect redis addr=[%s], password=[%s], dbNumber=[%d], retry",
				addr, pass, dbnum)
			time.Sleep(time.Second * 1)
			count++
			goto connDB
		}
		logrus.Infof("Redis Connection established")
	}()
}

func (c *client) SetObj(key string, obj interface{}, expiration time.Duration) error {
	b, err := json.Marshal(&obj)
	if err != nil {
		return err
	}
	v, err := c.Client.Set(key, string(b), expiration).Result()
	if err != nil {
		return err
	}
	logrus.Debugf("redis save success %s: ", v)
	return nil
}

func (c *client) GetObj(key string, obj interface{}) error {
	v, err := c.Client.Get(key).Result()
	if err != nil {
		return err
	}
	logrus.Debugf("get value from redis %s", v)
	err = json.Unmarshal([]byte(v), obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) Cli() *redis.Client {
	return c.Client
}
