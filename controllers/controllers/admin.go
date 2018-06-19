package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"regexp"
	"strconv"
)

type AdminController struct {
	OnlineController
}

type AdminBaseController struct {
	BaseController
}

// sid,phone,pwd,flag(1-管理员,2销售员)
func (this *AdminController) AddUser() {
	phone := this.GetString("phone")
	pwd := this.GetString("pwd")
	flag, _ := this.GetInt64("flag")

	//检查参数
	if !CheckArg(phone, pwd) {
		this.Rec = &Recv{5, "电话或密码不能为空", nil}
		return
	} else {
		reg := `[0-9]`
		rgx := regexp.MustCompile(reg)
		if !rgx.MatchString(phone) {
			this.Rec = &Recv{5, ps("[%s]请输入正确的手机号", phone), nil}
			return
		}
	}

	//业务逻辑
	pwd = StrToMD5(ps("%s_Cowin_%s", phone, pwd))
	var sql string = ps("insert into `user` (account,pwd,flag,unix) values ('%s','%s','%d','%d')", phone, pwd, flag, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加用户出错:[%v]", err)
		this.Rec = &Recv{5, ps("添加用户失败[%s]", err.Error()), nil}
		return
	}

	this.Rec = &Recv{3, "添加用户成功!", phone}
}

// sid,begidx,counts
func (this *AdminController) QueryUser() {
	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")

	// 检查参数
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	var sql, sqlc string = "", ""
	sql = ps("SELECT id,account,nick,intro,active,wallet,src,city,realname,idnumber,unix,log_unix from `user` order by unix desc limit %d,%d;", begidx, counts)
	sqlc = "SELECT count(id) AS num from `user` where flag>0;"

	db := orm.NewOrm()
	var result []orm.Params
	var total int = 0
	_, err := db.Raw(sqlc).Values(&result)
	if err == nil {
		total, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("[%s]查询user表出错:[%v]", this.User.Account, err)
		this.Rec = &Recv{5, ps("[%s]查询用户列表失败", this.User.Account), nil}
		return
	}

	for idx, _ := range result {
		result[idx]["headurl"] = ps("https://%s/head%s", conf("headdown"), result[idx]["id"].(string))
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	resex := RecvEx{total, result}
	this.Rec = &Recv{3, ps("[%s]查询用户列表成功!", this.User.Account), resex}
	return
}

// sid,uid,nick,img,intro
func (this *AdminController) ModifyUser() {
	uid, _ := this.GetInt64("uid")
	nick := this.GetString("nick")
	intro := this.GetString("intro")

	// 检查参数
	if !CheckArg(uid) {
		this.Rec = &Recv{5, "用户id不能为空", nil}
		return
	}

	f, _, err := this.GetFile("img")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("head文件传输失败:err[%v]", err)
			this.Rec = &Recv{5, "上传头像失败,请重新尝试", nil}
		} else {
			// 保存位置在 static/head,没有文件夹要先创建
			filename := ps("head%d", uid)
			err = this.SaveToFile("img", filepath.Join(conf("headpath"), filename))
			if err != nil {
				log("head文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "上传头像文件失败", nil}
			}
		}
	}

	// 业务逻辑
	var sql = "update `user` set "
	if nick != "" {
		sql += ps("nick='%s',", nick)
	}
	if intro != "" {
		sql += ps("intro='%s',", intro)
	}
	sql += ps("unix='%d' where id='%d';", TimeNow, uid)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("[%s]修改用户[%d]信息出错:[%v]", this.User.Account, uid, err)
		this.Rec = &Recv{5, "修改用户信息失败", nil}
		return
	}

	this.Rec = &Recv{3, "修改用户信息成功!", nil}
}

// sid,id
func (this *AdminController) DelUser() {
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
	var sql string = ps("SELECT flag,account from `user` where id='%d';", id)
	var account string = ""
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询待删除用户信息出错.", nil}
		return
	} else {
		if len(result) > 0 {
			account = result[0]["account"].(string)
			tmpflag, _ := strconv.Atoi(result[0]["flag"].(string))
			if int64(tmpflag) > this.User.Flag {
				this.Rec = &Recv{5, "无权删除比自己级别高的用户.", nil}
				return
			}
		}
	}

	// 业务逻辑
	sql = ps("delete from `user` where id='%d';", id)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("删除用户[%d]出错:[%v]", id, err)
		this.Rec = &Recv{5, "删除用户失败", nil}
		return
	}

	// 删除的用户注销登录信息
	usr, ok := UserSessions.QueryloginA(account)
	if ok {
		UserSessions.Deluser(usr.SessionId)
	}

	this.Rec = &Recv{3, ps("[%s]删除用户成功!", this.User.Account), nil}
}

// sid,id,level
func (this *AdminController) ModifyUserLevel() {
	id, _ := this.GetInt64("id")
	level, _ := this.GetInt64("level")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "用户id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("update `user` set level='%d' where id='%d';", level, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("[%s]修改用户[%d]等级出错:[%v]", this.User.Account, id, err)
		this.Rec = &Recv{5, "修改用户等级失败", nil}
		return
	}

	this.Rec = &Recv{3, "修改用户等级成功!", nil}
}

// sid,account
func (this *AdminController) SpecifyUser() {
	account := this.GetString("account")

	// 检查参数
	if !CheckArg(account) {
		this.Rec = &Recv{5, "账号不能为空", nil}
		return
	}

	var sql string = ps("SELECT id,account,nick,intro,level,flag,wallet,src,city,log_unix from `user` where account='%s';", account)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询出错:[%v]", err)
		this.Rec = &Recv{5, "查询用户列表失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询用户列表成功", result}
	return
}

// sid
func (this *AdminController) Authlist() {
	var sql string = "SELECT id,name,url from auth;"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("[%s]查询auth表出错:[%v]", this.User.Account, err)
		this.Rec = &Recv{5, ps("[%s]查询权限表失败", this.User.Account), nil}
		return
	}

	this.Rec = &Recv{3, ps("[%s]查询权限成功!", this.User.Account), result}
	return
}

// sid,account,flag(0-有,1-无)
func (this *AdminController) UserAuthQuery() {
	account := this.GetString("account")
	flag, _ := this.GetInt32("flag")
	if account == "" {
		this.Rec = &Recv{5, "账号不能为空.", nil}
		return
	}

	// 查询uid
	var uid string
	var sql string = ps("select id from `user` where account='%s';", account)
	var result []orm.Params
	db := orm.NewOrm()
	num, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询user表失败:err[%v]", err)
		return
	} else if num > 0 {
		uid = result[0]["id"].(string)
	}

	// 业务逻辑
	switch flag {
	case 0:
		sql = ps("SELECT b.id,b.name,b.url,a.uid from users_auth a,auth b where a.uid='%s' and a.aid=b.id;", uid)
	case 1:
		sql = ps("SELECT id,name,url from auth where id NOT IN (SELECT aid FROM users_auth WHERE uid='%s');", uid)
	}
	//log("%s", sql)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("[%s]查询权限表出错:err[%v]", this.User.Account, err)
		this.Rec = &Recv{5, ps("[%s]查询权限表失败", this.User.Account), nil}
		return
	}

	if flag == 1 {
		for idx, _ := range result {
			result[idx]["uid"] = uid
		}
	}

	this.Rec = &Recv{3, ps("[%s]查询权限成功!", this.User.Account), result}
}

// sid,uid,aid(权限id),account,aod(0-添加;1-删除)
func (this *AdminController) AuthUser() {
	var uid int64
	var aid, aod int32

	uid, _ = this.GetInt64("uid")
	aid, _ = this.GetInt32("aid")
	aod, _ = this.GetInt32("aod")
	account := this.GetString("account")

	if uid <= 0 || aid <= 0 || account == "" {
		log("%d %d %s", uid, aid, account)
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	// 判断权限
	var sql string = ps("select flag from `user` where id='%d';", uid)
	var result []orm.Params
	db := orm.NewOrm()
	num, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询user表失败:err[%v]", err)
	} else if num > 0 {
		flag, _ := strconv.Atoi(result[0]["flag"].(string))
		if int64(flag) > this.User.Flag {
			this.Rec = &Recv{5, ps("[%s]无法给比自己权限高的用户分配权限", this.User.Account), nil}
			return
		}
	}

	// 业务逻辑
	switch aod {
	case 0:
		sql = ps("insert into users_auth (uid,aid) values ('%d','%d');", uid, aid)
	case 1:
		sql = ps("delete from users_auth where uid='%d' and aid='%d';", uid, aid)
	}
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("[%s]操作权限表出错:[%v]", this.User.Account, err)
		this.Rec = &Recv{5, ps("[%s]授权失败,请检查", this.User.Account), nil}
		return
	}

	// 修改权限的用户注销登录信息,需重新登录
	usr, ok := UserSessions.QueryloginA(account)
	if ok {
		UserSessions.Deluser(usr.SessionId)
	}
	this.Rec = &Recv{3, ps("[%s]授权成功!", this.User.Account), nil}
}

// sid,account(账号,即手机号)
func (this *AdminController) UserAuthInfoBkQuery() {
	account := this.GetString("account")
	if !CheckArg(account) {
		this.Rec = &Recv{3, "查询账号不能为空", nil}
		return
	}

	sql := ps("select id,realname,idnumber,positive_img,negative_img from user where account='%s';", account)
	var result []orm.Params

	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询认证信息失败:%s", err.Error())
		this.Rec = &Recv{5, "查询认证信息失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询认证信息成功", result}
}

// sid,id,verify_status(1-通过,2-未通过),verify_reason(未通过时原因),verify_deadline(到期时间)
func (this *AdminController) UserAuthInfoCheck() {
	id, _ := this.GetInt64("id")
	verify_status, _ := this.GetInt32("verify_status")
	verify_reason := this.GetString("verify_reason")
	verify_deadline, _ := this.GetInt64("verify_deadline")

	if !CheckArg(id, verify_status) {
		this.Rec = &Recv{3, "id和状态不能为空", nil}
		return
	}

	if verify_status == 2 && verify_reason == "" {
		this.Rec = &Recv{3, "请填写不通过理由", nil}
		return
	}

	if verify_status == 1 && verify_deadline <= 0 {
		this.Rec = &Recv{3, "请填写过期日期", nil}
		return
	}

	var sql string
	db := orm.NewOrm()
	if verify_status == 1 {
		// 对于待认证的合同写入认证信息
		var result []orm.Params
		_, err := db.Raw("select realname,idnumber,positive_img,negative_img from `user` where id=?", id).Values(&result)
		if err != nil {
			log("查询认证信息失败:%s", err.Error())
			this.Rec = &Recv{5, "查询认证信息失败", nil}
			return
		}

		sql = ps("update `agreement` set text='%s',realname='%s',idnumber='%s',positive_img='%s',negative_img='%s',unix='%d',status=2 where status=1;",
			"", result[0]["realname"].(string), result[0]["idnumber"].(string), result[0]["positive_img"].(string), result[0]["negative_img"].(string), TimeNow)
		_, err = db.Raw(sql).Values(&result)
		if err != nil {
			log("签署待认证合同失败:%s", err.Error())
			this.Rec = &Recv{5, "签署待认证合同失败", nil}
			return
		}

		sql = ps("update `user` set verify_status='3',verify_deadline='%d' where id='%d';", verify_deadline, id)
	} else {
		sql = ps("update `user` set verify_status='2',verify_reason='%s' where id='%d';", verify_reason, id)
	}

	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("审核认证信息失败:%s", err.Error())
		this.Rec = &Recv{5, "审核认证信息失败", nil}
		return
	}

	// 添加通知信息
	if verify_status == 1 {
		sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", "你的实名认证信息已通过审核,请去查看详情.", id, TimeNow)
	} else {
		sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','0','%s','%d')", "通知", "你的实名认证信息审核未通过,请去查看详情.", id, TimeNow)
	}
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加通知失败:[%v]", err)
	}

	this.Rec = &Recv{3, "审核认证信息成功", nil}
}
