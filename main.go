package main

import (
	"errors"
	"github.com/hongfs/ecs-metadata/pkg/metadata"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v3/client"
	"github.com/alibabacloud-go/tea/tea"
)

var AccessKeyId = ""
var AccessKeySecret = ""
var SecurityToken = ""
var Region = ""
var Tags = make(map[string]string)
var Script = ""
var Client *ecs20140526.Client

// init 初始化
func init() {
	err := loadCredentials()

	if err != nil {
		panic("load credentials error: " + err.Error())
	}

	if value := os.Getenv("ALIYUN_REGION"); value != "" {
		Region = value
	} else {
		panic("REGIONS is empty")
	}

	if value := os.Getenv("ALIYUN_TAGS"); value != "" {
		values := strings.Split(value, ";")

		for _, v := range values {
			if strings.Contains(v, "=") {
				kv := strings.Split(v, "=")
				Tags[kv[0]] = kv[1]
			} else {
				panic("TAGS format error")
			}
		}
	}

	if value := os.Getenv("ALIYUN_SCRIPT"); value != "" {
		Script = value
	} else {
		panic("SCRIPT is empty")
	}

	client, err := getClient()

	if err != nil {
		panic("get client error: " + err.Error())
	}

	Client = client
}

// main 主函数
func main() {
	err := handle()

	if err != nil {
		panic("handle error: " + err.Error())
	}

	os.Exit(0)
}

// handle 处理
func handle() error {
	ids, err := getInstances()

	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return errors.New("no instance")
	}

	wg := new(sync.WaitGroup)

	for _, group := range splitSlice(ids, 50) {
		wg.Add(1)

		go func(group []string) {
			defer wg.Done()

			err = runCommand(group)

			if err != nil {
				log.Println("run command error: ", err.Error())
			}
		}(group)
	}

	wg.Wait()

	return nil
}

// runCommand 执行命令
func runCommand(ids []string) error {
	result, err := Client.RunCommand(&ecs20140526.RunCommandRequest{
		RegionId:        tea.String(Region),
		InstanceId:      tea.StringSlice(ids),
		Type:            tea.String("RunShellScript"), // 只能 Linux 脚本
		CommandContent:  tea.String(Script),
		Timeout:         tea.Int64(60 * 10), // 10分钟超时
		RepeatMode:      tea.String("Once"), // 仅执行一次
		KeepCommand:     tea.Bool(false),
		ContentEncoding: tea.String("PlainText"),
	})

	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	id := *result.Body.InvokeId

	for {
		// 获取执行结果
		result, err := Client.DescribeInvocationResults(&ecs20140526.DescribeInvocationResultsRequest{
			RegionId: tea.String(Region),
			InvokeId: tea.String(id),
		})

		if err != nil {
			return err
		}

		// 没有记录
		if len(result.Body.Invocation.InvocationResults.InvocationResult) == 0 {
			time.Sleep(time.Second * 5)
			continue
		}

		status := *result.Body.Invocation.InvocationResults.InvocationResult[0].InvokeRecordStatus

		switch status {
		case "Running":
			// 正在运行中
			time.Sleep(time.Second * 5)
			continue
		case "Finished":
			log.Println("已完成：", id)
		case "Failed":
			log.Println("执行失败：", id)
		case "PartialFailed":
			log.Println("部分执行失败：", id)
		case "Stopped":
			log.Println("命令执行已停止：", id)
		case "Stopping":
			log.Println("正在停止执行的命令：", id)
		}

		break
	}

	return nil
}

// getInstances 获取实例列表
func getInstances() ([]string, error) {
	var page int32 = 1
	var size int32 = 100

	tags := make([]*ecs20140526.DescribeInstancesRequestTag, 0)

	for k, v := range Tags {
		tags = append(tags, &ecs20140526.DescribeInstancesRequestTag{
			Key:   tea.String(k),
			Value: tea.String(v),
		})
	}

	var ids = make([]string, 0)

	for {
		input := &ecs20140526.DescribeInstancesRequest{
			RegionId:   tea.String(Region),
			Status:     tea.String("Running"), // 必须是运行中的
			PageNumber: tea.Int32(page),
			PageSize:   tea.Int32(size),
		}

		if len(tags) != 0 {
			input.Tag = tags
		}

		result, err := Client.DescribeInstances(input)

		if err != nil {
			return nil, err
		}

		for _, v := range result.Body.Instances.Instance {
			ids = append(ids, *v.InstanceId)
		}

		if len(result.Body.Instances.Instance) < int(size) {
			break
		}

		page++
	}

	return ids, nil
}

// getClient 获取客户端
func getClient() (_result *ecs20140526.Client, _err error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(AccessKeyId),
		AccessKeySecret: tea.String(AccessKeySecret),
		RegionId:        tea.String(Region),
	}

	if SecurityToken != "" {
		config.SecurityToken = tea.String(SecurityToken)
	}

	return ecs20140526.NewClient(config)
}

// splitSlice 切分切片
func splitSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string

	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

func loadCredentials() error {
	if ramValue := os.Getenv("ALIYUN_RAM_NAME"); ramValue != "" {
		ram := metadata.Ram(ramValue)

		if ram.AccessKeyID != "" {
			AccessKeyId = ram.AccessKeyID
		}

		if ram.AccessKeySecret != "" {
			AccessKeySecret = ram.AccessKeySecret
		}

		if ram.SecurityToken != "" {
			SecurityToken = ram.SecurityToken
		}
	}

	if AccessKeyId == "" {
		value := os.Getenv("ALIYUN_ACCESS_KEY_ID")

		if value == "" {
			return errors.New("ACCESS_KEY_ID is empty")
		}

		AccessKeyId = value
	}

	if AccessKeySecret == "" {
		value := os.Getenv("ALIYUN_ACCESS_KEY_SECRET")

		if value == "" {
			return errors.New("ALIYUN_ACCESS_KEY_SECRET is empty")
		}

		AccessKeySecret = value
	}

	return nil
}
