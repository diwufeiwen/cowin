package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
	"strings"
	"time"
)

type IncomeController struct {
	OnlineController
}

// sid,pt_id(产品类型id),period(0-周,1-月,2-年)
func (this *IncomeController) IncomeReport() {
	// 身份判断
	if this.User.Flag != 8 && this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("无此访问权限"), nil}
		return
	}

	period, _ := this.GetInt32("period")
	pt_id, _ := this.GetInt32("pt_id")

	//业务逻辑
	type CountData struct {
		Expend float64
		DatePd int64
	}

	var result []CountData
	var sql string = ""
	var res []orm.Params
	db := orm.NewOrm()

	var today_t time.Time
	switch period {
	case 2:
		today := time.Now().Format("2006-")
		today = today + "12-31"
		today_t, _ = time.ParseInLocation("2006-01-02", today, time.Local)

		// 连续5年的时间值
		var year_unix []int64
		year_unix = make([]int64, 6)
		year_unix[0] = today_t.Unix()
		for i := 1; i < 6; i++ {
			year_unix[i] = today_t.AddDate(-1*i, 0, 0).Unix()
		}

		// 构造年统计数据
		result = make([]CountData, 5)
		for i := 0; i < 5; i++ {
			if pt_id > 0 {
				sql = ps("Select sum(ic.total_fee) as fee from `income` AS ic,`user_product` AS up,`enjoy_product` AS ep where ic.up_id=up.id and up.ep_id=ep.id and ep.pt_id='%d' and ic.unix<=%d and ic.unix>%d;",
					pt_id, year_unix[i], year_unix[i+1])
			} else {
				sql = ps("Select sum(total_fee) as fee from `income` where unix<=%d and unix>%d;", year_unix[i], year_unix[i+1])
			}

			_, err := db.Raw(sql).Values(&res)
			if err == nil && len(res) > 0 {
				if res[0]["fee"] != nil {
					result[i].Expend, _ = strconv.ParseFloat(res[0]["fee"].(string), 64)
				}
			}

			result[i].DatePd = year_unix[i]
		}
	case 1:
		today := time.Now().Format("2006-01")
		today_t, _ = time.ParseInLocation("2006-01", today, time.Local)

		// 连续12个月的时间值
		var month_unix []int64
		month_unix = make([]int64, 13)
		for i := 0; i < 13; i++ {
			month_unix[i] = today_t.AddDate(0, -1*(i-1), 0).Unix()
		}

		// 构造月统计数据
		result = make([]CountData, 12)
		for i := 0; i < 12; i++ {
			if pt_id > 0 {
				sql = ps("Select sum(ic.total_fee) as fee from `income` AS ic,`user_product` AS up,`enjoy_product` AS ep where ic.up_id=up.id and up.ep_id=ep.id and ep.pt_id='%d' and ic.unix<=%d and ic.unix>%d;",
					pt_id, month_unix[i], month_unix[i+1])
			} else {
				sql = ps("Select sum(total_fee) as fee from `income` where unix<=%d and unix>%d;", month_unix[i], month_unix[i+1])
			}

			_, err := db.Raw(sql).Values(&res)
			if err == nil && len(res) > 0 {
				if res[0]["fee"] != nil {
					result[i].Expend, _ = strconv.ParseFloat(res[0]["fee"].(string), 64)
				}
			}

			result[i].DatePd = month_unix[i+1]
		}
	case 0:
		// 以每周一作为分界点
		today := time.Now().Format("2006-01-02")
		today_t, _ = time.ParseInLocation("2006-01-02", today, time.Local)

		// 连续10个周的时间值
		var week_unix []int64
		week_unix = make([]int64, 11)
		wd := time.Now().Weekday()
		today_t = today_t.AddDate(0, 0, 8-int(wd))
		week_unix[0] = today_t.Unix()
		for i := 1; i < 11; i++ {
			week_unix[i] = today_t.AddDate(0, 0, -7*i).Unix()
		}

		// 构造周统计数据
		result = make([]CountData, 10)
		for i := 0; i < 10; i++ {
			if pt_id > 0 {
				sql = ps("Select sum(ic.total_fee) as fee from `income` AS ic,`user_product` AS up,`enjoy_product` AS ep where ic.up_id=up.id and up.ep_id=ep.id and ep.pt_id='%d' and ic.unix<=%d and ic.unix>%d;",
					pt_id, week_unix[i], week_unix[i+1])
			} else {
				sql = ps("Select sum(total_fee) as fee from `income` where unix<=%d and unix>%d;", week_unix[i], week_unix[i+1])
			}

			_, err := db.Raw(sql).Values(&res)
			if err == nil && len(res) > 0 {
				if res[0]["fee"] != nil {
					result[i].Expend, _ = strconv.ParseFloat(res[0]["fee"].(string), 64)
				}
			}

			result[i].DatePd = week_unix[i+1]
		}
	}

	this.Rec = &Recv{5, "查询成功", result}
	return
}

// sid,pt_id(产品类型id),product_no(产品编号),phone(所有人手机号),begidx,counts
func (this *IncomeController) IncomeQuery() {
	// 身份判断
	if this.User.Flag != 8 && this.User.Flag != 1 {
		this.Rec = &Recv{5, ps("无此访问权限"), nil}
		return
	}

	pt_id, _ := this.GetInt64("pt_id")
	product_no := this.GetString("product_no")
	phone := this.GetString("phone")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	// 参数检测
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "请求总数不能为空", nil}
		return
	}

	var sql, sqlc string = "", ""
	if pt_id > 0 {
		sql = ps("select ep.pt_id,im.*,up.product_no,u.account as owner_phone,u.nick as owner_nick from `income` as im,`user_product` as up,`enjoy_product` as ep,`user` as u where up.ep_id=ep.id and im.up_id=up.id and up.user_id=u.id and ep.pt_id=%d", pt_id)
		sqlc = ps("select count(*) from `income` as im,`user_product` as up,`enjoy_product` as ep,`user` as u where up.ep_id=ep.id and im.up_id=up.id and up.user_id=u.id and ep.pt_id=%d", pt_id)
	} else {
		sql = ps("select ep.pt_id,im.*,up.product_no,u.account as owner_phone,u.nick as owner_nick from `income` as im,`user_product` as up,`enjoy_product` as ep,`user` as u where up.ep_id=ep.id and im.up_id=up.id and up.user_id=u.id")
		sqlc = ps("select count(*) from `income` as im,`user_product` as up,`enjoy_product` as ep,`user` as u where im.up_id=up.id and up.ep_id=ep.id and ep.user_id=u.id")
	}

	if CheckArg(phone) {
		if strings.Contains(sql, "where") {
			sql += ps(" and u.account='%s'", phone)
			sqlc += ps(" and u.account='%s'", phone)
		} else {
			sql += ps(" where u.account='%s'", phone)
			sqlc += ps(" where u.account='%s'", phone)
		}
	}

	if CheckArg(product_no) {
		if strings.Contains(sql, "where") {
			sql += ps(" and up.product_no='%s'", product_no)
			sqlc += ps(" and up.product_no='%s'", product_no)
		} else {
			sql += ps(" where up.product_no='%s'", product_no)
			sqlc += ps(" where up.product_no='%s'", product_no)
		}
	}

	sql += ps(" order by im.unix desc limit %d,%d;", begidx, counts)
	sqlc += ";"

	var total int = 0
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询总数失败", nil}
		return
	} else {
		if result[0]["num"] != nil {
			total, _ = strconv.Atoi(result[0]["num"].(string))
		}
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}

	for idx := range result {
		item := result[idx]
		if item["user_id"] != nil {
			sql = ps("select account,nick from `user` where id='%s';", item["user_id"].(string))
			var res []orm.Params
			_, err := db.Raw(sql).Values(&res)
			if err == nil {
				if len(res) > 0 {
					item["user_nick"] = res[0]["nick"].(string)
					item["user_acc"] = res[0]["account"].(string)
				}
			}
		}
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询成功!", &RecvEx{total, result}}

	return
}

// sid
func (this *IncomeController) IncomeBrief() {
	sql := ps("select sum(im.owner_income) as income from `income` as im,`user_product` as up where im.up_id=up.id and up.user_id='%d';", this.User.UserId)
	var totalIncome, dayIncome float64
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		_, str := ChecSQLerr(err)
		log("查询失败:err[%s]", str)
		this.Rec = &Recv{5, ps("查询失败:[%s]", str), nil}
		return
	} else {
		if result[0]["income"] != nil {
			totalIncome, _ = strconv.ParseFloat(result[0]["income"].(string), 64)
		}
	}

	today := time.Now().Format("2006-01-02")
	var today_t time.Time
	today_t, _ = time.ParseInLocation("2006-01-02", today, time.Local)
	today_unix := today_t.Unix()
	dt, _ := time.ParseDuration("-24h")
	yesterday_unix := today_t.Add(dt).Unix()
	sql = ps("select sum(im.owner_income) as income from `income` as im,`user_product` as up where im.up_id=up.id and up.user_id='%d' and im.unix<%d and im.unix>=%d;",
		this.User.UserId, today_unix, yesterday_unix)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		_, str := ChecSQLerr(err)
		log("查询失败:err[%s]", str)
		this.Rec = &Recv{5, ps("查询失败:[%s]", str), nil}
		return
	} else {
		if result[0]["income"] != nil {
			dayIncome, _ = strconv.ParseFloat(result[0]["income"].(string), 64)
		}
	}

	type RecvEx struct {
		TotalIncome float64
		DayIncome   float64
	}
	this.Rec = &Recv{3, "查询成功!", &RecvEx{totalIncome, dayIncome}}

	return
}

// sid,pt_id,days(几天内)
func (this *IncomeController) IncomeTotalQuery() {
	pt_id, _ := this.GetInt32("pt_id")
	days, _ := this.GetInt("days")

	today := time.Now().Format("2006-01-02")
	today_t, _ := time.ParseInLocation("2006-01-02", today, time.Local)
	beg_unix := today_t.AddDate(0, 0, -1*days).Unix()
	var sql string = ""
	if pt_id <= 0 {
		sql = ps("select sum(im.owner_income) as income from `income` as im,`user_product` as up where im.up_id=up.id and up.user_id='%d' and im.unix>=%d;", this.User.UserId, beg_unix)
	} else {
		sql = ps("select sum(im.owner_income) as income from `income` as im,`user_product` as up,`enjoy_product` as ep where im.up_id=up.id and up.ep_id=ep.id and up.user_id='%d' and ep.pt_id=%d and im.unix>=%d;",
			this.User.UserId, pt_id, beg_unix)
	}

	var totalIncome float64
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		_, str := ChecSQLerr(err)
		log("查询失败:err[%s]", str)
		this.Rec = &Recv{5, ps("查询失败:[%s]", str), nil}
		return
	} else {
		totalIncome, _ = strconv.ParseFloat(result[0]["income"].(string), 64)
	}

	this.Rec = &Recv{3, "查询成功!", &totalIncome}

	return
}

// sid,begidx,counts
func (this *IncomeController) IncomeDetailQuery() {
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")

	var sql string = ""
	sql = ps("select im.id,im.owner_income,im.use_time,im.unix,u.nick,up.product_no from `income` as im,`user_product` as up,`user` as u where im.up_id=up.id and im.user_id=u.id and up.user_id='%d' order by im.unix desc limit %d,%d;",
		this.User.UserId, begidx, counts)
	sqlc := ps("select count(im.id) as num from `income` as im,`user_product` as up where im.up_id=up.id and up.user_id='%d';", this.User.UserId)

	var totals int
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sqlc).Values(&result)
	if err != nil {
		_, str := ChecSQLerr(err)
		log("查询失败:err[%s]", str)
		this.Rec = &Recv{5, ps("查询失败:[%s]", str), nil}
		return
	} else {
		totals, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		_, str := ChecSQLerr(err)
		log("查询失败:err[%s]", str)
		this.Rec = &Recv{5, ps("查询失败:[%s]", str), nil}
		return
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, "查询成功!", &RecvEx{totals, result}}
	return
}
