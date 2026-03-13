package provider

import (
	"context"
	"encoding/json"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
)

type SmsProvider interface {
	Send(ctx context.Context, phone string, signName string, templateCode string, params map[string]string) error
}

type AliyunSmsProvider struct {
	client *dysmsapi.Client
}

func NewAliyunSmsProvider(accessKeyId, accessKeySecret, endpoint string) SmsProvider {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}
	client, err := dysmsapi.NewClient(config)
	if err != nil {
		panic(fmt.Errorf("创建阿里云 SMS 客户端失败: %w", err))
	}
	return &AliyunSmsProvider{client: client}
}

func (p *AliyunSmsProvider) Send(ctx context.Context, phone string, signName string, templateCode string, params map[string]string) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("序列化短信参数失败: %w", err)
	}
	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(string(paramsJSON)),
	}
	_, err = p.client.SendSms(req)
	if err != nil {
		return fmt.Errorf("发送短信失败: %w", err)
	}
	return nil
}
