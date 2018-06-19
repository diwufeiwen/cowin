package controllers

import (
	"github.com/astaxie/beego/orm"
)

type ExtractController struct {
	OnlineController
}

// sid,bdt,edt,status(0-已提交,未到账;1-已到账)
func (this *ExtractController) ExtractQuery() {
	bdt, _ := this.GetInt64("bdt")
	edt, _ := this.GetInt64("edt")
	status, _ := this.GetInt32("status")

	var sql string = ""
	if status >= 0 {
		sql = ps("SELECT * FROM `extract_cash` WHERE uid='%d' and status=%d and unix<=%d and unix>%d order by unix desc;", this.User.UserId, status, edt, bdt)
	} else {
		sql = ps("SELECT * FROM `extract_cash` WHERE uid='%d' and unix<=%d and unix>%d order by unix desc;", this.User.UserId, edt, bdt)
	}

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}

// sid,account,bdt,edt,status
func (this *ExtractController) ExtractSrvQuery() {
	account := this.GetString("account")
	bdt, _ := this.GetInt64("bdt")
	edt, _ := this.GetInt64("edt")
	status, _ := this.GetInt32("status")

	var sql string = ""
	if !CheckArg(account) {
		if status >= 0 {
			sql = ps("SELECT * FROM `extract_cash` WHERE status=%d and unix<=%d and unix>%d order by unix desc;", status, edt, bdt)
		} else {
			sql = ps("SELECT * FROM `extract_cash` WHERE unix<=%d and unix>%d order by unix desc;", edt, bdt)
		}
	} else {
		if status >= 0 {
			sql = ps("SELECT * FROM `extract_cash` WHERE account='%s' and status=%d and unix<=%d and unix>%d order by unix desc;", account, status, edt, bdt)
		} else {
			sql = ps("SELECT * FROM `extract_cash` WHERE account='%s' and unix<=%d and unix>%d order by unix desc;", account, edt, bdt)
		}
	}

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}

// sid,money,destination,cardno
func (this *ExtractController) Extract() {
	destination := this.GetString("destination")
	cardno := this.GetString("cardno")
	money, _ := this.GetFloat("money")

	//检查参数
	if !CheckArg(destination, cardno, money) {
		this.Rec = &Recv{5, "此接口参数均不能为空", nil}
		return
	}

	if this.User.Wallet < money {
		this.Rec = &Recv{5, "金额不足", nil}
		return
	}

	var sql string = ps("insert into extract_cash (uid,money,destination,cardno,unix) values ('%d','%v','%s','%s','%d')",
		this.User.UserId, money, destination, cardno, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		_, strerr := ChecSQLerr(err)
		log("申请提取金额失败:[%v]", err)
		this.Rec = &Recv{5, ps("申请提取金额失败:[%s]", strerr), nil}
		return
	}

	this.User.Wallet -= money
	sql = ps("update `user` set wallet=wallet-%v where id=%d", money, this.User.UserId)
	this.Rec = &Recv{3, "申请提取金额成功", nil}
}
