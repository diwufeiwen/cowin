package controllers

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/plugins/cors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mahonia"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

/* 微信信息*/
var (
	Access_Token   string
	Token_Expires  int64
	Jsapi_Ticket   string // 公众号用于调用微信JS接口的临时票据
	Ticket_Expires int64
)

/* account to be index*/
var VerCodes map[string]*vcode_t
var TimeNow int64 = 0
var TimeNowStr string = "2017-02-10 15:01:30"
var ps = fmt.Sprintf
var conf = beego.AppConfig.String
var Authlist map[string]*Auth //权限列表
var SidLife int64

/* code 3-success 5-error*/
type Recv struct {
	Code int
	Msg  string
	Data interface{}
}

/* store vcode*/
type vcode_t struct {
	code     string
	lasttime int64
}

type Loginuser struct {
	SessionId  string
	UserId     int64
	Account    string
	Nick       string
	Intro      string
	Flag       int64
	Level      int64
	Wallet     float64
	CanWallet  float64
	Src        string
	City       string
	DealerAcc  string
	Ptid       int32
	Phone      string
	Address    string
	Realname   string
	LastTime   int64
	Supid      int64 // 上级管理者的uid
	Deduct     float64
	Spreadlink string
	Qrcode     string
	WxOpenid   string
	Platform   int32
	Auth       map[string]*Auth
}

type WXUserInfo struct {
	Openid     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int64  `json:"sex"`
	City       string `json:"city"`
	Province   string `json:"province"`
	Country    string `json:"country"`
	Headimgurl string `json:"headimgurl"`
	Unionid    string `json:"unionid"`
}

type Auth struct {
	Id   string
	Name string
	Url  string
}

func init() {
	TimenowInit()
	MinuteTimerInit() // 过期未支付订单处理
	ArgInit()
	MysqlInit()
	FileInit()
	OriginInit()
	AuthlistInit()

	//log("%s", res)
	WxTokenInit() //微信信息初始化
	WxJsapiInit()

	// 读取利率
	ReadInterestRate()

	// 定时请求产品信息
	UpdateUserProductInfo()
	UpdateProductUserInfo() //产品使用信息

	// 内部用:修改用户初始权限
	// AuthInit()
}

func TimenowInit() {
	go func() {
		for {
			tim_t := time.Now()
			TimeNow = tim_t.Unix()
			TimeNowStr = tim_t.Format("2006-01-02 15:04:05")
			time.Sleep(time.Second)
			// 判断微信Access_Token是否过期
			if TimeNow >= Token_Expires {
				GetAccessToken()
			}
			// 判断微信Jsapi_Ticket是否过期
			if TimeNow >= Ticket_Expires {
				GetJsapiTicket()
			}
		}
	}()
}

func log(format string, v ...interface{}) {
	fmt.Println(fmt.Sprintf("[debug][%s]", TimeNowStr), fmt.Sprintf(format, v...))
	return
}

// 获取利率
type InterestRate struct {
	Ptid       int
	BeginMonth int
	EndMonth   int
	OpeRatio   float64
	UserRatio  float64
	TaxRatio   float64
	PlatRatio  float64
}

var CdbRate []InterestRate

func ReadInterestRate() {
	sql := "select * from `profit_ratio` where `operate`=0;"
	db := orm.NewOrm()
	var res []orm.Params
	nums, err := db.Raw(sql).Values(&res)
	if err != nil {
		log("读取税率失败:[%v]", err)
	}

	// 解析数据
	CdbRate = make([]InterestRate, nums)
	for idx := range res {
		item := res[idx]
		CdbRate[idx].Ptid, _ = strconv.Atoi(item["pt_id"].(string))
		idarr := strings.Split(item["put_month"].(string), "-")
		CdbRate[idx].BeginMonth, _ = strconv.Atoi(idarr[0])
		CdbRate[idx].EndMonth, _ = strconv.Atoi(idarr[1])
		CdbRate[idx].OpeRatio, _ = strconv.ParseFloat(item["ope_ratio"].(string), 64)
		CdbRate[idx].UserRatio, _ = strconv.ParseFloat(item["user_ratio"].(string), 64)
		CdbRate[idx].TaxRatio, _ = strconv.ParseFloat(item["tax_ratio"].(string), 64)
		CdbRate[idx].PlatRatio, _ = strconv.ParseFloat(item["plat_ratio"].(string), 64)
	}
}

func ArgInit() {
	SidLife, _ = beego.AppConfig.Int64("sidlife")
	if SidLife == 0 {
		SidLife = 250000 // 3天左右
	}

	// 捕获panic异常
	defer func() {
		if err := recover(); err != nil {
			log("panic: %v", err)
		}
	}()
	VerCodes = make(map[string]*vcode_t)
}

func AuthInit() {
	var sql string = "truncate table `users_auth`;"
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("清理权限分配表出错:[%v]", err)
		return
	}

	sql = "SELECT id,flag from user;"
	var result []orm.Params
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询用户表出错:[%v]", err)
		return
	}

	// 给用户分配权限
	for idx := range result {
		item := result[idx]
		flag, _ := strconv.Atoi(item["flag"].(string))

		var sqls []string
		switch flag {
		case 1: // 管理员
			// sqls = append(sqls, ps("insert into users_auth (uid,aid) values ('%s','%s');", item["id"].(string), Authlist["/cowin/tidea/add"].Id))
		}

		db.Begin() //开启事务
		for _, ele := range sqls {
			db.Raw(ele).Exec()
		}
		db.Commit() //提交事务
	}
}

func AuthlistInit() {
	Authlist = make(map[string]*Auth)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw("select * from auth").Values(&result)
	if err != nil {
		log("查询数据库失败:err[%v]", err)
	} else {
		// 用描述信息来映射用户是否有权限添加主要的列表信息
		for _, val := range result {
			auth := new(Auth)
			auth.Id = val["id"].(string)
			auth.Name = val["name"].(string)
			auth.Url = val["url"].(string)
			Authlist[auth.Url] = auth
		}
	}
}

func MysqlInit() {
	var maxIdle int = 30
	var maxConn int = 30
	orm.RegisterDataBase("default", "mysql", conf("mysql"), maxIdle, maxConn)
	log("mysql init success")
	return
}

func FileInit() {
	log("File init begin")
	beego.SetStaticPath("/static", "static")
	beego.SetStaticPath("/cowin/down", conf("tmppath"))
	beego.SetStaticPath("/cowin/head", conf("headpath"))
	beego.SetStaticPath("/cowin/level", conf("levelpath"))
	beego.SetStaticPath("/cowin/talk", conf("talkpath"))
	beego.SetStaticPath("/cowin/apk", conf("apkpath"))
	beego.SetStaticPath("/cowin/carousel", conf("carouselpath"))
	beego.SetStaticPath("/cowin/mall", conf("mallpath"))
	beego.SetStaticPath("/cowin/dealer", conf("dealerpath"))
	beego.SetStaticPath("/cowin/personal", conf("personalpath"))
	beego.SetStaticPath("/cowin/activity", conf("activitypath"))

	err := os.MkdirAll(conf("headpath"), os.ModePerm)
	if err != nil {
		log("head创建文件夹失败err[%v]", err)
	}

	err = os.MkdirAll(conf("levelpath"), os.ModePerm)
	if err != nil {
		log("levelpath创建文件夹失败err[%v]", err)
	}

	err = os.MkdirAll(conf("talkpath"), os.ModePerm)
	if err != nil {
		log("talkpath创建文件夹失败err[%v]", err)
	}

	err = os.MkdirAll(conf("apkpath"), os.ModePerm)
	if err != nil {
		log("apkpath创建文件夹失败err[%v]", err)
	}
	
	err = os.MkdirAll(conf("activitypath"), os.ModePerm)
	if err != nil {
		log("activitypath创建文件夹失败err[%v]", err)
	}

	err = os.MkdirAll(conf("carouselpath"), os.ModePerm)
	if err != nil {
		log("carouselpath创建文件夹失败err[%v]", err)
	}

	err = os.MkdirAll(conf("mallpath"), os.ModePerm)
	if err != nil {
		log("mallpath创建文件夹失败err[%v]", err)
	}

	err = os.MkdirAll(conf("dealerpath"), os.ModePerm)
	if err != nil {
		log("dealerpath创建文件夹失败err[%v]", err)
	}

	err = os.MkdirAll(conf("personalpath"), os.ModePerm)
	if err != nil {
		log("personalpath创建文件夹失败err[%v]", err)
	}

	log("File init end")
	return
}

func OriginInit() {
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "DELETE", "PUT", "PATCH", "POST"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	log("OriginInit init success")
	return
}

func StrToMD5(str string) (result string) {
	md5Ctx1 := md5.New()
	md5Ctx1.Write([]byte(str))
	result = ps("%x", md5Ctx1.Sum(nil))
	return
}

func CheckArg(a ...interface{}) bool {
	for _, arg := range a {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.String:
			if arg.(string) == "" {
				return false
			}
		case reflect.Int64:
			if arg.(int64) == 0 {
				return false
			}
		case reflect.Int32:
			if arg.(int32) == 0 {
				return false
			}

		case reflect.Int:
			if arg.(int) == 0 {
				return false
			}

		case reflect.Float64:
			if arg.(float64) < 0.000001 && arg.(float64) > -0.000001 {
				return false
			}
		case reflect.Float32:
			if arg.(float32) < 0.000001 && arg.(float32) > -0.000001 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func ChecSQLerr(err error) (code int64, info string) {
	if err == nil {
		code = 0
		return
	} else {
		str := err.Error()
		for i := 0; i < len(str); i++ {
			if str[i] == ':' {
				code, _ = strconv.ParseInt(str[6:i], 10, 64)
				info = str[i+1:]
				return
			}
		}
	}
	code = 9999
	info = err.Error()
	return
}

func SendMsg(telnum string, label string) (isok bool) {
	//中文转gb2312要优化！！！！！！！！！
	var content string
	enc := mahonia.NewEncoder("GBK")
	output := enc.ConvertString(label)

	// p(output)
	content = output + `%a1%be%b9%b2%b4%b4%a1%bf` //句号+【一线财经】,删两个是删句号 urlencode
	// p(telnum)
	t_time := time.Now()
	systime := t_time.Format("2006-Jan-02 15:04:05") //格式化时间
	h := md5.New()
	io.WriteString(h, "p62577798"+systime)
	md5toHex := fmt.Sprintf("%x", h.Sum(nil)) //加密

	host := fmt.Sprintf("http://www.sms1086.com/plan/Api/Send.aspx?username=13524691652&password=%s&mobiles=%s&content=%s&", md5toHex, telnum, content)
	q := url.Values{} //Values 是一个map[string][]string结构
	// p(host)
	q.Set("timestamp", systime)
	str := host + q.Encode() //把systime化成%的形式，timestamp=%。。。

	req, _ := http.NewRequest("GET", str, nil)
	req.Header.Add("Content-Type", "application/json;charset=GB2312")
	timeout := time.Duration(6000 * time.Millisecond) //200毫秒超时
	client := http.Client{                            //为一个结构体，Timeout是里面一条属性
		Timeout: timeout,
	}
	response, err := client.Do(req)
	if err != nil {
		log("err:%v", err)
		return false
	}
	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		bodystr := string(body)
		log("respons bodystr:[%s]", bodystr)
		if strings.Split(strings.Split(bodystr, "&")[0], "=")[1] == "0" { //切割
			return true
		} else {
			return false
		}
	} else {
		fmt.Println(err)
		return false
	}
}

func UpdateUserProductInfo() {
	ticker := time.NewTicker(time.Hour * 4) //1h定时器
	go func() {
		for range ticker.C /*chan*/ {
			log("更新产品基本信息,当前时间:%s", TimeNowStr)
			sql := ps("select up.id,up.friendpdt_no from `user_product` as up,`enjoy_product` as ep where up.ep_id=ep.id and up.friendpdt_no!='' and ep.pt_id=1;")
			db := orm.NewOrm()
			var result []orm.Params
			_, err := db.Raw(sql).Values(&result)
			if err != nil {
				log("查询产品失败:[%v]", err)
			} else {
				// 更新产品信息
				var sqls []string
				for idx := range result {
					item := result[idx]
					alias := item["friendpdt_no"].(string)
					id, _ := strconv.Atoi(item["id"].(string))

					// 请求产品信息
					client := &http.Client{}
					strval := ps("alias=%s&i=/v1/charging/all&rf=3&ts=%d&v=1.0.1", alias, time.Now().Unix())
					strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
					strsign := StrToMD5(strpara)
					strval += ps("&encry=%s", strsign)
					req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/all", strings.NewReader(strval))
					if err != nil {
						log("创建HttpRequest失败:%s", err.Error())
						return
					}

					req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
					resp, err := client.Do(req)
					if err != nil {
						log("请求失败:%s", err.Error())
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode == 200 {
						body, _ := ioutil.ReadAll(resp.Body)
						//解析JSON数据
						type ProductInfo struct {
							Code int `json:"code"`
							Data []struct {
								Alias           string `json:"alias"`
								LastUseTime     string `json:"last_use_time"`
								Status          string `json:"status"`
								Latitude        string `json:"latitude"`
								Longitude       string `json:"longitude"`
								SpecificAddress string `json:"specific_address"`
								ProvinceName    string `json:"province_name"`
								CityName        string `json:"city_name"`
								AreaName        string `json:"area_name"`
								PositionName    string `json:"position_name"`
								StoreName       string `json:"store_name"`
							} `json:"data"`
						}
						var pi ProductInfo
						err = json.Unmarshal(body, &pi)
						if err != nil {
							log("解析json数据失败:[%s]", err.Error())
							continue
						}
						if len(pi.Data) > 0 {
							sqls = append(sqls, ps("update `user_product` set last_use_time='%s',friend_status='%s',latitude='%s',longitude='%s',specific_address='%s',province_name='%s',city_name='%s',area_name='%s',position_name='%s',store_name='%s' where id=%d;",
								pi.Data[0].LastUseTime, pi.Data[0].Status, pi.Data[0].Latitude, pi.Data[0].Longitude, pi.Data[0].SpecificAddress, pi.Data[0].ProvinceName, pi.Data[0].CityName, pi.Data[0].AreaName, pi.Data[0].PositionName, pi.Data[0].StoreName, id))
						}
					} else {
						log("http状态错误:%v", resp.StatusCode)
					}
				}

				// 更新数据
				db.Begin() //开启事务
				for _, ele := range sqls {
					_, err := db.Raw(ele).Exec()
					if err != nil {
						log("更新产品信息失败:[%v]", err)
					}
				}
				db.Commit() //提交事务
			}
		}
	}()
}

func FindRateForUnix(upunix int, pt_id int) (opeRatio, userRatio, taxRatio, platRatio float64) {
	month := (int(TimeNow)-upunix)/(30*24*60*60) + 1
	for i := range CdbRate {
		if CdbRate[i].Ptid == pt_id && month >= CdbRate[i].BeginMonth && month < CdbRate[i].EndMonth {
			opeRatio = CdbRate[i].OpeRatio
			userRatio = CdbRate[i].UserRatio
			taxRatio = CdbRate[i].TaxRatio
			platRatio = CdbRate[i].PlatRatio
			return
		}
	}

	return 0.0, 0.0, 0.0, 0.0
}

func UpdateProductUserInfo() {
	ticker := time.NewTicker(time.Hour * 1) //1h定时器
	go func() {
		for range ticker.C /*chan*/ {
			if time.Now().Format("15") == "00" { //进入下一天,更新所有产品前一天使用信息
				log("更新产品基本信息并统计近三天使用信息,计算收益:%s", TimeNowStr)
				sql := ps("select up.id,up.user_id,up.friendpdt_no,up.unix from `user_product` as up,`enjoy_product` as ep where up.ep_id=ep.id and up.friendpdt_no!='' and up.friend_status=1 and ep.pt_id=1;")
				db := orm.NewOrm()
				var result []orm.Params
				_, err := db.Raw(sql).Values(&result)
				if err != nil {
					log("查询使用中产品信息失败:[%v]", err)
				} else {
					// 更新产品信息
					var sqls []string
					for idx := range result {
						item := result[idx]
						alias := item["friendpdt_no"].(string)
						id, _ := strconv.Atoi(item["id"].(string))
						user_id, _ := strconv.Atoi(item["user_id"].(string))
						upunix, _ := strconv.Atoi(item["unix"].(string))

						// 请求产品使用信息
						client := &http.Client{}
						strval := ps("alias=%s&i=/v1/charging/device_data&rf=3&ts=%d&v=1.0.1", alias, time.Now().Unix())
						strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
						strsign := StrToMD5(strpara)
						strval += ps("&encry=%s", strsign)
						req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/device_data", strings.NewReader(strval)) // 指定设备每日使用数据,包含今日
						if err != nil {
							log("创建HttpRequest失败:%s", err.Error())
							return
						}

						req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
						resp, err := client.Do(req)
						if err != nil {
							log("请求产品使用信息失败:%s", err.Error())
							return
						}
						defer resp.Body.Close()

						if resp.StatusCode == 200 {
							body, _ := ioutil.ReadAll(resp.Body)

							//解析JSON数据
							//log("%s", string(body))
							type HistoryUseinfo struct {
								Code int `json:"code"`
								Data []struct {
									GrowthFansNum string `json:"growth_fans_num"`
									UseNum        string `json:"use_num"`
									PayNum        string `json:"pay_num"`
									Date          string `json:"date"`
								} `json:"data"`
							}
							var hui HistoryUseinfo
							err = json.Unmarshal(body, &hui)
							if err != nil {
								log("解析json数据失败:[%s]", err.Error())
								continue
							}

							// 每个产品更新最新的三条数据
							totals := len(hui.Data)
							if totals <= 0 { // 无使用记录
								continue
							}
							cnts := 3
							if cnts > totals {
								cnts = totals
							}
							for i := 0; i < cnts; i++ {
								sql := ps("select id from `product_use` where up_id=%d and date='%s';", id, hui.Data[totals-cnts+i].Date)
								var res []orm.Params
								nums, err := db.Raw(sql).Values(&res)
								if err != nil {
									log("读取产品使用信息出错:%s", err.Error())
								} else {
									if nums > 0 {
										sqls = append(sqls, ps("update `product_use` set growth_fans_num='%s',use_num='%s',date='%s',pay_num='%s',unix=%d where id=%d;",
											hui.Data[totals-cnts+i].GrowthFansNum, hui.Data[totals-cnts+i].UseNum, hui.Data[totals-cnts+i].Date, hui.Data[totals-cnts+i].PayNum, TimeNow, id))
									} else {
										sqls = append(sqls, ps("insert into `product_use` (up_id,growth_fans_num,use_num,`date`,pay_num,unix) values('%d','%s','%s','%s','%s','%d')",
											id, hui.Data[totals-cnts+i].GrowthFansNum, hui.Data[totals-cnts+i].UseNum, hui.Data[totals-cnts+i].Date, hui.Data[totals-cnts+i].PayNum, TimeNow))
									}
								}
							}

							// 昨日收入统计
							yesterday_unix := time.Now().AddDate(0, 0, -1)
							strTm := time.Unix(yesterday_unix.Unix(), 0).Format("2006-01-02")
							for i := totals - 1; i >= 0; i-- {
								if strTm == hui.Data[i].Date {
									growthfansnum, _ := strconv.Atoi(hui.Data[i].GrowthFansNum)
									paysnum, _ := strconv.Atoi(hui.Data[i].PayNum)
									if paysnum > 0 {
										insertDB(int64(user_id), hui.Data[i].Date, paysnum, 1, upunix, id, 1)
									}
									if growthfansnum > 0 {
										insertDB(int64(user_id), hui.Data[i].Date, growthfansnum, 0.6, upunix, id, 0)
									}
									break
								}
							}
						} else {
							log("http状态错误:%v", resp.StatusCode)
						}
					}

					//更新数据
					//log("%s", sqls)
					db.Begin() //开启事务
					for _, ele := range sqls {
						_, err := db.Raw(ele).Exec()
						if err != nil {
							log("更新产品信息失败:[%v]", err)
						}
					}
					db.Commit() //提交事务
				}
			}

			// 每个一小时更新所有产品当天使用信息
			ChargingDate()
		}
	}()
}

func insertDB(user_id int64, td string, count int, proportion float64, upunix int, id int, ntype int) (err error) {
	today_t, _ := time.ParseInLocation("2006-01-02", td, time.Local)
	total_fee := float64(count) * proportion
	opeRatio, userRatio, taxRatio, platRatio := FindRateForUnix(upunix, 1)
	//log("总收益:%v 次数:%d 产品:%d 日期:%s", total_fee, count, id, td)
	if opeRatio > 0.0 && userRatio > 0.0 && taxRatio > 0.0 && platRatio > 0.0 {
		if opeRatio+userRatio == 1.0 {
			opera_income := total_fee * opeRatio                          // 运营商收益
			user_income := (total_fee - opera_income) * (1.00 - taxRatio) // 平台和用户总收益
			platform_income := user_income * platRatio                    // 平台收益
			owner_income := user_income - platform_income                 // 用户收益
			db := orm.NewOrm()
			sql := ps("insert into `income` (up_id,owner_income,platform_income,opera_income,total_fee,content,unix) values ('%d','%v','%v','%v','%v','%d','%d');",
				id, owner_income, platform_income, opera_income, total_fee, ntype, today_t.Unix())
			_, err = db.Raw(sql).Exec()
			if err != nil {
				log("收入记录失败:err[%v]", err)
			}

			// 更新用户资金, 资金流水
			sql = ps("update `user` set wallet=wallet+%v where id=%d;", owner_income, user_id)
			_, err := db.Raw(sql).Exec()
			if err != nil {
				log("收益结算更新用户资金失败:[%v]", err)
			}
			sql = ps("select wallet from `user` where id=%d;", user_id)
			var result []orm.Params
			_, err = db.Raw(sql).Values(&result)
			if err != nil {
				log("收益结算查询用户资金失败:[%v]", err)
			} else {
				if len(result) > 0 {
					if result[0]["wallet"] != nil {
						wallet, _ := strconv.ParseFloat(result[0]["wallet"].(string), 64)
						//插入资金变动流水
						AddMoneyFlow("用户收益", "", wallet, owner_income, user_id, "用户收益")
					} else {
						log("收益结算插入资金流水失败")
					}
				} else {
					log("收益结算插入资金流水失败")
				}
			}

		} else {
			log("产品[%d]此阶段[%d]收益比率不正确", id, upunix)
			err = errors.New(ps("产品[%d]此阶段[%d]收益比率不正确", id, upunix))
		}
	} else {
		log("产品[%d]未添加此阶段[%d]收益比率", id, upunix)
		err = errors.New(ps("产品[%d]未添加此阶段[%d]收益比率", id, upunix))
	}
	return
}

func ChargingAll() {
	client := &http.Client{}
	strval := ps("rf=3&ts=%d&v=1.0.1&i=/v1/charging/all", time.Now().Unix())
	strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
	strsign := StrToMD5(strpara)
	strval += ps("&encry=%s", strsign)
	req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/all", strings.NewReader(strval))
	if err != nil {
		log("创建HttpRequest失败:%s", err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		log("请求失败:%s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		bodystr := string(body)
		log("%s", bodystr)

	} else {
		log("http状态错误:%v", resp.StatusCode)
	}

	return
}

func ChargingDate() {
	log("更新产品当天使用信息,当前时间:%s", TimeNowStr)
	client := &http.Client{}
	strval := ps("date=%s&rf=3&ts=%d&v=1.0.1&i=/v1/charging/data", time.Now().Format("2006-01-02"), time.Now().Unix())
	strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
	strsign := StrToMD5(strpara)
	strval += ps("&encry=%s", strsign)
	req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/data", strings.NewReader(strval)) // 指定日期,所有设备使用数据
	if err != nil {
		log("创建HttpRequest失败:%s", err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		log("请求失败:%s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		type TodayUseinfo struct {
			Code int `json:"code"`
			Data []struct {
				Alias         string `json:"alias"`
				GrowthFansNum string `json:"growth_fans_num"`
				UseNum        string `json:"use_num"`
				PayNum        string `json:"pay_num"`
				Date          string `json:"date"`
			} `json:"data"`
		}

		var tui TodayUseinfo
		err = json.Unmarshal(body, &tui)
		if err != nil {
			log("解析json数据失败:[%s]", err.Error())
			return
		}
		// 更新数据库
		if tui.Code == 200 {
			var sqls []string
			db := orm.NewOrm()
			var result []orm.Params
			for i := 0; i < len(tui.Data); i++ {
				sql := ps("select id from `user_product` where `friendpdt_no`='%s';", tui.Data[i].Alias)
				nums, err := db.Raw(sql).Values(&result)
				if err != nil {
					log("读取产品信息出错:%s", err.Error())
				} else {
					if nums > 0 {
						var res []orm.Params
						sql := ps("select id from `product_use` where up_id=%s and date='%s';", result[0]["id"].(string), tui.Data[i].Date)
						nums, err := db.Raw(sql).Values(&res)
						if err != nil {
							log("读取产品使用信息出错:%s", err.Error())
						} else {
							if nums > 0 {
								sqls = append(sqls, ps("update `product_use` set growth_fans_num='%s',use_num='%s',pay_num='%s',date='%s',unix=%d where id=%s;",
									tui.Data[i].GrowthFansNum, tui.Data[i].UseNum, tui.Data[i].PayNum, tui.Data[i].Date, TimeNow, res[0]["id"].(string)))
							} else {
								sqls = append(sqls, ps("insert into `product_use` (up_id,growth_fans_num,use_num,pay_num,`date`,unix) values('%s','%s','%s','%s','%s','%d')",
									result[0]["id"].(string), tui.Data[i].GrowthFansNum, tui.Data[i].UseNum, tui.Data[i].PayNum, tui.Data[i].Date, TimeNow))
							}
						}
					}
				}
			}
			// 更新数据
			db.Begin() //开启事务
			for _, ele := range sqls {
				_, err := db.Raw(ele).Exec()
				if err != nil {
					log("更新产品使用信息失败:[%v][sql:%s]", err, ele)
				}
			}
			db.Commit() //提交事务
		}
	} else {
		log("http状态错误:%v", resp.StatusCode)
	}

	return
}

// 处理未支付订单
func MinuteTimerInit() {
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C /*chan*/ {
			db := orm.NewOrm()

			// 删除所有已过期订单
			sql := ps("delete from `enjoy_product` where pay_deadline<%d and pay_status=0;", TimeNow+58)
			_, err := db.Raw(sql).Exec()
			if err != nil {
				log("删除过期订单失败:[%v]", err)
				break
			}
			log("过期未支付订单处理完成")
		}
	}()

	return
}

func PayDetails(alias, start_date, end_date string) (rcode int, data interface{}) {
	// 请求产品信息
	rcode = 201
	client := &http.Client{}
	strval := ps("alias=%s&start_date=%s&end_date=%s&i=/v1/charging/pay_details&rf=3&ts=%d&v=1.0.1", alias, start_date, end_date, time.Now().Unix())
	strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
	strsign := StrToMD5(strpara)
	strval += ps("&encry=%s", strsign)
	req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/pay_details", strings.NewReader(strval))
	if err != nil {
		log("创建HttpRequest失败:%s", err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		log("请求失败:%s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)

		//解析JSON数据
		type RetData struct {
			Code int         `json:"code"`
			Data interface{} `json:"data"`
		}
		var rd RetData
		err = json.Unmarshal(body, &rd)
		if err != nil {
			log("解析json数据失败:[%s]", err.Error())
			return
		} else {
			rcode = rd.Code
			data = rd.Data
		}
	}
	return
}

// 微信服务
func GetOpenidForCode(code string) (openid string, errmsg string) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		conf("wx_appid"), conf("wx_secret"), code)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json;charset=GB2312")

	timeout := time.Duration(6000 * time.Millisecond) // 设置超时
	client := http.Client{                            // 为一个结构体，Timeout是里面一条属性
		Timeout: timeout,
	}

	openid = ""
	errmsg = ""
	response, err := client.Do(req)
	if err != nil {
		log("get请求失败:[%s]", err.Error())
		return
	}

	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		bodystr := string(body)

		if strings.Contains(bodystr, "errcode") { //错误信息
			type ErrData struct {
				Errcode int64  `json:"errcode"`
				Errmsg  string `json:"errmsg"`
			}

			var data ErrData
			err = json.Unmarshal([]byte(bodystr), &data)
			if err == nil {
				errmsg = data.Errmsg
				if data.Errcode == 40001 { //access_token is invalid
					GetAccessToken()
				}
			} else {
				log("解析数据出错:[%s]", err.Error())
			}
		} else {
			type RecvData struct {
				AccessToken  string `json:"access_token"`
				ExpiresIn    int64  `json:"expires_in"`
				RefreshToken string `json:"refresh_token"`
				Openid       string `json:"openid"`
				Scope        string `json:"scope"`
			}

			var data RecvData
			err = json.Unmarshal([]byte(bodystr), &data)
			if err == nil {
				log("openid:[%s]", data.Openid)
				openid = data.Openid
			} else {
				log("解析数据出错:[%s]", err.Error())
			}
		}
	} else {
		log("get请求状态错误:[%s]", err.Error())
	}
	return
}

func GetWxUserInfoForOpenid(openid string) (user *Loginuser, errmsg string) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN", Access_Token, openid)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json;charset=GB2312")

	timeout := time.Duration(6000 * time.Millisecond) // 设置超时
	client := http.Client{                            // 为一个结构体，Timeout是里面一条属性
		Timeout: timeout,
	}

	errmsg = ""
	response, err := client.Do(req)
	if err != nil {
		log("get请求失败:[%s]", err.Error())
		return
	}

	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		bodystr := string(body)

		if strings.Contains(bodystr, "errcode") { //错误信息
			type ErrData struct {
				Errcode int64  `json:"errcode"`
				Errmsg  string `json:"errmsg"`
			}
			var data ErrData
			err = json.Unmarshal([]byte(bodystr), &data)
			if err == nil {
				errmsg = data.Errmsg
				if data.Errcode == 40001 { //access_token is invalid
					GetAccessToken()
				}
			} else {
				log("解析数据出错:[%s]", err.Error())
			}
		} else {
			log("%s", bodystr)
			// type RecvData struct {
			// 	Subscribe     int64  `json:"subscribe"`
			// 	Openid        string `json:"openid"`
			// 	Nickname      string `json:"nickname"`
			// 	Sex           int64  `json:"sex"`
			// 	City          string `json:"city"`
			// 	Province      string `json:"province"`
			// 	Country       string `json:"country"`
			// 	Headimgurl    string `json:"headimgurl"`
			// 	SubscribeTime int64  `json:"subscribe_time"`
			// 	Remark        string `json:"remark"`
			// 	Groupid       int64  `json:"groupid"`
			// 	Tagid         interface{}
			// }
			// var data RecvData
			// json.Unmarshal([]byte(bodystr), &data)
		}
	} else {
		log("get请求状态错误:[%s]", err.Error())
	}
	return
}

func WxTokenInit() {
	var sql string = "SELECT * from wx_token WHERE id=1;"
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("[查询数据库失败:err[%v]", err)
	} else {
		if len(result) > 0 {
			expires_unix, _ := strconv.Atoi(result[0]["expires_unix"].(string))
			if int64(expires_unix) > TimeNow {
				Access_Token = result[0]["access_token"].(string)
				Token_Expires = int64(expires_unix)
				return
			}
		}
	}

	// 请求微信
	GetAccessToken()
}

func GetAccessToken() {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", conf("wx_appid"), conf("wx_secret"))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json;charset=GB2312")

	timeout := time.Duration(6000 * time.Millisecond) // 设置超时
	client := http.Client{                            // 为一个结构体，Timeout是里面一条属性
		Timeout: timeout,
	}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		bodystr := string(body)

		if strings.Contains(bodystr, "errcode") { //错误信息
			log("错误信息:%s", bodystr)
		} else {
			type WxToken struct {
				AccessToken string `json:"access_token"`
				ExpiresIn   int64  `json:"expires_in"`
			}
			var wtk WxToken
			err = json.Unmarshal([]byte(bodystr), &wtk)
			if err == nil {
				Access_Token = wtk.AccessToken
				Token_Expires = TimeNow + wtk.ExpiresIn

				// 存储数据库
				var sql string = "SELECT * from wx_token WHERE id=1;"
				db := orm.NewOrm()
				var result []orm.Params
				_, err := db.Raw(sql).Values(&result)
				if err != nil {
					fmt.Println(err)
				} else {
					if len(result) > 0 {
						sql = ps("update wx_token set access_token='%s',expires_unix=%d where id=1;", Access_Token, Token_Expires)
					} else {
						sql = ps("insert into wx_token(access_token,expires_unix) values('%s','%d');", Access_Token, Token_Expires)
					}
					_, err = db.Raw(sql).Exec()
					if err != nil {
						log("更新token失败:err[%v]", err)
					}
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		fmt.Println(err)
	}
}

func WxJsapiInit() {
	var sql string = "SELECT * from wx_token WHERE id=1;"
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("[查询数据库失败:err[%v]", err)
	} else {
		if len(result) > 0 {
			ticket_expire, _ := strconv.Atoi(result[0]["ticket_expire"].(string))
			if int64(ticket_expire) > TimeNow {
				Jsapi_Ticket = result[0]["jsapi_ticket"].(string)
				Ticket_Expires = int64(ticket_expire)
				return
			}
		}
	}

	// 请求Jsapi
	GetJsapiTicket()
}

func GetJsapiTicket() {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi", Access_Token)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json;charset=GB2312")

	timeout := time.Duration(6000 * time.Millisecond) // 设置超时
	client := http.Client{                            // 为一个结构体，Timeout是里面一条属性
		Timeout: timeout,
	}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		bodystr := string(body)

		type WxJsapi struct {
			Errcode    int64  `json:"errcode"`
			Errmsg     string `json:"errmsg"`
			Ticket     string `json:"ticket"`
			Expires_in int64  `json:"expires_in"`
		}
		var wja WxJsapi
		err = json.Unmarshal([]byte(bodystr), &wja)
		if err == nil {
			if wja.Errcode == 0 {
				Jsapi_Ticket = wja.Ticket
				Ticket_Expires = TimeNow + wja.Expires_in
				// 存储数据库
				var sql string = "SELECT * from wx_token WHERE id=1;"
				db := orm.NewOrm()
				var result []orm.Params
				_, err := db.Raw(sql).Values(&result)
				if err != nil {
					fmt.Println(err)
				} else {
					if len(result) > 0 {
						sql = ps("update wx_token set jsapi_ticket='%s',ticket_expire=%d where id=1;", Jsapi_Ticket, Ticket_Expires)
					} else {
						sql = ps("insert into wx_token(jsapi_ticket,ticket_expire) values('%s','%d');", Jsapi_Ticket, Ticket_Expires)
					}
					_, err = db.Raw(sql).Exec()
					if err != nil {
						log("更新jsapi失败:err[%v]", err)
					}
				}
			} else {
				fmt.Println("获取jsapi错误: " + wja.Errmsg)
			}
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Println(err)
	}
}

//获取微信登录用户信息
func GetWxLoginUserInfo(code string, platform int64) (data *WXUserInfo, errmsg string) {
	type AcessTokenData struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Openid       string `json:"openid"`
		Scope        string `json:"scope"`
	}
	var tokenData AcessTokenData
	var wx_appid, wx_secret string

	if platform == 1 || platform == 2 {
		platform = 1
	} else {
		platform = 2
	}

	if platform == 1 { //移动端
		wx_appid = conf("wx_phone_appid")
		wx_secret = conf("wx_phone_secret")
	} else if platform == 2 { //网站
		wx_appid = conf("wx_appid")
		wx_secret = conf("wx_secret")
	} else {
		wx_appid = conf("wx_phone_appid")
		wx_secret = conf("wx_phone_secret")
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		wx_appid, wx_secret, code)

	log("get请求acess_token的URL:[%s]", url)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json;charset=GB2312")

	timeout := time.Duration(6000 * time.Millisecond) // 设置超时
	client := http.Client{                            // 为一个结构体，Timeout是里面一条属性
		Timeout: timeout,
	}

	errmsg = ""
	response, err := client.Do(req)
	if err != nil {
		log("get请求acess_token失败:[%s]", err.Error())
		errmsg = err.Error()
		return
	}

	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		bodystr := string(body)

		//log("get请求acess_token获取到的数据:[%s]", bodystr)
		if strings.Contains(bodystr, "errcode") { //错误信息
			type ErrData struct {
				Errcode int64  `json:"errcode"`
				Errmsg  string `json:"errmsg"`
			}

			var data ErrData
			err = json.Unmarshal([]byte(bodystr), &data)
			if err == nil {
				errmsg = data.Errmsg
			} else {
				log("解析acess_token数据出错:[%s]", err.Error())
			}

		} else {

			err = json.Unmarshal([]byte(bodystr), &tokenData)
			if err == nil {

				//请求微信用户信息
				url = fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
					tokenData.AccessToken, tokenData.Openid)

				req, _ = http.NewRequest("GET", url, nil)
				req.Header.Add("Content-Type", "application/json;charset=GB2312")

				response, err := client.Do(req)
				if err != nil {
					log("get请求微信用户信息失败:[%s]", err.Error())
					errmsg = err.Error()
					return
				}

				if response.StatusCode == 200 {
					body, _ := ioutil.ReadAll(response.Body)
					bodystr := string(body)

					if strings.Contains(bodystr, "errcode") { //错误信息
						type ErrData struct {
							Errcode int64  `json:"errcode"`
							Errmsg  string `json:"errmsg"`
						}
						var data ErrData
						err = json.Unmarshal([]byte(bodystr), &data)
						if err == nil {
							errmsg = data.Errmsg
						} else {
							log("解析微信用户信息数据出错:[%s]", err.Error())
						}
					} else {
						log("%s", bodystr)
						err = json.Unmarshal([]byte(bodystr), &data)
						if err != nil {
							errmsg = err.Error()
							log("解析微信用户信息数据出错:[%s]", err.Error())
						}
					}

				} else {
					log("get请求微信用户信息错误:[%s]", err.Error())
				}

			} else {
				log("解析acess_token数据出错:[%s]", err.Error())
			}
		}

	} else {
		log("get请求acess_token状态错误:[%s]", err.Error())
	}

	return
}

// 微信服务

//获取登录信息
func GetLoginUser(result []orm.Params, platform int32) (loginUser *Loginuser) {
	userid, _ := strconv.Atoi(result[0]["id"].(string))
	account := result[0]["account"].(string)
	wallet, _ := strconv.ParseFloat(result[0]["wallet"].(string), 64)

	canWallet := GetCanWallet(wallet, int64(userid), account)

	// 记录用户信息(添加到映射)
	id, _ := strconv.Atoi(result[0]["id"].(string))
	flag, _ := strconv.Atoi(result[0]["flag"].(string))
	level, _ := strconv.Atoi(result[0]["level"].(string))
	pt_id, _ := strconv.Atoi(result[0]["pt_id"].(string))
	supid, _ := strconv.Atoi(result[0]["supid"].(string))
	deduct, _ := strconv.ParseFloat(result[0]["deduct"].(string), 64)
	var nick, intro string
	if result[0]["nick"] == nil {
		nick = ""
	} else {
		nick = result[0]["nick"].(string)
	}
	if result[0]["intro"] == nil {
		intro = ""
	} else {
		intro = result[0]["intro"].(string)
	}

	loginUser = &Loginuser{
		SessionId:  GetSid(),
		UserId:     int64(id),
		Account:    result[0]["account"].(string),
		Nick:       nick,
		Intro:      intro,
		Flag:       int64(flag),
		Level:      int64(level),
		Wallet:     float64(wallet),
		CanWallet:  canWallet,
		Src:        result[0]["src"].(string),
		City:       result[0]["city"].(string),
		DealerAcc:  result[0]["dealer_acc"].(string),
		Ptid:       int32(pt_id),
		Phone:      result[0]["phone"].(string),
		Address:    result[0]["address"].(string),
		Realname:   result[0]["realname"].(string),
		Deduct:     deduct,
		Spreadlink: result[0]["spread_link"].(string),
		Qrcode:     result[0]["qr_code"].(string),
		Platform:   platform,
		LastTime:   TimeNow,
		Supid:      int64(supid),
		WxOpenid:   result[0]["wx_openid"].(string),
	}
	return
}

//查询用户可提余额
func GetCanWallet(wallet float64, userid int64, account string) (canWallet float64) {

	startDate := time.Now().Format("2006-01")
	t, _ := time.ParseInLocation("2006-01", startDate, time.Local)
	endUnix := t.Unix()
	sql := ps("select sum(amount) as nocount from money_flow as m where user_id='%d' and amount>0 and unix>'%d'", userid, endUnix)
	var noresult []orm.Params
	var nocount float64
	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&noresult)
	if err != nil {
		log("[%s]查询当月收入失败:err[%v]", account, err)
		canWallet = 0
	} else {

		if len(noresult) <= 0 || noresult[0]["nocount"] == nil {
			nocount = 0
			canWallet = wallet - nocount
		} else {
			nocount, err = strconv.ParseFloat(noresult[0]["nocount"].(string), 64)
			if nocount < wallet {
				canWallet = wallet - nocount
			} else {
				canWallet = 0
			}
		}

	}
	return
}
