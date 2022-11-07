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
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
)

type FeishuRobot struct {
	*template.Template
	Client *http.Client
}

func NewFeishuRobot(tmpl *template.Template) FeishuRobot {
	return FeishuRobot{
		Template: tmpl,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

type FeishuResp struct {
	// success
	StatusCode    int    `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`

	// failed
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (resp FeishuResp) String() string {
	bts, _ := json.Marshal(resp)
	return string(bts)
}

// copy from feishu
func genFeishuSign(secret string, timestamp int64) (string, error) {
	//timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

func (f FeishuRobot) DoRequest(params url.Values, alert Alert) error {
	obj := struct {
		Alert
		// for template
		At        []string
		Timestamp int64
		Sign      string
	}{
		Alert: alert,
	}

	if params.Get("at") != "" {
		obj.At = strings.Split(params.Get("at"), ",")
	}
	if params.Get("signSecret") != "" {
		obj.Timestamp = time.Now().Unix()
		sign, err := genFeishuSign(params.Get("signSecret"), obj.Timestamp)
		if err != nil {
			return errors.Wrap(err, "gen sign")
		}
		obj.Sign = sign
	}

	buf := bytes.NewBuffer([]byte{})
	if err := f.Template.Execute(buf, obj); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, params.Get("url"), buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if err := retry.Do(func() error {
		resp, err := f.Client.Do(req)
		if err != nil {
			log.Println(errors.Wrap(err, "do request"))
			return nil
		}

		shouldRetry, err := checkFeishuResponse(resp)
		if shouldRetry {
			return errors.Wrap(err, "check response")
		}
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("send alert to: %s, msg: %s", params.Get("url"), alert.Annotations["message"])
		}
		return nil
	}, retry.Attempts(5), retry.Delay(5*time.Second)); err != nil {
		log.Println(err)
	}
	return nil
}

// {"StatusCode":0,"StatusMessage":"","code":9499,"msg":"too many request"}
// {"StatusCode":0,"StatusMessage":"","code":19007,"msg":"Bot Not Enabled"}
// {"StatusCode":0,"StatusMessage":"","code":19021,"msg":"sign match fail or timestamp is not within one hour from current time"}
func checkFeishuResponse(resp *http.Response) (shouldRetry bool, err error) {
	obj := FeishuResp{}
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(bts, &obj); err != nil {
		return false, errors.Wrapf(err, "unmarshal feishu resp, body: %s", string(bts))
	}
	if obj.StatusCode != 0 || obj.Code != 0 {
		if obj.Code == 9499 { // too many request
			return true, errors.New(obj.String())
		} else {
			return false, errors.New(obj.String())
		}
	}
	return false, nil
}
