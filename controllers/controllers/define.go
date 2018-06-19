package controllers

import (
	"crypto/md5"
	"encoding/json"
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
	SessionId string
	UserId    int64
	Account   string
	Nick      string
	Intro     string
	Flag      int64
	Level     int64
	Wallet    float64
	Src       string
	City      string
	DealerAcc string
	Ptid      int32
	Phone     string
	Address   string
	Realname  string
	LastTime  int64
	Platform  int32
	Auth      map[string]*Auth
}

type Auth struct {
	Id   string
	Name string
	Url  string
}

func init() {
	TimenowInit()
	MinuteTimerInit() // 分钟定时器
	ArgInit()
	MysqlInit()
	FileInit()
	OriginInit()
	AuthlistInit()

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
	sql := "select * from `profit_ratio` where `operate`=0 and pt_id=1;"
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
		CdbRate[idx].Ptid = 1
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
	beego.SetStaticPath("/cowin/carousel", conf("carouselpath"))
	beego.SetStaticPath("/cowin/mall", conf("mallpath"))
	beego.SetStaticPath("/cowin/dealer", conf("dealerpath"))
	beego.SetStaticPath("/cowin/personalpath", conf("personalpath"))

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

func FindRateForUnix(upunix int) (opeRatio, userRatio, taxRatio, platRatio float64) {
	month := (upunix - int(TimeNow)) / (30 * 24 * 60 * 60)
	// 判断是否超出了使用期限,超出一律按最后阶段算
	size := len(CdbRate)
	if size > 0 {
		if upunix >= CdbRate[size-1].EndMonth {
			opeRatio = CdbRate[size-1].OpeRatio
			userRatio = CdbRate[size-1].UserRatio
			taxRatio = CdbRate[size-1].TaxRatio
			platRatio = CdbRate[size-1].PlatRatio
			return
		}
	}

	for i := range CdbRate {
		if month >= CdbRate[i].BeginMonth && month < CdbRate[i].EndMonth {
			opeRatio = CdbRate[i].OpeRatio
			userRatio = CdbRate[i].UserRatio
			taxRatio = CdbRate[i].TaxRatio
			platRatio = CdbRate[i].PlatRatio
			break
		}
	}

	return 0.0, 0.0, 0.0, 0.0
}

func UpdateProductUserInfo() {
	ticker := time.NewTicker(time.Hour * 1) //1h定时器
	go func() {
		for range ticker.C /*chan*/ {
			if time.Now().Format("15") == "00" { //进入下一天,更新所有产品前一天使用信息
				sql := ps("select up.id,up.friendpdt_no,up.unix from `user_product` as up,`enjoy_product` as ep where up.ep_id=ep.id and up.friendpdt_no!='' and ep.pt_id=1;")
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
						upunix, _ := strconv.Atoi(item["unix"].(string))

						// 请求产品使用信息
						client := &http.Client{}
						strval := ps("alias=%s&i=/v1/charging/all&rf=3&ts=%d&v=1.0.1", alias, time.Now().Unix())
						strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
						strsign := StrToMD5(strpara)
						strval += ps("&encry=%s", strsign)
						req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/device_data", strings.NewReader(strval))
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
							type HistoryUseinfo struct {
								Code int `json:"code"`
								Data []struct {
									GrowthFansNum string `json:"growth_fans_num"`
									UseNum        string `json:"use_num"`
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
										sqls = append(sqls, ps("update `product_use` set growth_fans_num='%s',use_num='%s',date='%s',unix=%d where id=%d;",
											hui.Data[totals-cnts+i].GrowthFansNum, hui.Data[totals-cnts+i].UseNum, hui.Data[totals-cnts+i].Date, TimeNow, id))
									} else {
										sqls = append(sqls, ps("insert into `product_use` (up_id,growth_fans_num,use_num,`date`,unix) values('%d','%s','%s','%s','%d')",
											id, hui.Data[totals-cnts+i].GrowthFansNum, hui.Data[totals-cnts+i].UseNum, hui.Data[totals-cnts+i].Date, TimeNow))
									}
								}
							}

							// 收入统计(这里还要查询当前阶段利率)
							strTm := time.Now().Format("2006-01-02")
							if strTm == hui.Data[totals-1].Date {
								today_t, _ := time.ParseInLocation("2006-01-02", hui.Data[totals-1].Date, time.Local)
								growthfansnum, _ := strconv.Atoi(hui.Data[totals-1].GrowthFansNum)
								total_fee := float64(growthfansnum) * 0.5
								opeRatio, userRatio, taxRatio, platRatio := FindRateForUnix(upunix)
								log("[%d]收益比率%v %v %v %v", id, opeRatio, userRatio, taxRatio, platRatio)
								if opeRatio > 0.0 && userRatio > 0.0 && taxRatio > 0.0 && platRatio > 0.0 {
									if opeRatio+userRatio == 1.0 {
										opera_income := total_fee * opeRatio
										user_income := total_fee * userRatio * (1.00 - 0.06)
										platform_income := user_income * platRatio
										owner_income := user_income - platform_income
										sql := ps("insert into `income` (up_id,owner_income,platform_income,opera_income,total_fee,unix) values ('%d','%v','%v','%v','%v','%d');",
											id, owner_income, platform_income, opera_income, total_fee, today_t.Unix())
										_, err := db.Raw(sql).Exec()
										if err != nil {
											log("收入记录失败:err[%v]", err)
										}
									} else {
										log("产品[%d]此阶段[%d]收益比率不正确", id, upunix)
									}
								} else {
									log("产品[%d]未添加此阶段[%d]收益比率", id, upunix)
								}
							}

						} else {
							log("http状态错误:%v", resp.StatusCode)
						}
					}

					//更新数据
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
	client := &http.Client{}
	strval := ps("date=%s&rf=3&ts=%d&v=1.0.1&i=/v1/charging/data", time.Now().Format("2006-01-02"), time.Now().Unix())
	strpara := strval + "0q238ie8347fj3659fh$&HF^IE812*(23z7^&*12ksjSKW0"
	strsign := StrToMD5(strpara)
	strval += ps("&encry=%s", strsign)
	req, err := http.NewRequest("POST", "https://www.wacdd.com/external/v1/charging/data", strings.NewReader(strval))
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
				Date          string `json:"date"`
			} `json:"data"`
		}
		//bodystr := string(body)
		//log("data :%s", bodystr)

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
								sqls = append(sqls, ps("update `product_use` set growth_fans_num='%s',use_num='%s',date='%s',unix=%d where id=%s;",
									tui.Data[i].GrowthFansNum, tui.Data[i].UseNum, tui.Data[i].Date, TimeNow, res[0]["id"].(string)))
							} else {
								sqls = append(sqls, ps("insert into `product_use` (up_id,growth_fans_num,use_num,`date`,unix) values('%s','%s','%s','%s','%d')",
									result[0]["id"].(string), tui.Data[i].GrowthFansNum, tui.Data[i].UseNum, tui.Data[i].Date, TimeNow))
							}
						}
					}
				}
			}
			//log("%s", sqls)
			// 更新数据
			db.Begin() //开启事务
			for _, ele := range sqls {
				_, err := db.Raw(ele).Exec()
				if err != nil {
					log("更新产品使用信息失败:[%v]", err)
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
