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

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dyvmsapi"
	"github.com/pkg/errors"
)

type AliyunVoice struct {
	*template.Template
}

func NewAliyunVoice(tmpl *template.Template) AliyunVoice {
	return AliyunVoice{
		Template: tmpl,
	}
}

func (f AliyunVoice) DoRequest(params url.Values, alert Alert) error {
	client, err := dyvmsapi.NewClientWithAccessKey("cn-hangzhou", params.Get("accessKeyId"), params.Get("accessKeySecret"))
	if err != nil {
		return err
	}
	request := dyvmsapi.CreateSingleCallByTtsRequest()
	request.CalledNumber = params.Get("callNumber")
	request.TtsCode = params.Get("ttsCode")

	obj := struct {
		Alert
	}{
		Alert: alert,
	}
	buf := bytes.NewBuffer([]byte{})
	if err := f.Template.Execute(buf, obj); err != nil {
		return err
	}
	request.TtsParam = buf.String()

	resp, err := client.SingleCallByTts(request)
	if err != nil {
		return errors.Wrap(err, "SingleCallByTts")
	}

	log.Printf("send alert to phone number: %s by phone call, msg: %s, resp:\n%+v", request.CalledNumber, alert.Annotations["message"], resp)
	return nil
}
