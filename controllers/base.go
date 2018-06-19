package controllers

import (
	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2/bson"
	"sort"
	//"strconv"
	"strings"
)

type BaseController struct {
	beego.Controller
	Rec *Recv
}

type OnlineController struct {
	BaseController
	User *Loginuser
}

var AppKey string = "cmHsE0VMDXLcGBmaoepS&0b#WcVyH@c5"
var Version string = "1.0.0"

func GetSid() string {
	return ps("%x", string(bson.NewObjectId()))
}

func (this *BaseController) Prepare() {
	url := this.Ctx.Input.URL()
	if url != "/cowin/alipay/notice" && url != "/cowin/wxpay/notice" {
		var keys []string
		keys = make([]string, len(this.Ctx.Request.Form))
		idx := 0
		for key, _ := range this.Ctx.Request.Form {
			keys[idx] = key
			idx += 1
		}

		// 参数声明
		var (
			sign, clisign, strpara string
			ver                    string
		)

		// 检测必传参数
		// if _, ok := this.Ctx.Request.Form["ver"]; !ok {
		// 	this.Rec = &Recv{5, "签名错误:版本号未传", "no data"}
		// 	goto EXIT_BASEPREPARE
		// }

		// if _, ok := this.Ctx.Request.Form["ts"]; !ok {
		// 	this.Rec = &Recv{5, "签名错误:必传参数不全", "no data"}
		// 	goto EXIT_BASEPREPARE
		// }else {
		// 	strts := this.Ctx.Input.Query("ts")
		// 	ts, _ := strconv.Atoi(strts)
		// 	if ts-3*60 > int(TimeNow) || ts+3*60 < int(TimeNow) {
		// 		log("client:%v svr:%v", ts, TimeNow)
		// 		this.Rec = &Recv{5, "签名错误:请校准本地时间", "no data"}
		// 		goto EXIT_BASEPREPARE
		// 	}
		// }

		// 判断版本是否正确
		ver = this.Ctx.Input.Query("ver")
		if ver != Version {
			this.Rec = &Recv{5, "签名错误:版本不匹配", "no data"}
			goto EXIT_BASEPREPARE
		}

		// 按字典顺序排序并构造签名字符窜
		sort.Strings(keys)
		for _, val := range keys {
			if val != "sign" {
				strpara += ps("%s=%s&", val, this.Ctx.Input.Query(val))
			} else {
				clisign = this.Ctx.Input.Query(val)
			}
		}
		strpara = strpara[0 : len(strpara)-1]

		// md5签名
		sign = StrToMD5(strpara + "&key=" + AppKey)
		if sign != clisign {
			log("[%s] server:[%s]  client:[%s]", strpara+"&key="+AppKey, sign, clisign)
			this.Rec = &Recv{5, "签名错误", "no data"}
		}
	}
	// else {
	// 	log("wxpay: %v", this.Ctx.Request)
	// }

EXIT_BASEPREPARE:
	if this.Rec != nil {
		this.Data["json"] = this.Rec
		this.ServeJSON()
	}
}

func (this *BaseController) MoniSrv() {
	this.Rec = &Recv{3, "success", nil}
	return
}

// 请求预处理
func (this *OnlineController) Prepare() {
	// 参数声明
	var (
		sign, clisign, strpara, ver string
		sid                         string
		ok                          bool
	)
	var keys []string
	keys = make([]string, len(this.Ctx.Request.Form))
	idx := 0
	for key, _ := range this.Ctx.Request.Form {
		keys[idx] = key
		idx += 1
	}

	// 检测必传参数
	if _, ok := this.Ctx.Request.Form["ver"]; !ok {
		this.Rec = &Recv{5, "签名错误:版本号未传", "no data"}
		goto EXIT_ONLINEPREPARE
	}

	// if _, ok := this.Ctx.Request.Form["ts"]; !ok {
	// 	this.Rec = &Recv{5, "签名错误:必传参数不全", "no data"}
	// 	goto EXIT_ONLINEPREPARE
	// } else {
	// 	strts := this.Ctx.Input.Query("ts")
	// 	ts, _ := strconv.Atoi(strts)
	// 	if ts-3*60 > int(TimeNow) || ts+3*60 < int(TimeNow) {
	// 		log("client:%v svr:%v", ts, TimeNow)
	// 		this.Rec = &Recv{5, "签名错误:请校准本地时间", "no data"}
	// 		goto EXIT_ONLINEPREPARE
	// 	}
	// }

	// 判断版本是否正确
	ver = this.Ctx.Input.Query("ver")
	if ver != Version {
		this.Rec = &Recv{5, "签名错误:版本不匹配", "no data"}
		goto EXIT_ONLINEPREPARE
	}

	// 按字典顺序排序并构造签名字符窜
	sort.Strings(keys)
	for _, val := range keys {
		if val != "sign" {
			strpara += ps("%s=%s&", val, this.Ctx.Input.Query(val))
		} else {
			clisign = this.Ctx.Input.Query(val)
		}
	}
	strpara = strpara[0 : len(strpara)-1]
	// log("para:%s", strpara)
	// md5签名
	sign = StrToMD5(strpara + "&key=" + AppKey)
	if sign != clisign {
		log("[%s] server:[%s]  client:[%s]", strpara+"&key="+AppKey, sign, clisign)
		this.Rec = &Recv{5, "签名错误", "no data"}
		goto EXIT_ONLINEPREPARE
	}

	sid = this.GetString("sid")
	if sid == "" {
		this.Rec = &Recv{6, "sid不能为空", "no data"}
	} else if this.User, ok = UserSessions.QueryloginS(sid); !ok {
		this.Rec = &Recv{6, "登录状态已过期,请重新登陆", "no data"}
	} else {
		if TimeNow-this.User.LastTime > SidLife {
			this.Rec = &Recv{6, "登录超时,重新登陆", "no data"}
		} else {
			//this.User.LastTime = TimeNow //更新最新通信时间

			//权限校验
			url := this.Ctx.Input.URL()
			if _, ok = this.User.Auth[url]; ok {
				this.User.LastTime = TimeNow //更新最新通信时间
				if this.User.Flag > 0 {
					this.Rec = &Recv{6, "权限不够,请检查", "no data"}
				}
			} else {
				this.User.LastTime = TimeNow
			}
		}
	}

EXIT_ONLINEPREPARE:
	if this.Rec != nil {
		this.Data["json"] = this.Rec
		this.ServeJSON()
	}
}

// 给客户端回复数据
func (this *BaseController) Finish() {
	if this.Rec != nil {
		if this.Rec.Code != -1 {
			if this.Rec.Data == nil {
				this.Rec.Data = "no data"
			}
			this.Data["json"] = this.Rec
		} else {
			this.Data["json"] = this.Rec.Data
		}
		if this.Rec.Code != 6 && !strings.Contains(this.Rec.Msg, "签名错误") {
			this.ServeJSON()
		}
	}
}
