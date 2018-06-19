package controllers

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type ExpressbirdController struct {
	BaseController
}

// shipperCode(物流公司代码),logisticCode(物流单号)
func (this *ExpressbirdController) ExpressInfoQuery() {
	shipperCode := this.GetString("shipperCode")
	logisticCode := this.GetString("logisticCode")

	if !CheckArg(shipperCode, logisticCode) {
		this.Rec = &Recv{5, "参数存在空值", nil}
	}

	client := &http.Client{}
	strval := ps(`{"OrderCode":"","ShipperCode":"%s","LogisticCode":"%s"}`, shipperCode, logisticCode)
	urlval := url.QueryEscape(strval)
	strKey := "690d0fe5-2f8c-4c67-900a-c72e3b587924"
	md5sign := StrToMD5(strval + strKey)
	base64sign := base64.StdEncoding.EncodeToString([]byte(md5sign))
	urlsign := url.QueryEscape(base64sign)

	sendval := ps("RequestData=%s&EBusinessID=1304144&RequestType=1002&DataSign=%s&DataType=2", urlval, urlsign)
	//log("%s", sendval)
	req, err := http.NewRequest("POST", "http://api.kdniao.cc/Ebusiness/EbusinessOrderHandle.aspx", strings.NewReader(sendval))
	if err != nil {
		this.Rec = &Recv{5, "请求失败", nil}
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		this.Rec = &Recv{5, "发送验证码失败", nil}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		bodystr := string(body)
		this.Rec = &Recv{3, "请求成功", bodystr}

	} else {
		this.Rec = &Recv{5, "请求失败", nil}
	}

	return
}

func ExpressQuery(shipperCode, logisticCode string) string {
	client := &http.Client{}
	strval := ps(`{"OrderCode":"","ShipperCode":"%s","LogisticCode":"%s"}`, shipperCode, logisticCode)
	urlval := url.QueryEscape(strval)
	strKey := "690d0fe5-2f8c-4c67-900a-c72e3b587924"
	md5sign := StrToMD5(strval + strKey)
	base64sign := base64.StdEncoding.EncodeToString([]byte(md5sign))
	urlsign := url.QueryEscape(base64sign)

	sendval := ps("RequestData=%s&EBusinessID=1304144&RequestType=1002&DataSign=%s&DataType=2", urlval, urlsign)
	req, err := http.NewRequest("POST", "http://api.kdniao.cc/Ebusiness/EbusinessOrderHandle.aspx", strings.NewReader(sendval))
	if err != nil {
		return ""
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			bodystr := string(body)
			return bodystr

		}
	}

	return ""
}
