package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
)

type CouponController struct {
	OnlineController
}

type CouponBaseController struct {
	BaseController
}

// category,flag(0-已发,1-未发,2-过期),begidx,counts
func (this *CouponBaseController) CouponQuery() {
	flag, _ := this.GetInt32("flag")
	category := this.GetString("category")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	var sql, sqlc string = "", ""
	switch flag {
	case 0:
		sql = ps("SELECT ic.id,c.name,c.category,c.price,ic.amount,ic.unix FROM `issue_coupon` AS ic,`coupon` AS c WHERE c.category='%s' AND ic.cnid=c.id ORDER BY ic.unix DESC limit %d,%d;",
			category, begidx, counts)
		sqlc = "SELECT count(id) as num FROM `issue_coupon` where category='%s';"
	case 1:
		sql = ps("SELECT * FROM `coupon` WHERE category='%s' AND end_date>%d ORDER BY unix DESC limit %d,%d;", category, TimeNow, begidx, counts)
		sqlc = ps("SELECT count(id) as num FROM `coupon` WHERE category='%s' AND end_date<%d;", category, TimeNow)
	case 2:
		sql = ps("SELECT * FROM `coupon` WHERE category='%s' AND end_date<=%d ORDER BY unix DESC limit %d,%d;", category, TimeNow, begidx, counts)
		sqlc = ps("SELECT count(id) as num FROM `coupon` WHERE category='%s' AND end_date>=%d;", category, TimeNow)
	}

	db := orm.NewOrm()
	var result []orm.Params
	var total int
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询总数失败:err[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	} else {
		total, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{total, result}}
	return
}

// sid,name,category,start_date,end_date,price,amount
func (this *CouponController) CouponAdd() {
	name := this.GetString("name")
	category := this.GetString("category")
	start_date, _ := this.GetInt64("start_date")
	end_date, _ := this.GetInt64("end_date")
	price, _ := this.GetFloat("price")
	amount, _ := this.GetInt32("amount")

	//检查参数
	if !CheckArg(name, category, start_date, end_date, price, amount) {
		this.Rec = &Recv{5, "此接口参数均不能为空", nil}
		return
	}

	var sql string = ps("insert into coupon (name,category,start_date,end_date,price,amount,unix) values ('%s','%s','%d','%d','%v','%d','%d')",
		name, category, start_date, end_date, price, amount, TimeNow)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		_, strerr := ChecSQLerr(err)
		log("添加优惠券失败:[%v]", err)
		this.Rec = &Recv{5, ps("添加优惠券失败:[%s]", strerr), nil}
		return
	}
	this.Rec = &Recv{3, "添加优惠券成功", nil}
}

// sid,id,name,category,start_date,end_date,price,amount
func (this *CouponController) CouponModify() {
	id, _ := this.GetInt64("id")
	name := this.GetString("name")
	category := this.GetString("category")
	start_date, _ := this.GetInt64("start_date")
	end_date, _ := this.GetInt64("end_date")
	price, _ := this.GetFloat("price")
	amount, _ := this.GetInt32("amount")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = "update coupon set "
	if name != "" {
		sql += ps("name='%s',", name)
	}
	if category != "" {
		sql += ps("category='%s',", category)
	}
	if start_date > 0 {
		sql += ps("start_date='%d',", start_date)
	}
	if end_date > 0 {
		sql += ps("end_date='%d',", end_date)
	}
	if price > 0.0 {
		sql += ps("price='%v',", price)
	}
	if amount > 0 {
		sql += ps("amount='%d',", amount)
	}

	sql += ps("unix='%d' where id=%d", TimeNow, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		_, strerr := ChecSQLerr(err)
		log("修改优惠券失败:[%v]", err)
		this.Rec = &Recv{5, ps("修改优惠券失败:[%s]", strerr), nil}
		return
	}
	this.Rec = &Recv{3, "修改优惠券成功", nil}
}

// sid,id
func (this *CouponController) CouponDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from coupon where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除优惠券[%d]失败:err[%v]", id, err)
		this.Rec = &Recv{5, "删除优惠券失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除优惠券成功", nil}
}

// sid,status(0-未使用;1-已使用(包括已过期)),begidx,counts
func (this *CouponController) IsscoupQuery() {
	status, _ := this.GetInt32("status")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	var sql, sqlc string = "", ""
	switch status {
	case 0:
		sql = ps("SELECT ic.id,c.name,c.category,c.price,ic.status,ic.amount,ic.unix FROM `issue_coupon` AS ic,`coupon` AS c WHERE ic.status=0 AND c.end_date>%d and ic.cnid=c.id ORDER BY ic.unix DESC limit %d,%d;",
			TimeNow, begidx, counts)
		sqlc = ps("SELECT count(ic.id) FROM `issue_coupon` AS ic,`coupon` AS c WHERE ic.status=0 AND c.end_date>%d and ic.cnid=c.id;", TimeNow)
	case 1:
		sql = ps("SELECT ic.id,c.name,c.category,c.price,ic.status,ic.amount,ic.unix FROM `issue_coupon` AS ic,`coupon` AS c WHERE ic.status=1 OR c.end_date<=%d and ic.cnid=c.id ORDER BY ic.unix DESC limit %d,%d;",
			TimeNow, begidx, counts)
		sqlc = ps("SELECT count(ic.id) FROM `issue_coupon` AS ic,`coupon` AS c WHERE ic.status=1 OR c.end_date<=%d and ic.cnid=c.id;", TimeNow)
	}

	db := orm.NewOrm()
	var result []orm.Params
	var total int
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询总数失败:err[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	} else {
		total, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询成功", &RecvEx{total, result}}
	return
}
