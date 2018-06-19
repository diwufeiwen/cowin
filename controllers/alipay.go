package controllers

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/astaxie/beego/orm"
	//"github.com/mahonia"
	//"io/ioutil"
	"math/rand"
	//"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AlipayController struct {
	OnlineController
}

type AlipayBaseController struct {
	BaseController
}

var golOrderLock sync.RWMutex
var gloAlipayOrder map[string]*AlipayOrder

type AlipayOrder struct {
	order_type   int
	Out_Trade_No string
	Total_Amount float64
}

//生成随机字符串
func GetRandomString(size int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < size; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func CreateAlipayOrder(platform int32, out_trade_no string, total_amount float64, order_type int, notify_url string) (string, error) {
	mpara := make(map[string]string)
	mpara["app_id"] = conf("alipay_appid")
	if platform == 1 || platform == 2 {
		mpara["method"] = "alipay.trade.app.pay"
	} else if platform == 3 {
		mpara["method"] = "alipay.trade.wap.pay"
		mpara["return_url"] = notify_url
	} else if platform == 4 {
		mpara["method"] = "alipay.trade.page.pay"
		mpara["return_url"] = notify_url
	}
	mpara["charset"] = "utf-8"
	mpara["sign_type"] = "RSA2"
	mpara["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	mpara["version"] = "1.0"
	mpara["notify_url"] = conf("notify_url")
	// biz_content
	var product_code string
	if platform == 1 || platform == 2 {
		product_code = "QUICK_MSECURITY_PAY"
	} else if platform == 3 {
		product_code = "QUICK_WAP_WAY"
	} else if platform == 4 {
		product_code = "FAST_INSTANT_TRADE_PAY" // "FAST_INSTANT_TRADE_PAY"
	}
	mpara["biz_content"] = ps(`{"timeout_express":"15m","subject":"共创平台","out_trade_no":"%s","total_amount":"%v","product_code":"%s","goods_type":"1"}`,
		out_trade_no, total_amount, product_code)

	//生成sign
	sign, err := AlipaySign(mpara)
	if sign == "" || err != nil {
		return "", err
	}
	urlsign := url.QueryEscape(sign)

	//构造返回数据
	request := strings.Join([]string{"app_id=" + url.QueryEscape(mpara["app_id"]),
		"&method=" + url.QueryEscape(mpara["method"]),
		"&charset=" + url.QueryEscape(mpara["charset"]),
		"&sign_type=" + url.QueryEscape(mpara["sign_type"]),
		"&timestamp=" + url.QueryEscape(mpara["timestamp"]),
		"&version=" + url.QueryEscape(mpara["version"]),
		"&biz_content=" + url.QueryEscape(mpara["biz_content"]),
		"&notify_url=" + url.QueryEscape(mpara["notify_url"]),
		"&return_url=" + url.QueryEscape(mpara["return_url"]),
		"&sign=" + urlsign}, "")

	golOrderLock.Lock()
	defer golOrderLock.Unlock()
	if len(gloAlipayOrder) == 0 {
		gloAlipayOrder = make(map[string]*AlipayOrder)
		log("支付宝订单map初始化成功.")
	}
	order := new(AlipayOrder)
	order.order_type = order_type
	order.Out_Trade_No = out_trade_no
	order.Total_Amount = total_amount
	gloAlipayOrder[out_trade_no] = order
	// if platform == 4 {
	// 	res, _ := AlipayTradePagePay(request)
	// 	var dec mahonia.Decoder
	// 	dec = mahonia.NewDecoder("GBK")
	// 	request = dec.ConvertString(string(res))
	// 	log("recv: %v", request)
	// }
	//log("request: %s", request)
	return request, nil
}

//支付宝异步通知
func (this *AlipayBaseController) AlipayNotice() {
	// 解析数据
	var keys []string
	keys = make([]string, len(this.Ctx.Request.Form))
	idx := 0
	for key, _ := range this.Ctx.Request.Form {
		keys[idx] = key
		idx += 1
	}

	if idx <= 0 {
		this.Ctx.Output.Body([]byte("fail"))
		return
	}

	// 参数声明
	var (
		sign_type, alisign, strpara string
		svr_sign                    []byte
	)

	// 按字典顺序排序并构造签名字符窜
	sort.Strings(keys)
	for _, val := range keys {
		if val == "sign" {
			alisign = this.Ctx.Input.Query(val)
			svr_sign, _ = base64.StdEncoding.DecodeString(alisign)
		} else if val == "sign_type" {
			sign_type = this.Ctx.Input.Query(val)
		} else {
			url_value := this.Ctx.Input.Query(val)
			value, _ := url.QueryUnescape(url_value)
			strpara += ps("%s=%s&", val, value)
		}
	}
	strpara = strpara[0 : len(strpara)-1]
	status := RsaPublicVerify(strpara, svr_sign)
	if status == 0 {
		log("验签失败,支付无效.sign_type:%s", sign_type)
	}

	// 检查其他信息
	out_trade_no := this.GetString("out_trade_no")
	app_id := this.GetString("app_id")
	seller_id := this.GetString("seller_id")
	total_amount, _ := this.GetFloat("total_amount", 64)
	trade_status := this.GetString("trade_status")
	if trade_status != "TRADE_SUCCESS" && trade_status != "TRADE_FINISHED" {
		log("交易状态错误:%s", trade_status)
		this.Ctx.Output.Body([]byte("success"))
		return
	}

	golOrderLock.Lock()
	defer golOrderLock.Unlock()
	if val, ok := gloAlipayOrder[out_trade_no]; ok {
		res := 1
		exp_info := ""
		if app_id != conf("alipay_appid") {
			log("app_id异常:%s", app_id)
			exp_info = "app_id验检异常"
			res = 0
		}

		if val.Total_Amount != total_amount {
			log("total_amount异常:%v", total_amount)
			exp_info = "total_amount验检异常"
			res = 0
		}

		if seller_id != conf("seller_id") {
			log("seller_id异常:%s", seller_id)
			exp_info = "seller_id验检异常"
			res = 0
		}

		db := orm.NewOrm()
		if val.order_type == 0 {
			if res == 1 { //校验成功
				var sql string = ps("SELECT id,order_quantity,hosted_mid,user_id from `enjoy_product` where pay_orderno='%s';", out_trade_no)
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

					sql = ps("update `enjoy_product` set pay_status=1,pay_method='支付宝' where id='%d';", id)
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
				var sql string = ps("update `enjoy_product` set `exp_info`='%s' where pay_orderno='%s';", exp_info, out_trade_no)
				_, err := db.Raw(sql).Exec()
				if err != nil {
					log("写入订单异常信息失败:[%v]", err)
				}
			}
		} else if val.order_type == 1 { //充值订单
			if res == 1 {
				var sql string = ps("update `order_recharger` set `result`='0' where recd='%s';", out_trade_no)
				_, err := db.Raw(sql).Exec()
				if err != nil {
					log("写入订单异常信息失败1:[%v]", err)
				} else {
					sql = ps("select `account` from `order_recharger` where recd='%s';", out_trade_no)
					var result []orm.Params
					cnts, err := db.Raw(sql).Values(&result)
					if err != nil {
						log("写入订单异常信息失败2:[%v]", err)
					} else {
						if cnts > 0 {
							account := result[0]["account"].(string)

							var allWallet float64
							if u, ok := UserSessions.QueryloginA(account); ok {
								u.Wallet = u.Wallet + total_amount
								allWallet = u.Wallet
							}

							sql = ps("update `user` set `wallet`=`wallet`+'%v' where account='%s';", total_amount, account)
							_, err := db.Raw(sql).Exec()
							if err != nil {
								log("写入订单异常信息失败3:[%v]", err)
							}

							//插入资金变动流水
							sql = ps("select id from user where account='%s';", account)
							//log("变动流水:%s", sql)
							_, err = db.Raw(sql).Values(&result)
							if err != nil {
								log("插入资金变动流水失败2:[%v]", err)
							} else {
								if len(result) > 0 {
									if result[0]["id"] != nil {
										userid, _ := strconv.Atoi(result[0]["id"].(string))
										AddMoneyFlow("支付宝充值", out_trade_no, allWallet, total_amount, int64(userid), "支付宝")
									}
								}
							}
						}

					}
				}

			} else {
				log("支付失败:[%v]", out_trade_no)
				var sql string = ps("update `order_recharger` set `result`='2' where recd='%s';", out_trade_no)
				_, err := db.Raw(sql).Exec()
				if err != nil {
					log("写入订单异常信息失败1:[%v]", err)
				}
			}
		}
		delete(gloAlipayOrder, out_trade_no)
	} else {
		log("订单号不存在:%s", out_trade_no)
	}
	this.Ctx.Output.Body([]byte("success"))
	return
}

/**
 * RSA签名
 * @param $data 待签名数据
 * @param $private_key_path 商户私钥文件路径
 * return 签名结果
 */
func RsaSign(origData string, privateKey *rsa.PrivateKey) (string, error) {
	h := sha256.New()
	h.Write([]byte(origData))
	digest := h.Sum(nil)

	s, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, digest)
	if err != nil {
		log("rsaSign SignPKCS1v15 error:%s", err)
		return "", err
	}
	data := base64.StdEncoding.EncodeToString(s)
	return string(data), nil
}

func AlipaySign(mpara map[string]string) (string, error) {
	// 1.对key进行升序排序
	sorted_keys := make([]string, 0)
	for k, _ := range mpara {
		sorted_keys = append(sorted_keys, k)
	}
	sort.Strings(sorted_keys)
	//log("键值顺序: %s", sorted_keys)

	// 2.对key=value的键值对用&连接起来,略过控制
	var strdata string = ""
	for _, key := range sorted_keys {
		strdata += ps("%s=%s&", key, mpara[key])
	}
	strdata = strdata[0 : len(strdata)-1]
	// log("待签名字符窜: %s", strdata)

	//开始签名
	block, _ := pem.Decode([]byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAvu0AMn4/aD6mRnkoWlFJfmJw4Q7jBBhm5kfFjW2KdGHH2BdrC8r4/89L7UCfdtlZbWe7ckI0AQaN9s/uA67qUzNSOc6dfN5P5E0NQVW2qGnaS/
oVvTsJkx2ymvnFm9Esq1LLqNtwbEGdacA4cq6ziCskGLJe4Czs1blpZA4BZnVoCq0yhIdFNgRbSo8zflXvBs7imu5kbB4gA8IlKmGTJ5XuERv0/kibzkEuiayA2nwvUn2f7HmcGCjuJy9+4Eeg7giqYEQ8P5wadkn3fUpNOF2flhVY3Dd1x3QwOWtPoXbUFlS744Eu5
SrfDvO0tAWQmkFC8sBX+fTdhv3e7ic3GQIDAQABAoIBABMkmi913w+psGxCe5xKqC7G2gCGyJZBiBewvIsIn6g5oZr2BiKhkEO92iQIpbR56HCCxRWYs7QinxtPD9NIt2/
uJmFraPj7JVGDtD+Hw4+xRVT21zUo9TXN9Xl6b6jG2U64N3lPvz7reUgAIOjGwXN2t+DOCZs6heiL9Zg1m0hVcsBjHRSXVb92RuAN5jADZ0B+bQkfR7llsmz9/kmLUth8npkgzGzxQMr/iKn6t+2KVv8VbyO5ft8O/utFheaTLsGhaxDd7/
w7KnqaT7HKjeW82njBv04/TAg6zj+s3Tceege1kG0HGlUVCl5iYpy8tK19+rh/vUS4blJkdw9Ro80CgYEA8QGRLECZol8wMvaYNhsDhqf6MZWPIgiws2ZFIiKmORKBAYxysPkhDWBAptbmC6XYSgfjQIw4GoU9usSRmp1hsJ1QfPb04K3Ya3j6B1HJrvXzjk/
MLegrc+CrUOdztXfz3EMHeOTgK6z9THhr8k7lJmSUzgZdns+4WeudYzHU+IMCgYEAys3RrCAlzUYEVjV3+Uz5sAUo3IRTz7SaCg+VRb8Cq7V/PWg0o8CtUCnPnFOfTr4xuYYdnfAlWr/
EDDiQDfGWXxVM0Wr8L5P6qoSqCfTzNXy08EqaC1uTnPAkIoF+K8qyfgg5WS2hxxkw59Ba6uheCC5S4a2uvdJ5K/S3482/
ZzMCgYEA33jlhRQNoXsENW1k7F6WIWlm3E1i4FsQhfkhx6o7WZZn2ujBfIo1dLK4oDuKKmjIqrSvqy2Z5DWCbMlSffzLFbp5ZLaVkDSDBfyyUtEq4zoace5aVIMAr702/ZjwOOeWTro0lowbtUP9x8etyIwRfU0skfFjJBxWQ8LvOIh/
g0MCgYBHrLoftTTm+Ynq1fbS2wub1Bb+6J2eWNvgFmXRQpK1EO4pS7ze6ufV3xEK1NsGv11fjjDFcuwgyImHMC5pXyqf7C08Di2WuxvqS/y0jCewjaR9EEClJvZijtSWhWGMJJU0yb9K7z+v2A1awF6BiyJAje6o5/
NMDyjYCiM7lanB1QKBgHOgbiWA+rb5t2l9wkadiLRlHqjCUKEydKvDvV+RLQZSp822hNfyRtQ3j0IhYtG4OhVO3VHQEjBy6TKgExzs2ZKnTncLwhR6IDKyJM7VFda6PTKCsVom14mV1PQSV/BaMX3QG2dtin6pM4w/3AWBvDuq87dZWUlUGvk9gD6WAskl
-----END RSA PRIVATE KEY-----
`))

	if block == nil {
		fmt.Println("rsasign private_key error")
		return "", nil
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log("x509.ParsePKCS1PrivateKey error: %s", err.Error())
		return "", nil
	}

	result, err := RsaSign(strdata, privateKey)
	return result, err
}

// return:0-失败,1-成功
func RsaPublicVerify(strpara string, svr_sign []byte) int {
	var res int = 0
	//开始签名
	block, _ := pem.Decode([]byte(`-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAhMJ8gPa4sPArxbG3icVDyXVYceTI2roH5IX+hhadvT1X9nxiorqhsm7/S+4wjKdSYnKOW98JJtT3LDdyTr2IpIpiQU0AIoZbSR0GYylPz/pWXPdzvEgWRcAfTHPKETPAjGnPUlAf8PzqNnLBMAgynk1RmKpV4f7hXUWO7JhiK3FoFxaVH/NSnVP0r0nXxhL4Ah23Y8F7HxL/RoDdbCsf9q5OF4Aq3aiMzoSpyBYpwOFCxBpVs88RQJSzyjNAij5oG50yB6c7DY5v0aV8wQF1S+hPi1kMSkzq1ZAiwoGXw1DmdiIqlpmBNYiAQLwv3IftFN75PM0MM3oNKpJAi1YLwQIDAQAB
-----END RSA PUBLIC KEY-----
`))

	if block == nil {
		fmt.Println("rsa public_key error")
		return res
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log("x509.ParsePKIXPublicKey error: %s", err.Error())
		return res
	}
	pub := pubKey.(*rsa.PublicKey)

	h := sha256.New()
	h.Write([]byte(strpara))
	digest := h.Sum(nil)
	err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest, svr_sign)
	if err != nil {
		log("rsaSign VerifyPKCS1v15 error:%s", err)
		return res
	}
	return 1
}

//添加资金流水
func AddMoneyFlow(info string, orderno string, balance float64, amount float64, userId int64, pay_method string) {

	sql := ps("insert into `money_flow` (info,orderno,balance,amount,user_id,pay_method,unix) values('%s','%s','%v','%v','%d','%s','%v');",
		info, orderno, balance, amount, userId, pay_method, TimeNow)

	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("插入资金流水异常，失败信息:[%v]", err)
	}
}

func (this *AlipayController) QueryMoneyFlow() {

	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")

	var result []orm.Params
	sql := ps("select * from `money_flow` where user_id=%d order by unix desc limit %d,%d;", this.User.UserId, begidx, counts)
	sqlc := ps("select count(user_id) as num from `money_flow` where user_id=%d", this.User.UserId)
	db := orm.NewOrm()
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询余额明细总数失败:[%v]", err)
		this.Rec = &Recv{5, "查询余额明细失败.", nil}
		return
	}

	total, _ := strconv.Atoi(result[0]["num"].(string))

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询余额明细失败:[%v]", err)
		this.Rec = &Recv{5, "查询余额明细失败.", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}

	this.Rec = &Recv{3, ps("查询余额明细成功!"), &RecvEx{total, result}}
}
