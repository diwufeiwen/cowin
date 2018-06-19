package controllers

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"
)

type AlipayController struct {
	OnlineController
}

type AlipayBaseController struct {
	BaseController
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

func (this *AlipayController) AlipayOrder() {
	mpara := make(map[string]string)
	mpara["app_id"] = conf("alipay_appid")
	if this.User.Platform == 1 || this.User.Platform == 2 {
		mpara["method"] = "alipay.trade.app.pay"
	} else {
		mpara["method"] = "alipay.trade.wap.pay"
	}
	mpara["charset"] = "utf-8"
	mpara["sign_type"] = "RSA2"
	mpara["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	mpara["version"] = "1.0"
	mpara["notify_url"] = "https://api.yddtv.cn:10032/cowin/alipay/notice"
	// biz_content
	out_trade_no := ps("111_%s_KQS", time.Now().Format("20060102150405"))
	mpara["biz_content"] = ps(`{"timeout_express":"15m","subject":"充电宝","out_trade_no":"%s","total_amount":"0.01","product_code":"QUICK_MSECURITY_PAY","goods_type":"1"}`, out_trade_no)

	//生成sign
	sign, err := AlipaySign(mpara)
	if sign == "" || err != nil {
		this.Rec = &Recv{5, "请求失败", nil}
		return
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
		"&sign=" + urlsign}, "")
	this.Rec = &Recv{3, "请求成功", request}
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

func (this *AlipayBaseController) AlipayNotice() {
	// 解析数据
	var keys []string
	keys = make([]string, len(this.Ctx.Request.Form))
	idx := 0
	for key, _ := range this.Ctx.Request.Form {
		keys[idx] = key
		idx += 1
	}

	// 参数声明
	var (
		sign_type, alisign, strpara string
	)

	// 按字典顺序排序并构造签名字符窜
	log("key: %v", keys)
	sort.Strings(keys)
	for _, val := range keys {
		if val == "sign" {
			alisign, _ = url.QueryUnescape(this.Ctx.Input.Query(val))
		} else if val == "sign_type" {
			sign_type = this.Ctx.Input.Query(val)
		} else {
			url_value := this.Ctx.Input.Query(val)
			value, _ := url.QueryUnescape(url_value)
			strpara += ps("%s=%s&", val, value)
		}
	}
	strpara = strpara[0 : len(strpara)-1]
	log("待签名字符窜: %s, svr_sign:%s, sign_type:%s", strpara, alisign, sign_type)
	RsaPublicVerify(strpara, alisign)

	this.Rec = &Recv{-1, "success", "success"}
}

func RsaPublicVerify(strpara, alisign string) (string, error) {
	//开始签名
	block, _ := pem.Decode([]byte(`-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvu0AMn4/aD6mRnkoWlFJfmJw4Q7jBBhm5kfFjW2KdGHH2BdrC8r4/
89L7UCfdtlZbWe7ckI0AQaN9s/
uA67qUzNSOc6dfN5P5E0NQVW2qGnaS/oVvTsJkx2ymvnFm9Esq1LLqNtwbEGdacA4cq6ziCskGLJe4Czs1blpZA4BZnVoCq0yhIdFNgRbSo8zflXvBs7imu5kbB4gA8IlKmGTJ5XuERv0/
kibzkEuiayA2nwvUn2f7HmcGCjuJy9+4Eeg7giqYEQ8P5wadkn3fUpNOF2flhVY3Dd1x3QwOWtPoXbUFlS744Eu5SrfDvO0tAWQmkFC8sBX+fTdhv3e7ic3GQIDAQAB
-----END RSA PUBLIC KEY-----
`))

	if block == nil {
		fmt.Println("rsa public_key error")
		return "", nil
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log("x509.ParsePKIXPublicKey error: %s", err.Error())
		return "", nil
	}
	pub := pubKey.(*rsa.PublicKey)

	h := sha256.New()
	h.Write([]byte(strpara))
	digest := h.Sum(nil)
	err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest, []byte(alisign))
	if err != nil {
		log("rsaSign VerifyPKCS1v15 error:%s", err)
		return "", err
	}
	log("验签成功")
	return "", nil
}
