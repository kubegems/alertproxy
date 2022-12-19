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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
)

type DingdingRobot struct {
	*template.Template
	Client *http.Client
}

func NewDingdingRobot(tmpl *template.Template) DingdingRobot {
	return DingdingRobot{
		Template: tmpl,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func genDingdingSign(secret string, timestamp int64) (string, error) {
	//timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	_, err := h.Write([]byte(stringToSign))
	if err != nil {
		return "", err
	}
	signature := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	return signature, nil
}

func (f DingdingRobot) DoRequest(params url.Values, alert Alert) error {
	obj := struct {
		Alert
		// for template
		AtMobiles []string
	}{
		Alert: alert,
	}

	u, err := url.Parse(params.Get("url"))
	if err != nil {
		return err
	}
	query := u.Query()

	if params.Get("atMobiles") != "" {
		obj.AtMobiles = strings.Split(params.Get("atMobiles"), ",")
	}
	if params.Get("signSecret") != "" {
		t := time.Now() // 钉钉要求毫秒
		sign, err := genDingdingSign(params.Get("signSecret"), t.UnixMilli())
		if err != nil {
			return errors.Wrap(err, "gen sign")
		}
		query.Add("sign", sign)
		query.Add("timestamp", fmt.Sprintf("%v", t.UnixMilli()))
	}
	u.RawQuery = query.Encode()

	buf := bytes.NewBuffer([]byte{})
	if err := f.Template.Execute(buf, obj); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), buf)
	if err != nil {
		return err
	}
	req.URL.Query()
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.Client.Do(req)
	if err != nil {
		return errors.Wrap(err, "do request")
	}
	b := DingdingResp{}
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return errors.Wrap(err, "decode dingding resp")
	}
	if b.Errcode != 0 {
		return errors.Errorf("send to dingding robot failed, code: %d, msg: %s", b.Errcode, b.Errmsg)
	}
	log.Printf("send alert to: %s, msg: %s", params.Get("url"), alert.Annotations["message"])
	return nil
}

type DingdingResp struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}
