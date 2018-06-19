package controllers

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/astaxie/beego/orm"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type WxpayController struct {
	OnlineController
}

type WxpayBaseController struct {
	BaseController
}

var gloWxOrderLock sync.RWMutex
var gloWxOrder map[string]*WxpayOrder

type WxpayOrder struct {
	Out_Trade_No string
	Total_fee    string
	Trade_type   string
	Pay_method   string
}

type UnifyOrderResp struct {
	Return_code string `xml:"return_code"`
	Return_msg  string `xml:"return_msg"`
	Appid       string `xml:"appid"`
	Mch_id      string `xml:"mch_id"`
	Nonce_str   string `xml:"nonce_str"`
	Sign        string `xml:"sign"`
	Result_code string `xml:"result_code"` //业务结果:SUCCESS/FAIL
	Prepay_id   string `xml:"prepay_id"`   //预支付交易会话标识(微信生成的预支付会话标识，用于后续接口调用中使用，该值有效期为2小时)
	Trade_type  string `xml:"trade_type"`
	Code_url    string `xml:"code_url"`
	Mweb_url    string `xml:"mweb_url"`
}

type WxRecvEx struct {
	Out_trade_no string
	Prepay_id    string
	AppId        string
	TimeStamp    int64
	NonceStr     string
	Package      string
	SignType     string
	PaySign      string
	Return_code  string
	Return_msg   string
	Result_code  string
}

type WxNativeRecvEx struct {
	Out_trade_no string
	Result_code  string
	Prepay_id    string
	Code_url     string
}

//微信支付 下单签名
func wxpayCalcSign(mReq map[string]interface{}, key string) string {
	// 对参数按照key=value的格式，并按照参数名ASCII字典序排序如下
	sorted_keys := make([]string, 0)
	for k, _ := range mReq {
		sorted_keys = append(sorted_keys, k)
	}
	sort.Strings(sorted_keys)
	var signStrings string
	for _, k := range sorted_keys {
		value := fmt.Sprintf("%v", mReq[k])
		if value != "" {
			signStrings = signStrings + k + "=" + value + "&"
		}
	}

	// 拼接API密钥
	if key != "" {
		signStrings = signStrings + "key=" + key
	}
	//log("%s", signStrings)

	// 进行MD5签名并且将所有字符转为大写
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(signStrings))
	cipherStr := md5Ctx.Sum(nil)
	upperSign := strings.ToUpper(hex.EncodeToString(cipherStr))
	//log("%s", upperSign)
	return upperSign
}

//计算支付签名 跟下单签名不同的地方在于 最后一个字符串连接没有&
func wxpaySign(mReq map[string]interface{}) string {
	//STEP 1, 对key进行升序排序.
	sorted_keys := make([]string, 0)
	for k, _ := range mReq {
		sorted_keys = append(sorted_keys, k)
	}
	sort.Strings(sorted_keys)

	var signStrings string
	for i, k := range sorted_keys {
		value := fmt.Sprintf("%v", mReq[k])
		if value != "" {
			if i != (len(sorted_keys) - 1) {
				signStrings = signStrings + k + "=" + value + "&"
			} else {
				signStrings = signStrings + k + "=" + value //最后一个不加此符号
			}
		}
	}

	//STEP3, 在键值对的最后加上key=API_KEY
	signStrings = signStrings + "&key=" + conf("wxzf_key")
	//log("支付签名: %s", signStrings)

	md5Ctx := md5.New()
	md5Ctx.Write([]byte(signStrings))
	cipherStr := md5Ctx.Sum(nil)
	upperSign := strings.ToUpper(hex.EncodeToString(cipherStr))
	//log("签名: %s", upperSign)
	return upperSign
}

// 微信公众号订单
func CreateWxOrder(openid string, total_fee float64, notify_url string, ip string) (request *WxRecvEx, err error) {
	// 下单逻辑1:统一下单
	type UnifyOrderReq struct {
		Appid            string `xml:"appid"`            //公众账号ID
		Mch_id           string `xml:"mch_id"`           //商户号
		Nonce_str        string `xml:"nonce_str"`        //随机字符串
		Sign             string `xml:"sign"`             //签名
		Body             string `xml:"body"`             //商品描述
		Out_trade_no     string `xml:"out_trade_no"`     //商户订单号
		Total_fee        int    `xml:"total_fee"`        //标价金额(单位:分)
		Spbill_create_ip string `xml:"spbill_create_ip"` //终端IP
		Notify_url       string `xml:"notify_url"`       //通知地址
		Trade_type       string `xml:"trade_type"`       //交易类型
		Openid           string `xml:"openid"`           //用户标识trade_type=JSAPI时(即公众号支付),此参数必传,为微信用户在商户对应appid下的唯一标识
	}

	var myReq UnifyOrderReq
	myReq.Appid = conf("wx_appid")
	myReq.Mch_id = conf("wx_mchid")
	myReq.Nonce_str = GetRandomString(32)
	myReq.Body = "cowin"
	myReq.Out_trade_no = strconv.FormatInt(time.Now().Unix(), 10) + GetRandomString(3)
	myReq.Total_fee = int(total_fee * 100.0)
	myReq.Spbill_create_ip = ip
	myReq.Notify_url = conf("wx_url")
	myReq.Trade_type = "JSAPI"
	myReq.Openid = openid

	// 构造签名
	var m map[string]interface{}
	m = make(map[string]interface{}, 0)
	m["appid"] = myReq.Appid
	m["body"] = myReq.Body
	m["mch_id"] = myReq.Mch_id
	m["notify_url"] = myReq.Notify_url
	m["trade_type"] = myReq.Trade_type
	m["spbill_create_ip"] = myReq.Spbill_create_ip
	m["total_fee"] = myReq.Total_fee
	m["out_trade_no"] = myReq.Out_trade_no
	m["nonce_str"] = myReq.Nonce_str
	m["openid"] = myReq.Openid
	myReq.Sign = wxpayCalcSign(m, conf("wxzf_key"))

	// 转换为xml格式
	bytes_req, err := xml.Marshal(myReq)
	if err != nil {
		log("转换为xml格式错误:[%v]", err)
		err = errors.New("支付失败:构造xml数据错误")
		return
	}
	str_req := strings.Replace(string(bytes_req), "UnifyOrderReq", "xml", -1)
	bytes_req = []byte(str_req)
	//log("发送数据:[%v]", str_req)

	// 发送下单请求
	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/unifiedorder", bytes.NewReader(bytes_req))
	if err != nil {
		log("New Http Request发生错误:[%v]", err)
		err = errors.New("支付失败:New Http Request错误")
		return
	}
	// 设置http header
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Content-Type", "application/xml;charset=utf-8")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log("请求微信支付统一下单接口发送错误:[%v]", err)
		err = errors.New("支付失败:请求微信支付统一下单接口发送错误")
		return
	}

	// 处理返回结果
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log("解析返回body错误:[%v]", err)
			err = errors.New("支付失败:解析返回body错误")
		} else {
			xmlResp := UnifyOrderResp{}
			err = xml.Unmarshal(respBytes, &xmlResp)
			if err != nil {
				log("解析数据错误:[%v]", err)
			} else {
				if xmlResp.Return_code == "FAIL" {
					log("微信支付统一下单失败:[%v]", xmlResp)
					err = errors.New(ps("微信支付统一下单失败:%s", xmlResp.Return_msg))
				} else {
					// 加入订单
					gloWxOrderLock.Lock()
					defer gloWxOrderLock.Unlock()
					if len(gloWxOrder) == 0 {
						gloWxOrder = make(map[string]*WxpayOrder)
						log("微信订单map初始化成功.")
					}
					order := new(WxpayOrder)
					order.Out_Trade_No = myReq.Out_trade_no
					order.Total_fee = ps("%d", myReq.Total_fee)
					order.Trade_type = myReq.Trade_type
					order.Pay_method = "微信公众号"
					gloWxOrder[order.Out_Trade_No] = order

					// 生成支付签名
					timeStamp := time.Now().Unix()
					p := make(map[string]interface{}, 0)
					p["appId"] = myReq.Appid
					p["timeStamp"] = timeStamp
					p["nonceStr"] = myReq.Nonce_str
					p["package"] = "prepay_id=" + xmlResp.Prepay_id
					p["signType"] = "MD5"
					request = &WxRecvEx{myReq.Out_trade_no, xmlResp.Prepay_id, xmlResp.Appid, timeStamp, myReq.Nonce_str, "prepay_id=" + xmlResp.Prepay_id, "MD5", wxpaySign(p), xmlResp.Return_code, xmlResp.Return_msg, xmlResp.Result_code}
					err = nil
				}
			}
		}
	} else {
		err = errors.New(ps("支付失败,错误状态码: %d", resp.StatusCode))
	}
	return
}

// 微信网站支付统一下单
func CreateWxNativeOrder(out_trade_no string, total_fee float64, notify_url string, ip string) (request *WxNativeRecvEx, err error) {
	// 下单逻辑1:统一下单
	type UnifyOrderReq struct {
		Appid            string `xml:"appid"`            //公众账号ID
		Mch_id           string `xml:"mch_id"`           //商户号
		Nonce_str        string `xml:"nonce_str"`        //随机字符串
		Sign             string `xml:"sign"`             //签名
		Body             string `xml:"body"`             //商品描述
		Out_trade_no     string `xml:"out_trade_no"`     //商户订单号
		Total_fee        int    `xml:"total_fee"`        //标价金额(单位:分)
		Spbill_create_ip string `xml:"spbill_create_ip"` //终端IP
		Notify_url       string `xml:"notify_url"`       //通知地址
		Trade_type       string `xml:"trade_type"`       //交易类型
		Product_id       string `xml:"product_id"`       //商品ID,trade_type=NATIVE时(即扫码支付),此参数必传,此参数为二维码中包含的商品ID,商户自行定义
	}

	var myReq UnifyOrderReq
	myReq.Appid = conf("wx_appid")
	myReq.Mch_id = conf("wx_mchid")
	myReq.Nonce_str = GetRandomString(32)
	myReq.Body = "共创平台-商城中心"
	myReq.Out_trade_no = out_trade_no
	myReq.Total_fee = int(total_fee * 100.0)
	myReq.Spbill_create_ip = ip
	myReq.Notify_url = conf("wx_url")
	myReq.Trade_type = "NATIVE"
	myReq.Product_id = GetRandomString(16)

	// 构造签名
	var m map[string]interface{}
	m = make(map[string]interface{}, 0)
	m["appid"] = myReq.Appid
	m["body"] = myReq.Body
	m["mch_id"] = myReq.Mch_id
	m["notify_url"] = myReq.Notify_url
	m["trade_type"] = myReq.Trade_type
	m["spbill_create_ip"] = myReq.Spbill_create_ip
	m["total_fee"] = myReq.Total_fee
	m["out_trade_no"] = myReq.Out_trade_no
	m["nonce_str"] = myReq.Nonce_str
	m["product_id"] = myReq.Product_id
	myReq.Sign = wxpayCalcSign(m, conf("wxzf_key"))

	// 转换为xml格式
	bytes_req, err := xml.Marshal(myReq)
	if err != nil {
		log("转换为xml格式错误:[%v]", err)
		err = errors.New("支付失败:构造xml数据错误")
		return
	}
	str_req := strings.Replace(string(bytes_req), "UnifyOrderReq", "xml", -1)
	bytes_req = []byte(str_req)
	//log("xml请求数据:[%v]", str_req)

	// 发送下单请求
	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/unifiedorder", bytes.NewReader(bytes_req))
	if err != nil {
		log("New Http Request发生错误:[%v]", err)
		err = errors.New("支付失败:New Http Request错误")
		return
	}
	// 设置http header
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Content-Type", "application/xml;charset=utf-8")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log("请求微信支付统一下单接口发送错误:[%v]", err)
		err = errors.New("支付失败:请求微信支付统一下单接口发送错误")
		return
	}

	// 处理返回结果
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log("body错误:[%v]", err)
			err = errors.New("支付失败:解析返回body错误")
		} else {
			xmlResp := UnifyOrderResp{}
			err = xml.Unmarshal(respBytes, &xmlResp)
			if err != nil {
				log("解析数据错误:[%v]", err)
			} else {
				//log("微信返回数据:%s", respBytes)
				if xmlResp.Return_code == "FAIL" {
					log("微信支付统一下单失败:[%v]", xmlResp)
					err = errors.New(ps("微信支付统一下单失败:%s", xmlResp.Return_msg))
				} else {
					// 参数校验
					if xmlResp.Appid != m["appid"] {
						err = errors.New("appid校验异常")
					} else if xmlResp.Mch_id != m["mch_id"] {
						err = errors.New("mch_id校验异常")
					} else if xmlResp.Trade_type != myReq.Trade_type {
						err = errors.New("trade_type校验异常")
					} else {
						// 加入订单
						gloWxOrderLock.Lock()
						defer gloWxOrderLock.Unlock()
						if len(gloWxOrder) == 0 {
							gloWxOrder = make(map[string]*WxpayOrder)
							log("微信订单map初始化成功.")
						}
						order := new(WxpayOrder)
						order.Out_Trade_No = myReq.Out_trade_no
						order.Total_fee = ps("%d", myReq.Total_fee)
						order.Trade_type = myReq.Trade_type
						order.Pay_method = "微信网站"
						gloWxOrder[order.Out_Trade_No] = order

						request = &WxNativeRecvEx{myReq.Out_trade_no, xmlResp.Result_code, xmlResp.Prepay_id, xmlResp.Code_url}
						err = nil
					}
					if err != nil {
						log("下单失败: %v", err)
					}
				}
			}
		}
	} else {
		err = errors.New(ps("支付失败,错误状态码: %d", resp.StatusCode))
	}
	return
}

// 微信H5支付统一下单
func CreateWxMwebOrder(out_trade_no string, total_fee float64, notify_url string, ip string) (request *WxNativeRecvEx, err error) {
	// 下单逻辑1:统一下单
	type UnifyOrderReq struct {
		Appid            string `xml:"appid"`            //公众账号ID
		Mch_id           string `xml:"mch_id"`           //商户号
		Nonce_str        string `xml:"nonce_str"`        //随机字符串
		Sign             string `xml:"sign"`             //签名
		Body             string `xml:"body"`             //商品描述
		Out_trade_no     string `xml:"out_trade_no"`     //商户订单号
		Total_fee        int    `xml:"total_fee"`        //标价金额(单位:分)
		Spbill_create_ip string `xml:"spbill_create_ip"` //终端IP
		Notify_url       string `xml:"notify_url"`       //通知地址
		Trade_type       string `xml:"trade_type"`       //交易类型
	}

	var myReq UnifyOrderReq
	myReq.Appid = conf("wx_appid")
	myReq.Mch_id = conf("wx_mchid")
	myReq.Nonce_str = GetRandomString(32)
	myReq.Body = "共创平台-商城中心"
	myReq.Out_trade_no = out_trade_no
	myReq.Total_fee = int(total_fee * 100.0)
	myReq.Spbill_create_ip = ip
	myReq.Notify_url = conf("wx_url")
	myReq.Trade_type = "MWEB"

	// 构造签名
	var m map[string]interface{}
	m = make(map[string]interface{}, 0)
	m["appid"] = myReq.Appid
	m["body"] = myReq.Body
	m["mch_id"] = myReq.Mch_id
	m["notify_url"] = myReq.Notify_url
	m["trade_type"] = myReq.Trade_type
	m["spbill_create_ip"] = myReq.Spbill_create_ip
	m["total_fee"] = myReq.Total_fee
	m["out_trade_no"] = myReq.Out_trade_no
	m["nonce_str"] = myReq.Nonce_str
	myReq.Sign = wxpayCalcSign(m, conf("wxzf_key"))

	// 转换为xml格式
	bytes_req, err := xml.Marshal(myReq)
	if err != nil {
		log("转换为xml格式错误:[%v]", err)
		err = errors.New("支付失败:构造xml数据错误")
		return
	}
	str_req := strings.Replace(string(bytes_req), "UnifyOrderReq", "xml", -1)
	bytes_req = []byte(str_req)
	//log("xml请求数据:[%v]", str_req)

	// 发送下单请求
	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/unifiedorder", bytes.NewReader(bytes_req))
	if err != nil {
		log("New Http Request发生错误:[%v]", err)
		err = errors.New("支付失败:New Http Request错误")
		return
	}
	// 设置http header
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Content-Type", "application/xml;charset=utf-8")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log("请求微信支付统一下单接口发送错误:[%v]", err)
		err = errors.New("支付失败:请求微信支付统一下单接口发送错误")
		return
	}

	// 处理返回结果
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log("body错误:[%v]", err)
			err = errors.New("支付失败:解析返回body错误")
		} else {
			xmlResp := UnifyOrderResp{}
			err = xml.Unmarshal(respBytes, &xmlResp)
			if err != nil {
				log("解析数据错误:[%v]", err)
			} else {
				//log("微信返回数据:%s", respBytes)
				if xmlResp.Return_code == "FAIL" {
					log("微信支付统一下单失败:[%v]", xmlResp)
					err = errors.New(ps("微信支付统一下单失败:%s", xmlResp.Return_msg))
				} else {
					// 参数校验
					if xmlResp.Appid != m["appid"] {
						err = errors.New("appid校验异常")
					} else if xmlResp.Mch_id != m["mch_id"] {
						err = errors.New("mch_id校验异常")
					} else if xmlResp.Trade_type != myReq.Trade_type {
						err = errors.New("trade_type校验异常")
					} else {
						// 加入订单
						gloWxOrderLock.Lock()
						defer gloWxOrderLock.Unlock()
						if len(gloWxOrder) == 0 {
							gloWxOrder = make(map[string]*WxpayOrder)
							log("微信订单map初始化成功.")
						}
						order := new(WxpayOrder)
						order.Out_Trade_No = myReq.Out_trade_no
						order.Total_fee = ps("%d", myReq.Total_fee)
						order.Trade_type = myReq.Trade_type
						order.Pay_method = "微信H5"
						gloWxOrder[order.Out_Trade_No] = order

						request = &WxNativeRecvEx{myReq.Out_trade_no, xmlResp.Result_code, xmlResp.Prepay_id, xmlResp.Mweb_url}
						err = nil
					}
					if err != nil {
						log("下单失败: %v", err)
					}
				}
			}
		}
	} else {
		err = errors.New(ps("支付失败,错误状态码: %d", resp.StatusCode))
	}
	return
}

// 查询订单接口
func WxOrderQuery(out_trade_no string) (res string, err error) {
	type UnifyOrderReq struct {
		Appid        string `xml:"appid"`        //公众账号ID
		Mch_id       string `xml:"mch_id"`       //商户号
		Out_trade_no string `xml:"out_trade_no"` //商户订单号
		Nonce_str    string `xml:"nonce_str"`    //随机字符串
		Sign         string `xml:"sign"`         //签名
	}

	var myReq UnifyOrderReq
	myReq.Appid = conf("wx_appid")
	myReq.Mch_id = conf("wx_mchid")
	myReq.Out_trade_no = out_trade_no
	myReq.Nonce_str = GetRandomString(32)

	// 构造签名
	var m map[string]interface{}
	m = make(map[string]interface{}, 0)
	m["appid"] = myReq.Appid
	m["mch_id"] = myReq.Mch_id
	m["out_trade_no"] = myReq.Out_trade_no
	m["nonce_str"] = myReq.Nonce_str
	myReq.Sign = wxpayCalcSign(m, conf("wxzf_key"))

	// 转换为xml格式
	bytes_req, err := xml.Marshal(myReq)
	if err != nil {
		log("转换为xml格式错误:[%v]", err)
		err = errors.New("支付失败:构造xml数据错误")
		return
	}
	str_req := strings.Replace(string(bytes_req), "UnifyOrderReq", "xml", -1)
	bytes_req = []byte(str_req)
	//log("xml请求数据:[%v]", str_req)

	// 发送下单请求
	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/orderquery", bytes.NewReader(bytes_req))
	if err != nil {
		log("New Http Request发生错误:[%v]", err)
		err = errors.New("支付失败:New Http Request错误")
		return
	}
	// 设置http header
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Content-Type", "application/xml;charset=utf-8")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log("请求微信查询订单接口发送错误:[%v]", err)
		err = errors.New("支付失败:请求微信查询订单接口发送错误")
		return
	}

	// 处理返回结果
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log("读取body错误:[%v]", err)
			err = errors.New("支付失败:读取返回body错误")
		} else {
			type OrderQueryRecv struct {
				Return_code    string `xml:"return_code"`
				Return_msg     string `xml:"return_msg"`
				Appid          string `xml:"appid"`
				Mch_id         string `xml:"mch_id"`
				Nonce_str      string `xml:"nonce_str"`
				Sign           string `xml:"sign"`
				Result_code    string `xml:"result_code"` //业务结果:SUCCESS/FAIL
				Err_code       string `xml:"err_code"`
				Err_code_des   string `xml:"err_code_des"`
				Open_id        string `xml:"openid"`
				Is_subscribe   string `xml:"is_subscribe"`
				Trade_type     string `xml:"trade_type"`
				Bank_type      string `xml:"bank_type"`
				Total_fee      string `xml:"total_fee"`
				Fee_type       string `xml:"fee_type"`
				Transaction_id string `xml:"transaction_id"`
				Out_trade_no   string `xml:"out_trade_no"`
				trade_state    string `xml:"trade_state"`
			}
			xmlResp := OrderQueryRecv{}
			err = xml.Unmarshal(respBytes, &xmlResp)
			if err != nil {
				log("解析数据错误:[%v]", err)
			} else {
				err = nil
				res = xmlResp.Result_code
			}
		}
	} else {
		err = errors.New(ps("支付失败,错误状态码: %d", resp.StatusCode))
	}
	return
}

// sid,out_trade_no(订单号)
func (this *WxpayController) Recharge() {
	out_trade_no := this.GetString("out_trade_no")
	log("%s", out_trade_no)
	// 参数校验
	if !CheckArg(out_trade_no) {
		this.Rec = &Recv{5, "订单号不能为空", nil}
		return
	}

	// 业务逻辑
	db := orm.NewOrm()
	var sql string = ps("SELECT pay_status,exp_info from `enjoy_product` where pay_orderno='%s';", out_trade_no)
	//log("sql :%s", sql)
	var result []orm.Params
	cnts, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询订单支付信息失败:[%v]", err)
		return
	}

	if cnts <= 0 {
		this.Rec = &Recv{5, ps("查询的订单号不存在:%s", out_trade_no), nil}
		return
	}

	this.Rec = &Recv{3, "查询订单支付状态成功", result}
	return
}

func (this *WxpayBaseController) WxpayNotice() {
	type WxNotice struct {
		Appid          string `xml:"appid"`
		Bank_type      string `xml:"bank_type"`
		Cash_fee       string `xml:"cash_fee"`
		Fee_type       string `xml:"fee_type"`
		Is_subscribe   string `xml:"is_subscribe"`
		Mch_id         string `xml:"mch_id"`
		Nonce_str      string `xml:"nonce_str"`
		Openid         string `xml:"openid"`
		Out_trade_no   string `xml:"out_trade_no"`
		Result_code    string `xml:"result_code"`
		Return_code    string `xml:"return_code"`
		Sign           string `xml:"sign"`
		Time_end       string `xml:"time_end"`
		Total_fee      string `xml:"total_fee"`
		Trade_type     string `xml:"trade_type"`
		Transaction_id string `xml:"transaction_id"`
	}

	// 解析数据
	xmlResp := WxNotice{}
	err := xml.Unmarshal(this.Ctx.Input.RequestBody, &xmlResp)
	if err != nil {
		log("解析数据错误:[%v]", err)
		return
	}

	// 数据验证
	gloWxOrderLock.Lock()
	defer gloWxOrderLock.Unlock()
	if val, ok := gloWxOrder[xmlResp.Out_trade_no]; ok {
		res := 1
		exp_info := ""

		if xmlResp.Result_code != "SUCCESS" {
			log("Result_code异常:%s", xmlResp.Result_code)
			exp_info = ps("Result_code异常:%s", xmlResp.Result_code)
			res = 0
		}

		if xmlResp.Return_code != "SUCCESS" {
			log("Return_code异常:%s", xmlResp.Return_code)
			exp_info = ps("Return_code异常:%s", xmlResp.Return_code)
			res = 0
		}

		if xmlResp.Appid != conf("wx_appid") {
			log("app_id异常:%s", xmlResp.Appid)
			exp_info = "app_id验检异常"
			res = 0
		}

		if xmlResp.Mch_id != conf("wx_mchid") {
			log("Mch_id异常:%s", xmlResp.Mch_id)
			exp_info = "Mch_id验检异常"
			res = 0
		}

		if val.Out_Trade_No != xmlResp.Out_trade_no {
			log("Out_trade_no异常:%s", val.Out_Trade_No)
			exp_info = "Out_trade_no验检异常"
			res = 0
		}

		if val.Total_fee != xmlResp.Total_fee {
			log("Total_fee异常:%s", val.Total_fee)
			exp_info = "Total_fee验检异常"
			res = 0
		}

		if val.Trade_type != xmlResp.Trade_type {
			log("Trade_type异常:%s", val.Trade_type)
			exp_info = "Trade_type验检异常"
			res = 0
		}

		db := orm.NewOrm()
		if res == 1 { //校验成功
			var sql string = ps("SELECT id,order_quantity,hosted_mid,user_id from `enjoy_product` where pay_orderno='%s';", xmlResp.Out_trade_no)
			var result []orm.Params
			_, err := db.Raw(sql).Values(&result)
			if err != nil {
				log("查询订单信息失败:[%v]", err)
			}

			for i := range result {
				item := result[i]
				id, _ := strconv.Atoi(item["id"].(string))
				order_quantity, _ := strconv.Atoi(item["order_quantity"].(string))
				hosted_mid, _ := strconv.Atoi(item["hosted_mid"].(string))
				user_id, _ := strconv.Atoi(item["user_id"].(string))

				code, strerr := GenerateUserProduct(int64(id), order_quantity, hosted_mid, int64(user_id))
				if code == 5 {
					log("生成产品订单失败:%s", strerr)
					continue
				}

				sql = ps("update `enjoy_product` set pay_status=1,pay_method='%s' where id='%d';", val.Pay_method, id)
				_, err = db.Raw(sql).Exec()
				if err != nil {
					log("更新订单[%d]支付状态失败:[%v]", id, err)
					continue
				}

				// 托管订单生成合同
				if hosted_mid == 1 {
					sql = ps("insert into `agreement` (ep_id,text,unix) values('%d','%s','%d');", id, "", TimeNow)
					_, err = db.Raw(sql).Exec()
					if err != nil {
						log("生成订单[%d]合同失败:[%v]", id, err)
						continue
					}
				}

				// 添加订单状态
				AddLogisticsInfo(0, int64(id), "等待厂商生产...")
			}

		} else {
			var sql string = ps("update `enjoy_product` set `exp_info`='%s' where pay_orderno='%s';", exp_info, xmlResp.Out_trade_no)
			_, err := db.Raw(sql).Exec()
			if err != nil {
				log("写入订单异常信息失败:[%v]", err)
			}
		}

		delete(gloWxOrder, xmlResp.Out_trade_no)
	} else {
		log("%s订单号不存在", xmlResp.Out_trade_no)
	}

	this.Ctx.Output.Body([]byte("success"))
	return
}

// url
func (this *WxpayBaseController) WxSharing() {
	url := this.GetString("url")

	// 业务逻辑
	tm := TimeNow
	noncestr := GetRandomString(16)
	str := ps("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", Jsapi_Ticket, noncestr, tm, url)

	log("签名前:%s", str)
	sha1Ctx := sha1.New()
	sha1Ctx.Write([]byte(str))
	sign := ps("%x", sha1Ctx.Sum(nil))

	type RecvEx struct {
		Sign     string
		Tm       int64
		Noncestr string
	}

	this.Rec = &Recv{3, "构造签名成功.", &RecvEx{ps("%s", sign), tm, noncestr}}
	return
}
