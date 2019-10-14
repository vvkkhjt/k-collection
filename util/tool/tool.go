package tool

import (
	"github.com/rifflock/lfshook"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/watch"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	Log *logrus.Logger
	KafkaWriter *kafka.Writer
)

func init() {
	//kafkaURL := "kafka-broker-0.ppgame.com:31900,kafka-broker-1.ppgame.com:31901,kafka-broker-2.ppgame.com:31902"
	//topic := "kapp"
	//
	//KafkaWriter = newKafkaWriter(kafkaURL, topic)
	Log = newLogger()
}

// 获取本地k8s配置文件路径
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// reg
func RegCluster(data string, siteUrl string) bool {
	Log.Info("正在注册数据...")
	resp, err := http.PostForm(siteUrl, url.Values{"data": {data}})
	if err != nil {
		Log.Error("链接地址失败:" + err.Error())
		return false
	} else {
		defer func() {
			err := resp.Body.Close()
			if err != nil {
				Log.Error(err.Error())
			}
		}()

		if resp.StatusCode == 200 {
			Log.Info("数据注册完成...")
			return true
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Log.Error("读取数据失败:" + err.Error())
			} else {
				Log.Warn("数据注册失败:", string(body))
			}
			return false
		}
	}
}

// 发送数据
func HttpPostForm(data, siteUrl string, wtype watch.EventType) {
	//Log.Info("正在发送数据...")
	//
	//msg := kafka.Message{
	//	Key:   []byte(wtype),
	//	Value: []byte(data),
	//}
	//err := KafkaWriter.WriteMessages(context.Background(), msg)
	//if err != nil {
	//	log.Fatalln(err.Error())
	//}
	//
	//Log.Info("发送成功")

	resp, err := http.PostForm(siteUrl, url.Values{"data": {data}})
	if err != nil {
		Log.Error("链接地址失败:" + err.Error())
	} else {
		defer func() {
			err := resp.Body.Close()
			if err != nil {
				Log.Error(err.Error())
			}
		}()

		if resp.StatusCode == 200 {
			Log.Info("数据发送完成...")
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Log.Error("读取数据失败:" + err.Error())
			} else {
				Log.Warn("数据发送失败:", string(body))
			}
		}
	}
}

func newLogger() *logrus.Logger {
	if Log != nil {
		return Log
	}
	pathMap := lfshook.PathMap{
		logrus.InfoLevel:  "../log/info.log",
		logrus.ErrorLevel: "../log/error.log",
	}
	Log = logrus.New()
	Log.Hooks.Add(lfshook.NewHook(
		pathMap,
		&logrus.JSONFormatter{},
	))
	return Log
}

func newKafkaWriter(kafkaURL, topic string) *kafka.Writer {
	brokers := strings.Split(kafkaURL, ",")
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
}
