package controllers

import (
	"crypto/sha1"
	"encoding/json"
	"github.com/astaxie/beego/orm"
	"io"
	"io/ioutil"
	//"math/rand"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	//"time"
)

type UserLoginController struct {
	OnlineController
}

type LoginController struct {
	BaseController
}

func (this *LoginController) SendVerCode() {
	phone := this.GetString("phone")
	/***************参数校验********************/
	if CheckArg(phone) {
		reg := `[0-9]`
		var rgx *regexp.Regexp = regexp.MustCompile(reg)
		if !rgx.MatchString(phone) {
			this.Rec = &Recv{5, ps("[%s]请输入正确的手机号", phone), nil}
			return
		}
	} else {
		this.Rec = &Recv{5, "手机号码不能为空", nil}
		return
	}

	// 发送验证码
	//生成6位数字验证码
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// vcode := new(vcode_t)
	// vcode.code = ps("%06v", r.Int31n(999999))
	// vcode.lasttime = TimeNow
	// VerCodes[phone] = vcode //存储信息

	// if SendMsg(phone, ps("尊敬的客户，您的手机验证码为：%s，本验证码5分钟之内有效。请保证是本人使用，否则请忽略此短信", vcode.code)) {
	// 	log("用户[%s]验证码是[%s]", phone, vcode.code)
	// 	this.Rec = &Recv{3, ps("[%s]验证码发送成功", phone), nil}
	// } else {
	// 	this.Rec = &Recv{5, ps("[%s]验证码发送失败", phone), nil}
	// }

	client := &http.Client{}
	strval := ps("templateid=3077019&mobile=%s", phone)
	req, err := http.NewRequest("POST", "https://api.netease.im/sms/sendcode.action", strings.NewReader(strval))
	if err != nil {
		this.Rec = &Recv{5, "发送验证码失败", nil}
		return
	}

	strTime := ps("%d", TimeNow)
	strKey := "6323e05596739704a0086af70b6b062f"
	strNonce := "cowin"
	strSecret := "e7a3a9f802f5"
	tsh := sha1.New()
	tmpCheckSum := strSecret + strNonce + strTime
	io.WriteString(tsh, tmpCheckSum)
	strCheckNum := ps("%x", tsh.Sum(nil))

	req.Header.Set("AppKey", strKey)
	req.Header.Set("CurTime", strTime)
	req.Header.Set("CheckSum", strCheckNum)
	req.Header.Set("Nonce", strNonce)
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
		//log("%s", bodystr)
		type VerifyCode struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Obj  string `json:"obj"`
		}
		var vcs VerifyCode
		err = json.Unmarshal([]byte(bodystr), &vcs)

		if err == nil {
			if vcs.Code == 200 {
				vcode := new(vcode_t)
				vcode.code = vcs.Obj
				vcode.lasttime = TimeNow
				VerCodes[phone] = vcode //存储信息
				this.Rec = &Recv{3, "验证码已发送至你手机,请注意查收.", nil}
			} else {
				log("%s", bodystr)
				this.Rec = &Recv{5, ps("验证码发送失败:%s", vcs.Msg), nil}
			}
		} else {
			this.Rec = &Recv{5, ps("验证码发送失败:%v", err), nil}
		}

	} else {
		this.Rec = &Recv{5, "发送验证码失败", nil}
	}

	return
}

// account,pwd,vcode(验证码),nick(昵称,可不传),dealer_acc(经销商编号)
func (this *LoginController) Register() {
	account := this.GetString("account")
	pwd := this.GetString("pwd")
	vcode := this.GetString("vcode")
	nick := this.GetString("nick")
	dealer_acc := this.GetString("dealer_acc")

	// 判断经销商编号
	// log("%s", dealer_acc)
	db := orm.NewOrm()
	var sql string
	var result []orm.Params
	if dealer_acc == "" {
		dealer_acc = conf("platnumb")
	} else {
		sql = ps("select id from `user` where `account`='%s';", dealer_acc)
		cnts, err := db.Raw(sql).Values(&result)
		if err != nil {
			log("查询user表失败:[%v]", err)
			this.Rec = &Recv{5, "注册失败,校验推荐码失败.", nil}
			return
		} else if cnts <= 0 {
			this.Rec = &Recv{5, "你填写的推荐码不存在,请检查.", nil}
			return
		}
	}

	/***************参数校验********************/
	if CheckArg(account, pwd, vcode) {
		reg := `[0-9]`
		rgx := regexp.MustCompile(reg)
		if !rgx.MatchString(account) {
			this.Rec = &Recv{5, ps("[%s]请输入正确的手机号", account), nil}
			return
		}
		/*验证码校验*/
		if val, ok := VerCodes[account]; ok {
			if val.code == vcode {
				if val.lasttime-TimeNow > 300 {
					this.Rec = &Recv{5, ps("[%s]验证码超时", vcode), nil}
					return
				}
			} else {
				this.Rec = &Recv{5, ps("[%s]验证码不正确", vcode), nil}
				return
			}
		} else {
			this.Rec = &Recv{5, ps("[%s]验证码不存在", vcode), nil}
			return
		}
	} else {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}
	if strings.Contains(nick, "'") {
		this.Rec = &Recv{5, "昵称不能包含单引号", nil}
		return
	}

	// 密码转为MD5
	pwd = StrToMD5(ps("%s_Cowin_%s", account, pwd))

	/***************开始业务逻辑****************/
	sql = ps("SELECT * from user WHERE account = '%s'", account)
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("[%s]查询数据库失败:err[%v]", account, err)
		this.Rec = &Recv{5, ps("[%s]注册失败,请重新尝试或联系客服", account), nil}
		return
	} else if len(result) > 0 {
		this.Rec = &Recv{5, ps("[%s]用户已注册,请登陆或找回密码", account), nil}
		return
	}

	if nick != "" {
		sql = ps("insert into user(account,pwd,nick,dealer_acc,unix) values('%s','%s','%s','%s','%d');", account, pwd, nick, dealer_acc, TimeNow)
	} else {
		sql = ps("insert into user(account,pwd,dealer_acc,unix) values('%s','%s','%s','%d');", account, pwd, dealer_acc, TimeNow)
	}

	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("[%s]注册失败,插入数据库失败[%s]", account, sql)
		this.Rec = &Recv{5, ps("[%s]注册失败,请重新尝试或联系客服", account), nil}
		return
	}
	log("[%s]注册成功", account)
	delete(VerCodes, account)
	this.Rec = &Recv{3, ps("[%s]注册成功!", account), nil}
	return
}

func (this *LoginController) Login() {
	account := this.GetString("account")
	pwd := this.GetString("pwd")
	platform, _ := this.GetInt32("platform")
	/***************参数校验********************/
	if CheckArg(account, pwd, platform) {
		reg := `[0-9]`
		rgx := regexp.MustCompile(reg)
		if !rgx.MatchString(account) {
			this.Rec = &Recv{5, ps("[%s]请输入正确的账号", account), nil}
			return
		}
	} else {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}
	pwd = StrToMD5(ps("%s_Cowin_%s", account, pwd))

	/***************开始业务逻辑****************/
	var sql string = ps("SELECT * from user WHERE account = '%s'", account)
	db := orm.NewOrm()
	var result []orm.Params
	num, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("[%s]查询数据库失败:err[%v]", account, err)
		this.Rec = &Recv{5, ps("[%s]登陆失败,请重新尝试或联系客服", account), nil}
		return
	} else if num == 0 {
		this.Rec = &Recv{5, ps("[%s]用户未注册,请注册后登陆", account), nil}
		return
	} else if result[0]["pwd"].(string) != pwd {
		this.Rec = &Recv{5, ps("[%s]用户密码错误请重新尝试", account), nil}
		return
	} else if result[0]["active"].(string) == "1" {
		this.Rec = &Recv{5, ps("[%s]账号被锁死,请联系客服解锁", account), nil}
		return
	}

	verify_deadline, _ := strconv.Atoi(result[0]["verify_deadline"].(string))
	verify_status, _ := strconv.Atoi(result[0]["verify_status"].(string))
	if verify_status == 3 && int64(verify_deadline) < TimeNow {
		verify_status = 4
	}

	sql = ps("update `user` set log_unix=%d,verify_status='%d' where account='%s';", TimeNow, verify_status, account)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("更新登录时间失败[%v]", err)
	}
	log("[%s]登陆成功", account)

	// 记录用户信息(添加到映射)
	id, _ := strconv.Atoi(result[0]["id"].(string))
	flag, _ := strconv.Atoi(result[0]["flag"].(string))
	level, _ := strconv.Atoi(result[0]["level"].(string))
	pt_id, _ := strconv.Atoi(result[0]["pt_id"].(string))
	wallet, _ := strconv.ParseFloat(result[0]["wallet"].(string), 64)
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

	var User *Loginuser
	User = &Loginuser{
		SessionId: GetSid(),
		UserId:    int64(id),
		Account:   result[0]["account"].(string),
		Nick:      nick,
		Intro:     intro,
		Flag:      int64(flag),
		Level:     int64(level),
		Wallet:    float64(wallet),
		Src:       result[0]["src"].(string),
		City:      result[0]["city"].(string),
		DealerAcc: result[0]["dealer_acc"].(string),
		Ptid:      int32(pt_id),
		Phone:     result[0]["phone"].(string),
		Address:   result[0]["address"].(string),
		Realname:  result[0]["realname"].(string),
		Platform:  platform,
		LastTime:  TimeNow,
	}

	//给用户分配权限:存储用户不具有的权限
	if User.Flag != 0 {
		sql = ps("SELECT url from auth where id NOT IN (SELECT aid FROM users_auth WHERE uid='%d');", id)
		_, err = db.Raw(sql).Values(&result)
		if err != nil {
			log("[%s]查询用户权限表失败:err[%v]", account, err)
		} else {
			User.Auth = make(map[string]*Auth)
			for _, value := range result {
				vurl := value["url"].(string)
				User.Auth[vurl] = Authlist[value["url"].(string)]
			}
		}
	}

	UserSessions.Adduser(User)
	log("platform: %d", User.Platform)
	if User.Flag == 0 {
		type TagUser struct {
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
		}

		var user *TagUser
		user = &TagUser{
			SessionId: User.SessionId,
			UserId:    User.UserId,
			Account:   User.Account,
			Nick:      User.Nick,
			Intro:     User.Intro,
			Flag:      User.Flag,
			Level:     User.Level,
			Wallet:    float64(wallet),
			Src:       User.Src,
			City:      User.City,
			DealerAcc: User.DealerAcc,
			Ptid:      User.Ptid,
			Phone:     User.Phone,
			Address:   User.Address,
			Realname:  User.Realname,
			LastTime:  User.LastTime,
		}
		this.Rec = &Recv{3, ps("[%s]登录成功!", account), user}
	} else {
		this.Rec = &Recv{3, ps("[%s]登录成功!", account), User}
	}
	return
}

func (this *LoginController) ResetPwd() {
	//account,newpwd,vcode
	account := this.GetString("account")
	newpwd := this.GetString("newpwd")
	vcode := this.GetString("vcode")
	/***************参数校验********************/
	db := orm.NewOrm()
	if CheckArg(account, newpwd, vcode) {
		reg := `[0-9]`
		rgx := regexp.MustCompile(reg)
		if !rgx.MatchString(account) {
			this.Rec = &Recv{5, ps("[%s]请输入正确的账号", account), nil}
			return
		}
		/*验证码校验===========================================start*/
		var sql string = ps("SELECT account from user WHERE account='%s';", account)
		var result []orm.Params
		num, err := db.Raw(sql).Values(&result)
		if err != nil {
			log("[%s]查询数据库失败:err[%v]", account, err)
			this.Rec = &Recv{5, ps("[%s]修改密码失败.", account), nil}
			return
		} else if num == 0 {
			this.Rec = &Recv{5, ps("[%s]用户不存在", account), nil}
			return
		}

		if val, ok := VerCodes[account]; ok {
			if val.code == vcode {
				if val.lasttime-TimeNow > 300 {
					this.Rec = &Recv{5, ps("[%s]验证码超时", vcode), nil}
					return
				} else {
					newpwd = StrToMD5(ps("%s_Cowin_%s", result[0]["account"], newpwd))
				}
			} else {
				this.Rec = &Recv{5, ps("[%s]验证码不正确", vcode), nil}
				return
			}
		} else {
			this.Rec = &Recv{5, ps("[%s]验证码不存在", vcode), nil}
			return
		}
		/*验证码校验===========================================end*/
	} else {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}
	_, err := db.Raw("update user set pwd=? where account=?;", newpwd, account).Exec()
	if err != nil {
		log("[%s]更新数据库失败", account)
		this.Rec = &Recv{5, ps("[%s]密码重置失败,请重新尝试或联系客服", account), nil}
		return
	}
	delete(VerCodes, account)
	this.Rec = &Recv{3, ps("[%s]密码重置成功", account), nil}
	return
}

// sid,oldpwd,newpwd
func (this *UserLoginController) ModifyPwd() {
	newpwd := this.GetString("newpwd")
	oldpwd := this.GetString("oldpwd")
	account := this.User.Account
	/***************参数校验********************/
	if !CheckArg(newpwd, oldpwd) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	pwd := StrToMD5(ps("%s_Cowin_%s", account, oldpwd))
	var sql string = ps("SELECT pwd from user WHERE account = '%s'", account)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询数据库失败:err[%v]", err)
		_, str := ChecSQLerr(err)
		this.Rec = &Recv{5, ps("查询数据失败[%s]", str), nil}
		return
	} else {
		if result[0]["pwd"].(string) != pwd {
			this.Rec = &Recv{5, "你提交的旧密码不对,请检查.", nil}
			return
		}
	}

	newpwd = StrToMD5(ps("%s_Cowin_%s", account, newpwd))
	_, err = db.Raw("update user set pwd=? where account =?;", newpwd, account).Exec()
	if err != nil {
		log("[%s]更新数据库失败", account)
		this.Rec = &Recv{5, ps("[%s]密码修改失败,请重新尝试或联系客服", account), nil}
		return
	}
	log("[%s]密码修改成功", account)
	this.Rec = &Recv{3, ps("[%s]密码修改成功", account), nil}
	return
}

// sid,nick,intro,city,file
func (this *UserLoginController) UpdateUserInfo() {
	nick := this.GetString("nick")
	intro := this.GetString("intro")
	city := this.GetString("city")
	account := this.User.Account
	if strings.Contains(nick, "'") {
		this.Rec = &Recv{5, "昵称不能包含单引号", nil}
		return
	}
	if strings.Contains(intro, "'") {
		this.Rec = &Recv{5, "简介不能包含单引号", nil}
		return
	}

	f, _, err := this.GetFile("file")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("head文件上传失败:err[%v]", err)
			this.Rec = &Recv{5, "上传头像文件失败,请重新尝试", nil}
		} else {
			// 保存位置在 static/head,没有文件夹要先创建
			filename := ps("head%d", this.User.UserId)
			err = this.SaveToFile("file", filepath.Join(conf("headpath"), filename))
			if err != nil {
				log("head文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "修改头像失败", nil}
			}
		}
	} else {
		log("上传文件为空")
	}

	// 开始业务逻辑
	if nick == "" && intro == "" && city == "" {
		this.Rec = &Recv{3, "修改信息成功", nil}
		return
	}

	var sql string = "update `user` set "
	if nick != "" {
		sql += ps("nick='%s',", nick)
	}
	if intro != "" {
		sql += ps("intro='%s',", intro)
	}
	if city != "" {
		sql += ps("city='%s',", city)
	}
	sql = sql[0 : len(sql)-1]
	sql += ps(" where account='%s';", account)

	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("更新数据库失败[%s]", sql)
		this.Rec = &Recv{5, ps("[%s]修改信息失败,请重新尝试或联系客服", account), nil}
		return
	}
	if nick != "" {
		this.User.Nick = nick
	}
	if intro != "" {
		this.User.Intro = intro
	}
	if city != "" {
		this.User.City = city
	}
	this.Rec = &Recv{3, ps("[%s]修改信息成功", account), this.User}
}

func (this *UserLoginController) Logout() {
	UserSessions.Deluser(this.User.SessionId)
	this.Rec = &Recv{3, ps("[%s]退出成功", this.User.Account), nil}
}
