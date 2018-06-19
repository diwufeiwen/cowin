package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	var supid int64 = 0 // 查询平台的
	if this.User.Flag == 3 {
		supid = this.User.UserId
	}

	if flag > 0 {
		sql = ps("SELECT * from `user` where flag=%d and supid=%d order by unix desc;", flag, supid)
	} else {
		sql = ps("SELECT * from `user` where flag>1 and flag<5 and supid=%d order by unix desc;", supid)
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
	return
}

// 代理商累计佣金
func DealearCommission(user *Loginuser, bt int64, et int64) (money float64, err error) {
	db := orm.NewOrm()
	var sql, sqlm string = "", ""
	var result, res []orm.Params
	money = 0.0
	if user.Flag == 1 { //平台管理员
		// 1.查询一级代理商
		sql = "SELECT id,account,deduct FROM user WHERE flag=3 AND supid=0;"
		_, err := db.Raw(sql).Values(&result)
		if err != nil {
			log("查询代理商出错:[%v]", err)
			return money, err
		}

		for idx := range result {
			item := result[idx]
			deduct, _ := strconv.ParseFloat(item["deduct"].(string), 64) // 分成比率
			sqlm = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and (order_no LIKE '%s_%%'", item["account"].(string))

			// 查询二级代理商,销售员,个人代理
			uid, _ := strconv.Atoi(item["id"].(string))
			sql = ps("SELECT id,account,flag FROM `user` WHERE flag>1 and flag<5 AND supid=%d;", uid)
			_, err := db.Raw(sql).Values(&res)
			if err != nil {
				log("查询一级代理下属失败:[%v]", err)
				return money, err
			}

			for i := range res {
				sitem := res[i]
				sqlm += ps(" OR order_no LIKE '%s_%%'", sitem["account"].(string))

				flag, _ := strconv.Atoi(sitem["flag"].(string))
				if flag == 3 {
					// 查询二级代理销售
					suid, _ := strconv.Atoi(sitem["id"].(string))
					sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", suid)
					var sres []orm.Params
					_, err := db.Raw(sql).Values(&sres)
					if err != nil {
						log("查询二级代理下销售出错:[%v]", err)
						return money, err
					}

					for j := range sres {
						sqlm += ps(" OR order_no LIKE '%s_%%'", sres[j]["account"].(string))
					}
				}
			}
			sqlm += ")"
			if bt > 0 {
				sqlm += ps(" and ep.unix>=%d", bt)
			}
			if et > 0 {
				sqlm += ps(" and ep.unix<%d", et)
			}
			sqlm += ";"
			//log("%s", sqlm)
			_, err = db.Raw(sqlm).Values(&res)
			if err != nil {
				log("查询订单出错:[%v]", err)
				return money, err
			}
			for pidx := range res {
				pitem := res[pidx]
				discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
				order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
				money += discount_price * order_quantity * deduct
			}
		}
	} else if user.Flag == 3 && user.Supid == 0 { // 一级代理商
		// 查询二级代理商,销售员,个人代理
		sql = ps("SELECT id,account,deduct,flag FROM user WHERE flag=3 AND supid=%d;", user.UserId)
		_, err := db.Raw(sql).Values(&res)
		if err != nil {
			log("查询一级代理下属失败:[%v]", err)
			return money, err
		}
		for i := range res {
			sitem := res[i]
			sqlm = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 AND (order_no LIKE '%s_%%'", sitem["account"].(string))
			deduct, _ := strconv.ParseFloat(sitem["deduct"].(string), 64)
			flag, _ := strconv.Atoi(sitem["flag"].(string))
			if flag == 3 {
				// 查询二级代理销售
				suid, _ := strconv.Atoi(sitem["id"].(string))
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", suid)
				var sres []orm.Params
				_, err := db.Raw(sql).Values(&sres)
				if err != nil {
					log("查询二级代理下销售出错:[%v]", err)
					return money, err
				}

				for j := range sres {
					sqlm += ps(" OR order_no LIKE '%s_%%'", sres[j]["account"].(string))
				}
			}
			sqlm += ")"
			if bt > 0 {
				sqlm += ps(" and ep.unix>=%d", bt)
			}
			if et > 0 {
				sqlm += ps(" and ep.unix<%d", et)
			}
			sqlm += ";"

			_, err = db.Raw(sqlm).Values(&res)
			if err != nil {
				log("查询订单出错:[%v]", err)
				return money, err
			}
			for pidx := range res {
				pitem := res[pidx]
				discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
				order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
				money += discount_price * order_quantity * deduct
			}
		}
	}

	return money, nil
}

// 所有代理商累计佣金 sid
func (this *DealerController) DealerBrokerage() {
	money, err := DealearCommission(this.User, 0, 0)

	if err != nil {
		this.Rec = &Recv{5, "查询失败", err.Error()}
		return
	}
	this.Rec = &Recv{3, "查询成功", money}
	return
}

// 代理商总客户数
func AddedUsersForAgents(uid int64, acc string, supid int, bt int64, et int64) (int, error) {
	var sqlm, sql string
	sqlm = ps("SELECT count(id) as cnt FROM `user` where dealer_acc LIKE '%s'", acc)
	db := orm.NewOrm()
	var res []orm.Params
	if supid > 0 { //二级代理商
		sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", supid)
		var sres []orm.Params
		_, err := db.Raw(sql).Values(&sres)
		if err != nil {
			log("查询user表出错:[%v]", err)
			return 0, err
		}

		for j := range sres {
			sqlm += ps(" OR dealer_acc LIKE '%s'", sres[j]["account"].(string))
		}
	} else { // 一级代理商
		// 查询二级代理商,销售员,个人代理
		sql = ps("SELECT id,account,flag FROM `user` WHERE flag>1 and flag<5 AND supid=%d;", uid)
		_, err := db.Raw(sql).Values(&res)
		if err != nil {
			log("查询user表出错:[%v]", err)
			return 0, err
		}
		for i := range res {
			sitem := res[i]
			sqlm += ps(" OR dealer_acc LIKE '%s'", sitem["account"].(string))

			flag, _ := strconv.Atoi(sitem["flag"].(string))
			if flag == 3 { //查询二级代理商下的销售
				suid, _ := strconv.Atoi(sitem["id"].(string))
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", suid)
				var sres []orm.Params
				_, err := db.Raw(sql).Values(&sres)
				if err != nil {
					log("查询user表出错:[%v]", err)
					return 0, err
				}

				for j := range sres {
					sqlm += ps(" OR dealer_acc LIKE '%s'", sres[j]["account"].(string))
				}
			}
		}
	}

	if bt > 0 {
		sqlm += ps(" and unix>=%d", bt)
	}
	if et > 0 {
		sqlm += ps(" and unix<%d", et)
	}
	sqlm += ";"

	_, err := db.Raw(sqlm).Values(&res)
	if err != nil {
		log("查询客户总数出错:[%v]", err)
		return 0, err
	}
	users, _ := strconv.Atoi(res[0]["cnt"].(string))
	return users, nil
}

// 查询代理商销量
func AddedSalesForAgents(uid int64, acc string, supid int, bt int64, et int64) (float64, error) {
	money := 0.0
	db := orm.NewOrm()
	var sqlm, sql string = "", ""
	var res []orm.Params
	sqlm = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and (order_no LIKE '%s_%%'", acc)
	if supid == 0 {
		// 查询二级代理商,二级销售员,二级个人代理
		sql = ps("SELECT id,account,flag FROM `user` WHERE flag>1 and flag<5 AND supid=%d;", uid)
		_, err := db.Raw(sql).Values(&res)
		if err != nil {
			log("查询user表出错:[%v]", err)
			return 0.0, err
		}

		for i := range res {
			sitem := res[i]
			sqlm += ps(" OR order_no LIKE '%s_%%'", sitem["account"].(string))
			flag, _ := strconv.Atoi(sitem["flag"].(string))
			if flag == 3 { // 查询二级代理商下的销售
				suid, _ := strconv.Atoi(sitem["id"].(string))
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", suid)
				var sres []orm.Params
				_, err := db.Raw(sql).Values(&sres)
				if err != nil {
					log("查询user表出错:[%v]", err)
					return 0.0, err
				}

				for j := range sres {
					sqlm += ps(" OR order_no LIKE '%s_%%'", sres[j]["account"].(string))
				}
			}
		}
	} else {
		// 查询销售
		sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", uid)
		var sres []orm.Params
		_, err := db.Raw(sql).Values(&sres)
		if err != nil {
			log("查询user表出错:[%v]", err)
			return 0.0, err
		}

		for j := range sres {
			sqlm += ps(" OR order_no LIKE '%s_%%'", sres[j]["account"].(string))
		}
	}
	sqlm += ps(") and ep.unix>=%d and ep.unix<%d;", bt, et)
	_, err := db.Raw(sqlm).Values(&res)
	if err != nil {
		log("查询订单出错:[%v]", err)
		return 0.0, nil
	}
	for pidx := range res {
		pitem := res[pidx]
		discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
		order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
		money += discount_price * order_quantity
	}

	return money, nil
}

// sid,auid(代理商用户id)
func (this *DealerController) DealerPerform() {
	auid, _ := this.GetInt64("auid")

	// 参数检测
	if !CheckArg(auid) {
		this.Rec = &Recv{5, "代理商uid不能为空", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err := db.Raw("select account,id,supid from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询代理商信息失败", nil}
		return
	}
	acc := res[0]["account"].(string)
	supid, _ := strconv.Atoi(res[0]["supid"].(string))

	// 销售数量
	var salers, pagents, users, month_users int
	var sales, month_sales float64
	sql := ps("SELECT count(id) as cnt FROM `user` WHERE flag=2 AND supid=%d;", auid)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询销售人数失败", nil}
		return
	}
	if len(res) > 0 {
		salers, _ = strconv.Atoi(res[0]["cnt"].(string))
	}

	// 个人代理
	sql = ps("SELECT count(id) as cnt FROM `user` WHERE flag=4 AND supid=%d;", auid)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询个人代理人数失败", nil}
		return
	}
	if len(res) > 0 {
		pagents, _ = strconv.Atoi(res[0]["cnt"].(string))
	}

	// 查询客户总数
	users, _ = AddedUsersForAgents(auid, acc, supid, 0, 0)
	// 查询当月客户总数
	today := time.Now().Format("2006-01")
	month, _ := time.ParseInLocation("2006-01", today, time.Local)
	month_users, _ = AddedUsersForAgents(auid, acc, supid, month.Unix(), 0)

	sales, _ = AddedSalesForAgents(auid, acc, supid, 0, 0)
	month_sales, _ = AddedSalesForAgents(auid, acc, supid, month.Unix(), 0)
	type RecvEx struct {
		Salers      int
		Pagents     int
		Users       int
		Sales       float64
		Month_users int
		Month_sales float64
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{salers, pagents, users, sales, month_users, month_sales}}
	return
}

// sid,auid(代理商用户id)
func (this *DealerController) DealerUserparch() {
	auid, _ := this.GetInt64("auid")

	// 参数检测
	if !CheckArg(auid) {
		this.Rec = &Recv{5, "代理商uid不能为空", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err := db.Raw("select account,supid,deduct from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询代理商信息失败", nil}
		return
	}
	acc := res[0]["account"].(string)
	supid, _ := strconv.Atoi(res[0]["supid"].(string))
	deduct, _ := strconv.ParseFloat(res[0]["deduct"].(string), 64)

	var sqlm string = ""
	sqlm = ps("SELECT id,unix,dealer_acc,nick,account,log_unix FROM `user` where dealer_acc LIKE '%s'", acc)
	if supid == 0 {
		sql := ps("SELECT id,account,flag FROM `user` WHERE flag>1 and flag<5 AND supid=%d;", auid)
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询客户失败:[%v]", err)
			this.Rec = &Recv{5, "查询客户失败", nil}
		}
		for i := range res {
			sitem := res[i]
			sqlm += ps(" OR dealer_acc LIKE '%s'", sitem["account"].(string))

			flag, _ := strconv.Atoi(sitem["flag"].(string))
			if flag == 3 { // 查询二级代理商下的销售
				suid, _ := strconv.Atoi(sitem["id"].(string))
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", suid)
				var sres []orm.Params
				_, err := db.Raw(sql).Values(&sres)
				if err != nil {
					log("查询user表出错:[%v]", err)
					this.Rec = &Recv{5, "查询个人代理人数失败", nil}
				}

				for j := range sres {
					sqlm += ps(" OR dealer_acc LIKE '%s'", sres[j]["account"].(string))
				}
			}
		}
	} else { //二级代理商
		sql := ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", auid)
		var sres []orm.Params
		_, err := db.Raw(sql).Values(&sres)
		if err != nil {
			log("查询user表出错:[%v]", err)
			this.Rec = &Recv{5, "查询个人代理人数失败", nil}
		}

		for j := range sres {
			sqlm += ps(" OR dealer_acc LIKE '%s'", sres[j]["account"].(string))
		}
	}
	sqlm += ";"

	_, err = db.Raw(sqlm).Values(&res)
	if err != nil {
		log("查询客户失败:[%v]", err)
		this.Rec = &Recv{5, "查询客户失败", nil}
	}

	// 最近购买时间,购买总金额,代理商累计佣金
	for i := range res {
		item := res[i]
		id, _ := strconv.Atoi(item["id"].(string))
		var restmp []orm.Params
		_, err = db.Raw("select unix from enjoy_product where user_id=? and pay_status=1 order by unix desc limit 1;", id).Values(&restmp)
		if err != nil {
			log("查询最近购买时间失败:[%v]", err)
		} else {
			if len(restmp) > 0 {
				item["recent_unix"] = restmp[0]["unix"]
			}
		}

		// 购买总金额
		_, err = db.Raw("select ep.order_quantity,p.discount_price from enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and ep.user_id=?;", id).Values(&restmp)
		if err != nil {
			log("查询购买金额失败:[%v]", err)
		} else {
			money := 0.0
			for pidx := range restmp {
				pitem := restmp[pidx]
				discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
				order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
				money += discount_price * order_quantity
			}
			item["money"] = money
			item["brokerage"] = deduct * money
		}
	}

	this.Rec = &Recv{3, "查询成功", res}
	return
}

// sid
func (this *DealerController) AgentsRanking() {
	db := orm.NewOrm()
	var res []orm.Params

	var supid int64
	if this.User.Flag == 1 { //平台管理员
		supid = 0
	} else {
		supid = this.User.UserId
	}

	sql := ps("SELECT id,account,nick,supid FROM `user` WHERE flag=3 AND supid=%d;", supid)
	_, err := db.Raw(sql).Values(&res)
	if err != nil {
		log("查询user表失败:[%v]", err)
		this.Rec = &Recv{5, "查询销售失败", nil}
	}

	// 统计每个个人代理累计销售额和上月销售额
	date := time.Now().Format("2016-01")
	month, _ := time.ParseInLocation("2006-01", date, time.Local)
	month_unix := month.Unix()
	pre_mon_unix := month.AddDate(0, -1, 0).Unix()
	pre_twomon_unix := month.AddDate(0, -2, 0).Unix()
	for i := range res {
		item := res[i]
		id, _ := strconv.Atoi(item["id"].(string))
		supid, _ := strconv.Atoi(item["supid"].(string))
		sales, _ := AddedSalesForAgents(int64(id), item["account"].(string), supid, 0, 0)
		pre_mon_sales, _ := AddedSalesForAgents(int64(id), item["account"].(string), supid, pre_mon_unix, month_unix)
		pre_twomon_unix, _ := AddedSalesForAgents(int64(id), item["account"].(string), supid, pre_twomon_unix, pre_mon_unix)
		item["sales"] = sales
		item["pre_mon_sales"] = pre_mon_sales
		if pre_mon_sales >= pre_twomon_unix {
			item["trend"] = "增长"
		} else {
			item["trend"] = "下跌"
		}
	}

	this.Rec = &Recv{3, "查询成功", res}
	return
}

// sid,auid(经销商id),uid(用户id)
func (this *DealerController) DealerUserorder() {
	auid, _ := this.GetInt64("auid")
	uid, _ := this.GetInt64("uid")

	// 参数检测
	if !CheckArg(auid) {
		this.Rec = &Recv{5, "auid不能为空", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err := db.Raw("select deduct from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询代理商信息失败", nil}
		return
	}
	deduct, _ := strconv.ParseFloat(res[0]["deduct"].(string), 64)

	_, err = db.Raw("select ep.unix,ep.order_no,p.product_name,p.start_date,p.end_date,ep.hosted_mid,ep.order_quantity,ep.hosted_city,p.discount_price,ep.`status` from enjoy_product as ep,product as p where p.id=ep.pid and ep.user_id=? and ep.pay_status=1 order by ep.unix desc;", uid).Values(&res)
	if err != nil {
		log("查询订单失败:[%v]", err)
		this.Rec = &Recv{5, "查询订单失败", nil}
		return
	}

	for i := range res {
		item := res[i]
		order_quantity, _ := strconv.ParseFloat(item["order_quantity"].(string), 64)
		discount_price, _ := strconv.ParseFloat(item["discount_price"].(string), 64)
		item["totalprice"] = order_quantity * discount_price
		item["brokerage"] = order_quantity * discount_price * deduct
	}

	this.Rec = &Recv{3, "查询订单成功", res}
	return
}

// sid,date(eg. 2017-01)
func (this *DealerController) DealerCount() {
	date := this.GetString("date")

	// 参数检测
	if !CheckArg(date) {
		this.Rec = &Recv{5, "月份不能为空", nil}
		return
	}

	month, err := time.ParseInLocation("2006-01", date, time.Local)
	if err != nil {
		log("时间格式错误:%s", date)
		this.Rec = &Recv{5, "时间格式不对", nil}
		return
	}

	// 查询所有一级或二级代理
	db := orm.NewOrm()
	var res []orm.Params
	var supid int64 = 0
	if this.User.Flag == 3 {
		supid = this.User.UserId
	}
	sql := ps("SELECT id,account,nick,supid,deduct FROM `user` WHERE flag=3 AND supid=%d;", supid)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询user表出错", nil}
	}

	// 查询本月累计佣金
	nextmonth_unix := month.AddDate(0, 1, 0).Unix()
	money, err := DealearCommission(this.User, month.Unix(), nextmonth_unix)

	// 查询每个代理商信息
	var restmp []orm.Params
	for i := range res {
		item := res[i]
		supid, _ := strconv.Atoi(item["supid"].(string))
		acc := item["account"].(string)
		id, _ := strconv.Atoi(item["id"].(string))
		deduct, _ := strconv.ParseFloat(item["deduct"].(string), 64)
		// 订单数量
		sqlm := ps("select count(id) as num from enjoy_product where pay_status=1 and (order_no LIKE '%s_%%'", acc)
		if supid > 0 { // 二级代理商
			sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", id)
			_, err := db.Raw(sql).Values(&restmp)
			if err != nil {
				log("查询user表出错:[%v]", err)
				this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
				return
			}

			for i := range restmp {
				sitem := restmp[i]
				sqlm += ps(" OR order_no LIKE '%s_%%'", sitem["account"].(string))
			}
		} else {
			sql = ps("SELECT id,account,flag,supid FROM `user` WHERE flag>1 AND flag<5 AND supid=%d;", id)
			_, err := db.Raw(sql).Values(&restmp)
			if err != nil {
				log("查询user表出错:[%v]", err)
				this.Rec = &Recv{5, "查询一级代理商下属出错", nil}
				return
			}

			for i := range restmp {
				sitem := restmp[i]
				flag, _ := strconv.Atoi(sitem["flag"].(string))
				supid, _ := strconv.Atoi(sitem["supid"].(string))
				sqlm += ps(" OR order_no LIKE '%s_%%'", sitem["account"].(string))
				if flag == 3 {
					sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", supid)
					var saleres []orm.Params
					_, err := db.Raw(sql).Values(&saleres)
					if err != nil {
						log("查询user表出错:[%v]", err)
						this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
						return
					}

					for j := range saleres {
						sqlm += ps(" OR order_no LIKE '%s_%%'", saleres[j]["account"].(string))
					}
				}
			}
		}
		sqlm += ")"
		sqlm += ps(" and unix>=%d and unix<%d;", month.Unix(), nextmonth_unix)
		//log("123:%s", sqlm)
		_, err = db.Raw(sqlm).Values(&restmp)
		if err != nil {
			log("查询订单数量出错:[%v]", err)
			this.Rec = &Recv{5, "查询订单数量出错", nil}
			return
		}
		if restmp[0]["num"] != nil {
			item["orders"], _ = strconv.Atoi(restmp[0]["num"].(string))
		} else {
			item["orders"] = 0
		}

		// 当月销售总额
		month_sales, _ := AddedSalesForAgents(int64(id), acc, supid, month.Unix(), nextmonth_unix)
		item["month_sales"] = month_sales

		// 应得佣金
		item["month_brokerage"] = deduct * month_sales
	}

	type RecvEx struct {
		Brokerage float64
		Detail    interface{}
	}

	this.Rec = &Recv{3, "查询订单出错", &RecvEx{money, res}}
	return
}

// sid,auid(代理商用户id),date(eg. 2017-01)
func (this *DealerController) AgentsOrderQuery() {
	auid, _ := this.GetInt64("auid")
	date := this.GetString("date")

	// 参数检测
	if !CheckArg(date, auid) {
		this.Rec = &Recv{5, "代理商uid和月份不能为空", nil}
		return
	}

	month, err := time.ParseInLocation("2006-01", date, time.Local)
	if err != nil {
		log("时间格式错误:%s", date)
		this.Rec = &Recv{5, "时间格式不对", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err = db.Raw("select account,id,supid from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询代理商信息失败", nil}
		return
	}
	acc := res[0]["account"].(string)
	supid, _ := strconv.Atoi(res[0]["supid"].(string))

	sqlm := ps("select ep.id,ep.unix,u.nick,u.account,ep.order_no,p.discount_price,ep.order_quantity from enjoy_product as ep,user as u,product as p where ep.user_id=u.id and p.id=ep.pid and ep.pay_status=1 and (ep.order_no LIKE '%s_%%'", acc)
	if supid > 0 { // 二级代理商
		sql := ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", auid)
		var restmp []orm.Params
		_, err := db.Raw(sql).Values(&restmp)
		if err != nil {
			log("查询user表出错:[%v]", err)
			this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
		}

		for i := range restmp {
			sitem := restmp[i]
			sqlm += ps(" OR ep.order_no LIKE '%s_%%'", sitem["account"].(string))
		}
	} else {
		sql := ps("SELECT id,account,flag,supid FROM `user` WHERE flag>1 AND flag<5 AND supid=%d;", auid)
		var restmp []orm.Params
		_, err := db.Raw(sql).Values(&restmp)
		if err != nil {
			log("查询user表出错:[%v]", err)
			this.Rec = &Recv{5, "查询一级代理商下属出错", nil}
		}

		for i := range restmp {
			item := restmp[i]
			flag, _ := strconv.Atoi(item["flag"].(string))
			supid, _ := strconv.Atoi(item["supid"].(string))
			sqlm += ps(" OR ep.order_no LIKE '%s_%%'", item["account"].(string))
			if flag == 3 {
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", supid)
				var saleres []orm.Params
				_, err := db.Raw(sql).Values(&saleres)
				if err != nil {
					log("查询user表出错:[%v]", err)
					this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
				}

				for j := range saleres {
					sqlm += ps(" OR ep.order_no LIKE '%s_%%'", saleres[j]["account"].(string))
				}
			}
		}
	}
	sqlm += ps(")")
	nextmonth_unix := month.AddDate(0, 1, 0).Unix()
	sqlm += ps(" and ep.unix>=%d and ep.unix<%d;", month.Unix(), nextmonth_unix)
	_, err = db.Raw(sqlm).Values(&res)
	if err != nil {
		log("查询订单出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单出错", nil}
	}

	// 判断来源,来源编号,总价,佣金
	for i := range res {
		item := res[i]
		idx := strings.Index(item["order_no"].(string), "_")
		src_acc := item["order_no"].(string)
		src_acc = src_acc[0:idx]
		item["src_acc"] = src_acc

		var restmp []orm.Params
		cnts, err := db.Raw("select flag from `user` where account=?;", src_acc).Values(&restmp)
		if err == nil {
			if cnts > 0 {
				item["flag"] = restmp[0]["flag"]
			} else {
				item["flag"] = ""
			}
		}

		// 总价和佣金
		discount_price, _ := strconv.ParseFloat(item["discount_price"].(string), 64)
		order_quantity, _ := strconv.Atoi(item["order_quantity"].(string))
		item["total_price"] = discount_price * float64(order_quantity)
		item["brokerage"] = discount_price * float64(order_quantity) * this.User.Deduct
	}

	this.Rec = &Recv{3, "查询成功", res}
	return
}

// 计算代理商自己应得佣金
// level:0-二级代理商,1-一级代理商
func AgentsPersonalBrokeageCount(user *Loginuser, bt int64, et int64) (money float64, err error) {
	db := orm.NewOrm()
	var sql, sqlm string = "", ""
	var res []orm.Params
	money = 0.0
	if user.Supid == 0 { // 一级代理商
		sqlm = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and (order_no LIKE '%s_%%'", user.Account)

		// 查询二级代理商,销售员,个人代理
		sql = ps("SELECT id,account,flag FROM `user` WHERE flag>1 and flag<5 AND supid='%d';", user.UserId)
		_, err := db.Raw(sql).Values(&res)
		if err != nil {
			log("查询一级代理下属失败:[%v]", err)
			return money, err
		}

		for i := range res {
			sitem := res[i]
			sqlm += ps(" OR order_no LIKE '%s_%%'", sitem["account"].(string))

			flag, _ := strconv.Atoi(sitem["flag"].(string))
			if flag == 3 {
				// 查询二级代理旗下销售
				suid, _ := strconv.Atoi(sitem["id"].(string))
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid='%d';", suid)
				var sres []orm.Params
				_, err := db.Raw(sql).Values(&sres)
				if err != nil {
					log("查询二级代理下销售出错:[%v]", err)
					return money, err
				}

				for j := range sres {
					sqlm += ps(" OR order_no LIKE '%s_%%'", sres[j]["account"].(string))
				}
			}
		}
		sqlm += ")"
		if bt > 0 {
			sqlm += ps(" and ep.unix>=%d", bt)
		}
		if et > 0 {
			sqlm += ps(" and ep.unix<%d", et)
		}
		sqlm += ";"

		_, err = db.Raw(sqlm).Values(&res)
		if err != nil {
			log("查询订单出错:[%v]", err)
			return money, err
		}
		for pidx := range res {
			pitem := res[pidx]
			discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
			order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
			money += discount_price * order_quantity * user.Deduct
		}
	} else if user.Supid > 0 {
		sqlm = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 AND (order_no LIKE '%s_%%'", user.Account)
		// 查询旗下销售员
		sql = ps("SELECT account FROM user WHERE flag=2 AND supid='%d';", user.UserId)
		_, err := db.Raw(sql).Values(&res)
		if err != nil {
			log("查询二级代理下销售失败:[%v]", err)
			return money, err
		}

		for i := range res {
			sqlm += ps(" OR order_no LIKE '%s_%%'", res[i]["account"].(string))
		}
		sqlm += ")"
		if bt > 0 {
			sqlm += ps(" and ep.unix>=%d", bt)
		}
		if et > 0 {
			sqlm += ps(" and ep.unix<%d", et)
		}
		sqlm += ";"

		_, err = db.Raw(sqlm).Values(&res)
		if err != nil {
			log("查询订单出错:[%v]", err)
			return money, err
		}
		for idx := range res {
			item := res[idx]
			discount_price, _ := strconv.ParseFloat(item["discount_price"].(string), 64)
			order_quantity, _ := strconv.ParseFloat(item["order_quantity"].(string), 64)
			money += discount_price * order_quantity * user.Deduct
		}
	}

	return money, nil
}

// sid
func (this *DealerController) AgentsPersonalBrokeage() {
	money, err := AgentsPersonalBrokeageCount(this.User, 0, 0)

	if err != nil {
		this.Rec = &Recv{5, "查询失败", err.Error()}
		return
	}
	this.Rec = &Recv{3, "查询成功", money}
	return
}

// sid,date(eg. 2017.01)
func (this *DealerController) AgentsPersonalOrder() {
	date := this.GetString("date")

	// 参数检测
	if !CheckArg(date) {
		this.Rec = &Recv{5, "月份不能为空", nil}
		return
	}

	// 查询月佣金
	month, err := time.ParseInLocation("2006-01", date, time.Local)
	if err != nil {
		log("时间格式错误:%s", date)
		this.Rec = &Recv{5, "时间格式不对", nil}
		return
	}
	nextmonth_unix := month.AddDate(0, 1, 0).Unix()
	money, err := AgentsPersonalBrokeageCount(this.User, month.Unix(), nextmonth_unix)

	// 查询客户订单
	db := orm.NewOrm()
	sqlm := ps("select ep.id,ep.unix,u.nick,u.account,ep.order_no,p.discount_price,ep.order_quantity from enjoy_product as ep,user as u,product as p where ep.user_id=u.id and p.id=ep.pid and ep.pay_status=1 and (ep.order_no LIKE '%s_%%'", this.User.Account)
	var res []orm.Params
	if this.User.Supid > 0 { // 二级代理商
		sql := ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", this.User.UserId)
		var restmp []orm.Params
		_, err := db.Raw(sql).Values(&restmp)
		if err != nil {
			log("查询user表出错:[%v]", err)
			this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
		}

		for i := range restmp {
			sitem := restmp[i]
			sqlm += ps(" OR ep.order_no LIKE '%s_%%'", sitem["account"].(string))
		}
	} else {
		sql := ps("SELECT id,account,flag FROM `user` WHERE flag>1 AND flag<5 AND supid=%d;", this.User.UserId)
		var restmp []orm.Params
		_, err := db.Raw(sql).Values(&restmp)
		if err != nil {
			log("查询user表出错:[%v]", err)
			this.Rec = &Recv{5, "查询一级代理商下属出错", nil}
		}

		for i := range restmp {
			item := restmp[i]
			flag, _ := strconv.Atoi(item["flag"].(string))
			suid, _ := strconv.Atoi(item["id"].(string))
			sqlm += ps(" OR ep.order_no LIKE '%s_%%'", item["account"].(string))
			if flag == 3 {
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", suid)
				var saleres []orm.Params
				_, err := db.Raw(sql).Values(&saleres)
				if err != nil {
					log("查询user表出错:[%v]", err)
					this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
				}

				for j := range saleres {
					sqlm += ps(" OR ep.order_no LIKE '%s_%%'", saleres[j]["account"].(string))
				}
			}
		}
	}
	sqlm += ps(") and ep.unix>=%d and ep.unix<%d", month.Unix(), nextmonth_unix)
	_, err = db.Raw(sqlm).Values(&res)
	if err != nil {
		log("查询订单出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单出错", nil}
		return
	}

	// 判断来源,来源编号,总价,佣金
	for i := range res {
		item := res[i]
		idx := strings.Index(item["order_no"].(string), "_")
		src_acc := item["order_no"].(string)
		src_acc = src_acc[0:idx]
		item["src_acc"] = src_acc

		var restmp []orm.Params
		cnts, err := db.Raw("select flag from `user` where account=?;", src_acc).Values(&restmp)
		if err == nil {
			if cnts > 0 {
				item["flag"] = restmp[0]["flag"]
			} else {
				item["flag"] = ""
			}
		}

		// 总价和佣金
		discount_price, _ := strconv.ParseFloat(item["discount_price"].(string), 64)
		order_quantity, _ := strconv.Atoi(item["order_quantity"].(string))
		item["total_price"] = discount_price * float64(order_quantity)
		item["brokerage"] = discount_price * float64(order_quantity) * this.User.Deduct
	}

	type RecvEx struct {
		Money float64
		Order interface{}
	}

	this.Rec = &Recv{3, "查询成功", &RecvEx{money, res}}
	return
}

// 个人代理或销售客户数
func AddedUsersForPerorSales(acc string, bt int64, et int64) (int, error) {
	sql := ps("SELECT count(id) as cnt FROM `user` where dealer_acc LIKE '%s'", acc)
	db := orm.NewOrm()
	var res []orm.Params
	if bt > 0 {
		sql += ps(" and unix>=%d", bt)
	}
	if et > 0 {
		sql += ps(" and unix<%d", et)
	}
	sql += ";"
	_, err := db.Raw(sql).Values(&res)
	if err != nil {
		log("查询客户总数出错:[%v]", err)
		return 0, err
	}
	users, _ := strconv.Atoi(res[0]["cnt"].(string))
	return users, nil
}

// 查询个人代理或销售销售额
func AddedSalesForPerorSales(acc string, bt int64, et int64) (float64, error) {
	money := 0.0
	db := orm.NewOrm()
	var res []orm.Params
	sql := ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and order_no LIKE '%s_%%';", acc)
	_, err := db.Raw(sql).Values(&res)
	if err != nil {
		log("查询订单出错:[%v]", err)
		return 0.0, nil
	}
	for pidx := range res {
		pitem := res[pidx]
		discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
		order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
		money += discount_price * order_quantity
	}

	return money, nil
}

// sid,auid(个人代理用户id)
func (this *DealerController) PeragentsPerform() {
	auid, _ := this.GetInt64("auid")

	// 参数检测
	if !CheckArg(auid) {
		this.Rec = &Recv{5, "个人代理uid不能为空", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err := db.Raw("select account from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询代理商信息失败", nil}
		return
	}
	acc := res[0]["account"].(string)

	var users, month_users int
	var sales, month_sales float64
	// 查询客户总数
	users, _ = AddedUsersForPerorSales(acc, 0, 0)
	// 查询当月客户总数
	today := time.Now().Format("2006-01")
	month, _ := time.ParseInLocation("2006-01", today, time.Local)
	month_users, _ = AddedUsersForPerorSales(acc, month.Unix(), 0)

	sales, _ = AddedSalesForPerorSales(acc, 0, 0)
	month_sales, _ = AddedSalesForPerorSales(acc, month.Unix(), 0)
	type RecvEx struct {
		Users       int
		Sales       float64
		Month_users int
		Month_sales float64
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{users, sales, month_users, month_sales}}
	return
}

// sid,auid(个人代理用户id)
func (this *DealerController) PeragentsUserparch() {
	auid, _ := this.GetInt64("auid")

	// 参数检测
	if !CheckArg(auid) {
		this.Rec = &Recv{5, "个人代理uid不能为空", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err := db.Raw("select deduct,account from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询个人代理信息失败", nil}
		return
	}
	acc := res[0]["account"].(string)
	deduct, _ := strconv.ParseFloat(res[0]["deduct"].(string), 64)

	sql := ps("SELECT id,unix,dealer_acc,nick,account,log_unix FROM `user` where dealer_acc LIKE '%s';", acc)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询客户失败:[%v]", err)
		this.Rec = &Recv{5, "查询客户失败", nil}
	}

	// 最近购买时间,购买总金额,个人代理累计佣金
	for i := range res {
		item := res[i]
		id, _ := strconv.Atoi(item["id"].(string))
		var restmp []orm.Params
		_, err = db.Raw("select unix from enjoy_product where user_id=? and pay_status=1 order by unix desc limit 1;", id).Values(&restmp)
		if err != nil {
			log("查询最近购买时间失败:[%v]", err)
		} else {
			if len(restmp) > 0 {
				item["recent_unix"] = restmp[0]["unix"]
			} else {
				item["recent_unix"] = ""
			}
		}

		// 购买总金额
		_, err = db.Raw("select ep.order_quantity,p.discount_price from enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and ep.user_id=?;", id).Values(&restmp)
		if err != nil {
			log("查询购买金额失败:[%v]", err)
		} else {
			money := 0.0
			for pidx := range restmp {
				pitem := restmp[pidx]
				discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
				order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
				money += discount_price * order_quantity
			}
			item["money"] = money
			item["brokerage"] = deduct * money
		}
	}

	this.Rec = &Recv{3, "查询成功", res}
	return
}

// sid
func (this *DealerController) PeragentsRanking() {
	db := orm.NewOrm()
	var res []orm.Params

	var supid int64
	if this.User.Flag == 1 { //平台管理员
		supid = 0
	} else {
		supid = this.User.UserId
	}

	sql := ps("SELECT account,nick FROM `user` WHERE flag=4 AND supid=%d;", supid)
	_, err := db.Raw(sql).Values(&res)
	if err != nil {
		log("查询user表失败:[%v]", err)
		this.Rec = &Recv{5, "查询销售失败", nil}
	}

	// 统计每个个人代理累计销售额和上月销售额
	date := time.Now().Format("2016-01")
	month, _ := time.ParseInLocation("2006-01", date, time.Local)
	month_unix := month.Unix()
	pre_mon_unix := month.AddDate(0, -1, 0).Unix()
	pre_twomon_unix := month.AddDate(0, -2, 0).Unix()
	for i := range res {
		item := res[i]
		sales, _ := AddedSalesForPerorSales(item["account"].(string), 0, 0)
		pre_mon_sales, _ := AddedSalesForPerorSales(item["account"].(string), pre_mon_unix, month_unix)
		pre_twomon_unix, _ := AddedSalesForPerorSales(item["account"].(string), pre_twomon_unix, pre_mon_unix)
		item["sales"] = sales
		item["pre_mon_sales"] = pre_mon_sales
		if pre_mon_sales >= pre_twomon_unix {
			item["trend"] = "增长"
		} else {
			item["trend"] = "下跌"
		}
	}

	this.Rec = &Recv{3, "查询成功", res}
	return
}

// 所有个人代理累计佣金
func PeragentsTotalBrokerage(user *Loginuser, bt int64, et int64) (money float64, err error) {
	db := orm.NewOrm()
	var sql, sqlm string = "", ""
	var result, res []orm.Params
	money = 0.0
	var supid int64 = 0 // 平台个人代理
	if user.Flag == 3 { // 一级代理商
		supid = user.UserId
	}

	// 查询所有代理商
	if user.Flag == 1 || user.Flag == 3 {
		sql = ps("SELECT account,deduct FROM `user` WHERE flag=4 AND supid='%d';", supid)
		_, err = db.Raw(sql).Values(&result)
		if err != nil {
			log("查询个人代理出错:[%v]", err)
			return money, err
		}

		for idx := range result {
			item := result[idx]
			deduct, _ := strconv.ParseFloat(item["deduct"].(string), 64) // 分成比率
			sqlm = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and ep.order_no LIKE '%s_%%'", item["account"].(string))
			if bt > 0 {
				sqlm += ps(" and ep.unix>=%d", bt)
			}
			if et > 0 {
				sqlm += ps(" and ep.unix<%d", et)
			}
			sqlm += ";"
			//log("代理商订单:%s", sqlm)
			_, err = db.Raw(sqlm).Values(&res)
			if err != nil {
				log("查询订单出错:[%v]", err)
				return money, err
			}
			for pidx := range res {
				pitem := res[pidx]
				discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
				order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
				money += discount_price * order_quantity * deduct
			}
		}
	} else if user.Flag == 4 { // 个人代理自己的佣金
		sql = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and ep.order_no LIKE '%s_%%'", user.Account)
		if bt > 0 {
			sql += ps(" and ep.unix>=%d", bt)
		}
		if et > 0 {
			sql += ps(" and ep.unix<%d", et)
		}
		sql += ";"
		_, err = db.Raw(sql).Values(&res)
		if err != nil {
			log("查询订单出错:[%v]", err)
			return money, err
		}
		for idx := range res {
			item := res[idx]
			discount_price, _ := strconv.ParseFloat(item["discount_price"].(string), 64)
			order_quantity, _ := strconv.ParseFloat(item["order_quantity"].(string), 64)
			money += discount_price * order_quantity * user.Deduct
		}
	}

	return money, nil
}

// sid
func (this *DealerController) PeragentsBrokerage() {
	money, err := PeragentsTotalBrokerage(this.User, 0, 0)

	if err != nil {
		this.Rec = &Recv{5, "查询失败", err.Error()}
		return
	}
	this.Rec = &Recv{3, "查询成功", money}
	return
}

// sid,date(eg. 2017-01)
func (this *DealerController) PeragentsCount() {
	date := this.GetString("date")

	// 参数检测
	if !CheckArg(date) {
		this.Rec = &Recv{5, "月份不能为空", nil}
		return
	}

	// 查询本月累计佣金
	month, err := time.ParseInLocation("2006-01", date, time.Local)
	if err != nil {
		log("时间格式错误:%s", date)
		this.Rec = &Recv{5, "时间格式不对", nil}
		return
	}
	nextmonth_unix := month.AddDate(0, 1, 0).Unix()
	money, err := PeragentsTotalBrokerage(this.User, month.Unix(), nextmonth_unix)

	// 查询所有个人代理
	db := orm.NewOrm()
	var res, restmp []orm.Params
	var supid int64 = 0
	if this.User.Flag == 3 { //一级代理商
		supid = this.User.UserId
	}
	sql := ps("SELECT id,account,nick,deduct FROM `user` WHERE flag=4 AND supid=%d;", supid)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询user表出错", nil}
	}

	// 查询每个个人代理信息
	for i := range res {
		item := res[i]
		acc := item["account"].(string)
		deduct, _ := strconv.ParseFloat(item["deduct"].(string), 64)
		// 订单数量
		sqlm := ps("select count(id) as num from enjoy_product where pay_status=1 and order_no LIKE '%s_%%' and unix>=%d and unix<%d;", acc, month.Unix(), nextmonth_unix)
		_, err = db.Raw(sqlm).Values(&restmp)
		if err != nil {
			log("查询订单总数出错:[%v]", err)
			this.Rec = &Recv{5, "查询订单总数出错", nil}
		}
		item["orders"], _ = strconv.Atoi(restmp[0]["num"].(string))

		// 当月销售总额
		sql = ps("SELECT ep.order_quantity,p.discount_price FROM enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and ep.order_no LIKE '%s_%%' and ep.unix>=%d and ep.unix<%d;",
			acc, month.Unix(), nextmonth_unix)
		_, err = db.Raw(sql).Values(&restmp)
		//log("个人代理订单:%s", sql)
		if err != nil {
			log("查询订单出错:[%v]", err)
		}
		month_sales := 0.0
		for idx := range restmp {
			item := restmp[idx]
			discount_price, _ := strconv.ParseFloat(item["discount_price"].(string), 64)
			order_quantity, _ := strconv.ParseFloat(item["order_quantity"].(string), 64)
			month_sales += discount_price * order_quantity
		}
		item["month_sales"] = month_sales

		// 应得佣金
		item["month_brokerage"] = deduct * month_sales
	}

	type RecvEx struct {
		Brokerage float64
		Detail    interface{}
	}

	this.Rec = &Recv{3, "查询成功", &RecvEx{money, res}}
	return
}

// sid
func (this *DealerController) PeragentsMybrokerage() {
	money, err := PeragentsTotalBrokerage(this.User, 0, 0)

	if err != nil {
		this.Rec = &Recv{5, "查询失败", err.Error()}
		return
	}
	this.Rec = &Recv{3, "查询成功", money}
	return
}

// sid,date(eg. 2017-01)
func (this *DealerController) PeragentsMyorder() {
	date := this.GetString("date")

	// 参数检测
	if !CheckArg(date) {
		this.Rec = &Recv{5, "月份不能为空", nil}
		return
	}

	// 查询月佣金
	month, err := time.ParseInLocation("2006-01", date, time.Local)
	if err != nil {
		log("时间格式错误:%s", date)
		this.Rec = &Recv{5, "时间格式不对", nil}
		return
	}
	nextmonth_unix := month.AddDate(0, 1, 0).Unix()
	money, err := PeragentsTotalBrokerage(this.User, month.Unix(), nextmonth_unix)

	// 查询客户订单
	db := orm.NewOrm()
	sqlm := ps("select ep.id,ep.unix,u.nick,u.account,ep.order_no,p.discount_price,ep.order_quantity from enjoy_product as ep,user as u,product as p where ep.user_id=u.id and p.id=ep.pid and ep.pay_status=1 and (ep.order_no LIKE '%s_%%'", this.User.Account)
	var res []orm.Params
	if this.User.Supid > 0 { // 二级代理商
		sql := ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", this.User.UserId)
		var restmp []orm.Params
		_, err := db.Raw(sql).Values(&restmp)
		if err != nil {
			log("查询user表出错:[%v]", err)
			this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
		}

		for i := range restmp {
			sitem := restmp[i]
			sqlm += ps(" OR ep.order_no LIKE '%s_%%'", sitem["account"].(string))
		}
	} else {
		sql := ps("SELECT id,account,flag FROM `user` WHERE flag>1 AND flag<5 AND supid=%d;", this.User.UserId)
		var restmp []orm.Params
		_, err := db.Raw(sql).Values(&restmp)
		if err != nil {
			log("查询user表出错:[%v]", err)
			this.Rec = &Recv{5, "查询一级代理商下属出错", nil}
		}

		for i := range restmp {
			item := restmp[i]
			flag, _ := strconv.Atoi(item["flag"].(string))
			suid, _ := strconv.Atoi(item["suid"].(string))
			sqlm += ps(" OR ep.order_no LIKE '%s_%%'", item["account"].(string))
			if flag == 3 {
				sql = ps("SELECT account FROM `user` WHERE flag=2 AND supid=%d;", suid)
				var saleres []orm.Params
				_, err := db.Raw(sql).Values(&saleres)
				if err != nil {
					log("查询user表出错:[%v]", err)
					this.Rec = &Recv{5, "查询二级代理商下销售出错", nil}
				}

				for j := range saleres {
					sqlm += ps(" OR ep.order_no LIKE '%s_%%'", saleres[j]["account"].(string))
				}
			}
		}
	}
	sqlm += ps(")")
	sqlm += ps(" and ep.unix>=%d and ep.unix<%d;", month.Unix(), nextmonth_unix)
	_, err = db.Raw(sqlm).Values(&res)
	if err != nil {
		log("查询订单出错:[%v]", err)
		this.Rec = &Recv{5, "查询订单出错", nil}
	}

	// 总价,佣金
	for i := range res {
		item := res[i]
		discount_price, _ := strconv.ParseFloat(item["discount_price"].(string), 64)
		order_quantity, _ := strconv.Atoi(item["order_quantity"].(string))
		item["total_price"] = discount_price * float64(order_quantity)
		item["brokerage"] = discount_price * float64(order_quantity) * this.User.Deduct
	}

	type RecvEx struct {
		Money float64
		Order interface{}
	}

	this.Rec = &Recv{3, "查询成功", &RecvEx{money, res}}
	return

}

// sid,auid(销售用户id)
func (this *DealerController) SalesPerform() {
	auid, _ := this.GetInt64("auid")

	// 参数检测
	if !CheckArg(auid) {
		this.Rec = &Recv{5, "销售uid不能为空", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err := db.Raw("select account from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询代理商信息失败", nil}
		return
	}
	acc := res[0]["account"].(string)

	var users, month_users int
	var sales, month_sales float64
	// 查询客户总数
	users, _ = AddedUsersForPerorSales(acc, 0, 0)
	// 查询当月客户总数
	today := time.Now().Format("2006-01")
	month, _ := time.ParseInLocation("2006-01", today, time.Local)
	month_users, _ = AddedUsersForPerorSales(acc, month.Unix(), 0)

	sales, _ = AddedSalesForPerorSales(acc, 0, 0)
	month_sales, _ = AddedSalesForPerorSales(acc, month.Unix(), 0)
	type RecvEx struct {
		Users       int
		Sales       float64
		Month_users int
		Month_sales float64
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{users, sales, month_users, month_sales}}
	return
}

// sid,auid(销售用户id)
func (this *DealerController) SalesUserparch() {
	auid, _ := this.GetInt64("auid")

	// 参数检测
	if !CheckArg(auid) {
		this.Rec = &Recv{5, "销售uid不能为空", nil}
		return
	}

	// 查询用户信息
	db := orm.NewOrm()
	var res []orm.Params
	_, err := db.Raw("select deduct,account from user where id=?", auid).Values(&res)
	if err != nil {
		log("查询user表出错:[%v]", err)
		this.Rec = &Recv{5, "查询销售信息失败", nil}
		return
	}
	acc := res[0]["account"].(string)
	deduct, _ := strconv.ParseFloat(res[0]["deduct"].(string), 64)

	sql := ps("SELECT id,unix,dealer_acc,nick,account,log_unix FROM `user` where dealer_acc LIKE '%s';", acc)
	_, err = db.Raw(sql).Values(&res)
	if err != nil {
		log("查询客户失败:[%v]", err)
		this.Rec = &Recv{5, "查询客户失败", nil}
	}

	// 最近购买时间,购买总金额,个人代理累计佣金
	for i := range res {
		item := res[i]
		id, _ := strconv.Atoi(item["id"].(string))
		var restmp []orm.Params
		_, err = db.Raw("select unix from enjoy_product where user_id=? and pay_status=1 order by unix desc limit 1;", id).Values(&restmp)
		if err != nil {
			log("查询最近购买时间失败:[%v]", err)
		} else {
			if len(restmp) > 0 {
				item["recent_unix"] = restmp[0]["unix"]
			} else {
				item["recent_unix"] = ""
			}
		}

		// 购买总金额
		_, err = db.Raw("select ep.order_quantity,p.discount_price from enjoy_product as ep,product as p where p.id=ep.pid and ep.pay_status=1 and ep.user_id=?;", id).Values(&restmp)
		if err != nil {
			log("查询购买金额失败:[%v]", err)
		} else {
			money := 0.0
			for pidx := range restmp {
				pitem := restmp[pidx]
				discount_price, _ := strconv.ParseFloat(pitem["discount_price"].(string), 64)
				order_quantity, _ := strconv.ParseFloat(pitem["order_quantity"].(string), 64)
				money += discount_price * order_quantity
			}
			item["money"] = money
			item["brokerage"] = deduct * money
		}
	}

	this.Rec = &Recv{3, "查询成功", res}
	return
}

// sid
func (this *DealerController) SalesRanking() {
	db := orm.NewOrm()
	var res []orm.Params

	var supid int64
	if this.User.Flag == 1 { //平台管理员
		supid = 0
	} else {
		supid = this.User.UserId
	}

	sql := ps("SELECT account,nick FROM `user` WHERE flag=2 AND supid=%d;", supid)
	_, err := db.Raw(sql).Values(&res)
	if err != nil {
		log("查询user表失败:[%v]", err)
		this.Rec = &Recv{5, "查询销售失败", nil}
	}

	// 统计每个销售累计销售额和上月销售额
	date := time.Now().Format("2016-01")
	month, _ := time.ParseInLocation("2006-01", date, time.Local)
	month_unix := month.Unix()
	pre_mon_unix := month.AddDate(0, -1, 0).Unix()
	pre_twomon_unix := month.AddDate(0, -2, 0).Unix()
	for i := range res {
		item := res[i]
		sales, _ := AddedSalesForPerorSales(item["account"].(string), 0, 0)
		pre_mon_sales, _ := AddedSalesForPerorSales(item["account"].(string), pre_mon_unix, month_unix)
		pre_twomon_unix, _ := AddedSalesForPerorSales(item["account"].(string), pre_twomon_unix, pre_mon_unix)
		item["sales"] = sales
		item["pre_mon_sales"] = pre_mon_sales
		if pre_mon_sales >= pre_twomon_unix {
			item["trend"] = "增长"
		} else {
			item["trend"] = "下跌"
		}
	}

	this.Rec = &Recv{3, "查询成功", res}
	return
}
