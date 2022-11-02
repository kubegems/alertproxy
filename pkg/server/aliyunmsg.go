/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"bytes"
	"log"
	"net/url"
	"text/template"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
	"github.com/pkg/errors"
)

type AliyunMsg struct {
	*template.Template
}

func NewAliyunMsg(tmpl *template.Template) AliyunMsg {
	return AliyunMsg{
		Template: tmpl,
	}
}

func (f AliyunMsg) DoRequest(params url.Values, alert Alert) error {
	client, err := dysmsapi.NewClientWithAccessKey("cn-hangzhou", params.Get("accessKeyId"), params.Get("accessKeySecret"))
	if err != nil {
		return err
	}
	request := dysmsapi.CreateSendSmsRequest()
	request.Scheme = "https"
	request.PhoneNumbers = params.Get("phoneNumbers")
	request.SignName = params.Get("signName")
	request.TemplateCode = params.Get("templateCode")

	obj := struct {
		Alert
	}{
		Alert: alert,
	}
	buf := bytes.NewBuffer([]byte{})
	if err := f.Template.Execute(buf, obj); err != nil {
		return err
	}
	request.TemplateParam = buf.String()

	_, err = client.SendSms(request)
	if err != nil {
		return errors.Wrap(err, "send sms")
	}
	log.Printf("send alert to phoneNumbers: %s, msg: %s", request.PhoneNumbers, alert.Annotations["message"])
	return nil
}
