package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"regexp"
	"strconv"
)

type DealerController struct {
	OnlineController
}

// sid,flag(2-销售,3-代理商,4-个人代理),account(用户编号),nick(名称),phone(手机号),deduct(提成比例:0.2),pwd
func (this *DealerController) DealerAdd() {
	account := this.GetString("account")
	nick := this.GetString("nick")
	phone := this.GetString("phone")
	deduct, _ := this.GetFloat("deduct", 32)
	pwd := this.GetString("pwd")
	flag, _ := this.GetInt32("flag")

	//检查参数
	if !CheckArg(account, nick, phone, deduct, pwd) {
		this.Rec = &Recv{5, "此接口参数均不能为空", nil}
		return
	} else {
		reg := `[0-9]`
		rgx := regexp.MustCompile(reg)
		if !rgx.MatchString(phone) {
			this.Rec = &Recv{5, ps("[%s]请输入正确的手机号", phone), nil}
			return
		}
	}

	// 判断权限
	if this.User.Flag != 1 && this.User.Flag != 3 {
		this.Rec = &Recv{5, "无权添加此类用户", nil}
	}

	//业务逻辑
	pwd = StrToMD5(ps("%s_Cowin_%s", account, pwd))
	var supid int64 = 0 // 平台管理员添加的一切用户都属于平台管理账号,其上级id为0
	if this.User.Flag == 3 {
		supid = this.User.UserId
	}
	var sql string = ps("insert into `user` (account,nick,flag,supid,phone,deduct,pwd,unix) values ('%s','%s','%d','%d','%s','%v','%s','%d')",
		account, nick, flag, supid, phone, deduct, pwd, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加角色出错:[%v]", err)
		this.Rec = &Recv{5, ps("添加角色失败[%s]", err.Error()), nil}
		return
	}
	this.Rec = &Recv{3, "添加角色成功", phone}
}

// id,nick(名称),phone(手机号),deduct(提成比例:0.2),pwd,spread_link(推广链接),qr_code(二维码,图片),license(营业执照)
func (this *DealerController) DealerModify() {
	id, _ := this.GetInt64("id")
	nick := this.GetString("nick")
	phone := this.GetString("phone")
	deduct, _ := this.GetFloat("deduct", 32)
	pwd := this.GetString("pwd")
	spread_link := this.GetString("spread_link")

	// 图片参数
	var qr_code, license string
	f, h, err := this.GetFile("qr_code")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("文件上传失败:err[%v]", err)
		} else {
			// 保存位置在 static/dealer
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("qr_code", filepath.Join(conf("dealerpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				qr_code = ps("https://%s/%s;", conf("dealerdown"), filename)
			}
		}
	}

	f, h, err = this.GetFile("license")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("文件上传失败:err[%v]", err)
		} else {
			// 保存位置在 static/dealer
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("license", filepath.Join(conf("dealerpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				license = ps("https://%s/%s;", conf("dealerdown"), filename)
			}
		}
	}

	// 业务逻辑
	db := orm.NewOrm()
	var sql = "update `user` set "
	if nick != "" {
		sql += ps("nick='%s',", nick)
	}
	if phone != "" {
		sql += ps("phone='%s',", phone)
	}
	if deduct > 0.00 {
		sql += ps("deduct='%v',", deduct)
	}
	if pwd != "" {
		var result []orm.Params
		nums, err := db.Raw("select account from user where id=?", id).Values(&result)
		if err == nil && nums > 0 {
			pwd = StrToMD5(ps("%s_Cowin_%s", result[0]["account"].(string), pwd))
		}
		sql += ps("pwd='%s',", pwd)
	}
	if spread_link != "" {
		sql += ps("spread_link='%s',", spread_link)
	}
	if qr_code != "" {
		sql += ps("qr_code='%s',", qr_code)
	}
	if license != "" {
		sql += ps("license='%s',", license)
	}
	sql += ps("unix='%d' where id='%d';", TimeNow, id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("修改用户信息出错:[%v]", err)
		_, str := ChecSQLerr(err)
		this.Rec = &Recv{5, ps("修改失败:%s", str), nil}
		return
	}

	this.Rec = &Recv{3, "修改成功,该用户重新登录后生效.", nil}
	return
}

// sid,id
func (this *DealerController) DealerDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	if id == this.User.UserId {
		this.Rec = &Recv{5, "不能删除自己的账号", nil}
		return
	}

	// 检查是权限
	var sql string = ps("SELECT supid,account from `user` where id='%d';", id)
	db := orm.NewOrm()
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	var supid int = 0
	var account string = ""
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询待删除用户信息出错.", nil}
		return
	} else {
		if nums > 0 {
			supid, _ = strconv.Atoi(result[0]["supid"].(string))
			account = result[0]["account"].(string)
			if supid <= 0 {
				if this.User.Flag != 1 {
					this.Rec = &Recv{5, "你无权删除此用户", nil}
					return
				}
			} else {
				if this.User.UserId != int64(supid) {
					this.Rec = &Recv{5, "你无权删除此用户", nil}
					return
				}
			}
		} else {
			this.Rec = &Recv{5, "待删除角色不存在.", nil}
			return
		}
	}

	// 业务逻辑
	sql = ps("delete from `user` where id='%d';", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("删除角色[%d]出错:[%v]", id, err)
		this.Rec = &Recv{5, "删除角色失败", nil}
		return
	}

	// 删除的用户注销登录信息
	usr, ok := UserSessions.QueryloginA(account)
	if ok {
		UserSessions.Deluser(usr.SessionId)
	}

	this.Rec = &Recv{3, "删除角色成功", nil}
}

// sid,flag(0-全部,2-销售,3-代理商,4-个人代理)
func (this *DealerController) DealerQuery() {
	flag, _ := this.GetInt32("flag")

	// 业务逻辑
	var sql string = ""
	var supid int64 = 0
	if this.User.Flag == 3 {
		supid = this.User.UserId
	}
	if flag > 0 {
		sql = ps("SELECT * from `user` where flag=%d and supid=%d order by unix desc;", flag, supid)
	} else {
		sql = ps("SELECT * from `user` where flag>1 and supid=%d order by unix desc;", supid)
	}

	db := orm.NewOrm()
	var result, res []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询角色失败", nil}
		return
	}

	// 累计客户量
	for idx := range result {
		item := result[idx]
		sql = ps("SELECT id from `user` where dealer_acc='%s';", item["account"].(string))
		num, err := db.Raw(sql).Values(&res)
		if err == nil {
			item["customers"] = num
		} else {
			log("查询user表出错:[%v]", err)
		}
	}

	this.Rec = &Recv{3, "查询角色成功", result}
}
