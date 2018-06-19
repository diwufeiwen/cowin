package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
)

type PrfratioController struct {
	OnlineController
}

// sid,pt_id,put_month,ope_ratio,user_ratio,tax_ratio,plat_ratio
func (this *PrfratioController) PrfratioAdd() {
	pt_id, _ := this.GetInt32("pt_id")
	put_month := this.GetString("put_month")
	ope_ratio, _ := this.GetFloat("ope_ratio", 32)
	user_ratio, _ := this.GetFloat("user_ratio", 32)
	tax_ratio, _ := this.GetFloat("tax_ratio", 32)
	plat_ratio, _ := this.GetFloat("plat_ratio", 32)

	//检查参数
	if !CheckArg(pt_id, put_month, ope_ratio, user_ratio, tax_ratio, plat_ratio) {
		this.Rec = &Recv{5, "此接口参数均不能为空", nil}
		return
	}

	if ope_ratio > 1.0 || user_ratio > 1.0 || tax_ratio > 1.0 || plat_ratio > 1.0 {
		this.Rec = &Recv{5, "收益百分比转为小数不能大于1", nil}
		return
	}

	// 业务逻辑
	db := orm.NewOrm()
	sql := ps("insert into `profit_ratio` (pt_id,put_month,ope_ratio,user_ratio,tax_ratio,plat_ratio,unix) values('%d','%s','%v','%v','%v','%v','%d');",
		pt_id, put_month, ope_ratio, user_ratio, tax_ratio, plat_ratio, TimeNow)
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加利息分配行出错:[%v]", err)
		this.Rec = &Recv{5, "添加失败", nil}
		return
	}

	// 重新加载利率
	ReadInterestRate()

	this.Rec = &Recv{3, "添加成功", nil}
	return
}

// sid,id,pt_id,put_month,ope_ratio,user_ratio,tax_ratio,plat_ratio
func (this *PrfratioController) PrfratioModify() {
	id, _ := this.GetInt64("id")
	pt_id, _ := this.GetInt("pt_id", 32)
	put_month := this.GetString("put_month")
	ope_ratio, _ := this.GetFloat("ope_ratio", 32)
	user_ratio, _ := this.GetFloat("user_ratio", 32)
	tax_ratio, _ := this.GetFloat("tax_ratio", 32)
	plat_ratio, _ := this.GetFloat("plat_ratio", 32)

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string = ps("UPDATE `profit_ratio` set operate='1',unix='%d' where id='%d';", TimeNow, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改利息分配行[%d]出错:[%v]", id, err)
		this.Rec = &Recv{5, "修改失败", nil}
		return
	}

	sql = ps("select pt_id,put_month,ope_ratio,user_ratio,tax_ratio,plat_ratio from `profit_ratio` where id='%d';", id)
	var result []orm.Params
	nums, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询利息分配行[%d]出错:[%v]", id, err)
		this.Rec = &Recv{5, "修改失败", nil}
		return
	}

	if nums > 0 {
		if pt_id <= 0 {
			pt_id, _ = strconv.Atoi(result[0]["pt_id"].(string))
		}
		if put_month == "" {
			put_month = result[0]["put_month"].(string)
		}
		if ope_ratio <= 0.00 {
			ope_ratio, _ = strconv.ParseFloat(result[0]["ope_ratio"].(string), 32)
		}
		if user_ratio <= 0.00 {
			user_ratio, _ = strconv.ParseFloat(result[0]["user_ratio"].(string), 32)
		}
		if tax_ratio <= 0.00 {
			tax_ratio, _ = strconv.ParseFloat(result[0]["tax_ratio"].(string), 32)
		}
		if plat_ratio <= 0.00 {
			plat_ratio, _ = strconv.ParseFloat(result[0]["plat_ratio"].(string), 32)
		}
		sql = ps("insert into `profit_ratio` (pt_id,put_month,ope_ratio,user_ratio,tax_ratio,plat_ratio,unix) values('%d','%s','%v','%v','%v','%v','%d');",
			pt_id, put_month, ope_ratio, user_ratio, tax_ratio, plat_ratio, TimeNow)
		_, err = db.Raw(sql).Exec()
		if err != nil {
			log("添加利息分配行[%d]出错:[%v]", id, err)
			this.Rec = &Recv{5, "修改失败", nil}
			return
		}
	}

	// 重新加载利率
	ReadInterestRate()

	this.Rec = &Recv{3, "修改成功", nil}
	return
}

// sid,id
func (this *PrfratioController) PrfratioDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string = ps("UPDATE `profit_ratio` set operate='2',unix='%d' where id=%d;", TimeNow, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改利息分配行[%d]出错:[%v]", id, err)
		this.Rec = &Recv{5, "删除失败", nil}
		return
	}

	// 重新加载利率
	ReadInterestRate()

	this.Rec = &Recv{3, "删除成功", nil}
	return
}

// sid,operate(0-最新,1-历史),pt_id
func (this *PrfratioController) PrfratioQuery() {
	operate, _ := this.GetInt("operate")
	pt_id, _ := this.GetInt("pt_id")

	// 业务逻辑
	var sql string
	switch operate {
	case 0:
		sql = ps("SELECT * from `profit_ratio` where operate=0")
	case 1:
		sql = ps("SELECT * from `profit_ratio` where operate>0")
	}

	if pt_id > 0 {
		sql += ps(" and pt_id=%d", pt_id)
	}
	sql += " order by unix desc;"

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询利息比例表出错:[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}
