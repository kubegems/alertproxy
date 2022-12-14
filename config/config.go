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

package config

type ProxyConfigs struct {
	Listen    string           `json:"listen"`
	Templates []*ProxyTemplate `json:"proxyTemplates"`
}

type ProxyType string

const (
	Feishu      ProxyType = "feishu"
	DingDing    ProxyType = "dingding"
	AliyunMsg   ProxyType = "aliyunMsg"
	AliyunVoice ProxyType = "aliyunVoice"
)

type ProxyTemplate struct {
	Type     ProxyType `json:"type"`
	Template string    `json:"template"`
}
