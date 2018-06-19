package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
	"strings"
	"time"
)

type AccountController struct {
	OnlineController
}

// sid,period(0-周,1-月,2-年)
func (this *AccountController) RegisterReport() {
	period, _ := this.GetInt32("period")

	//业务逻辑
	type CountData struct {
		Number int
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
			sql = ps("Select count(id) as num from `user` where unix<=%d and unix>%d;", year_unix[i], year_unix[i+1])
			_, err := db.Raw(sql).Values(&res)
			if err == nil && len(res) > 0 {
				if res[0]["num"] != nil {
					result[i].Number, _ = strconv.Atoi(res[0]["num"].(string))
				}
			}

			result[i].DatePd = year_unix[i]
		}
	case 1:
		today := time.Now().Format("2006-01")
		today_t, _ = time.ParseInLocation("2006-01", today, time.Local)

		// 连续10个月的时间值
		var month_unix []int64
		month_unix = make([]int64, 13)
		for i := 0; i < 13; i++ {
			month_unix[i] = today_t.AddDate(0, -1*(i-1), 0).Unix()
		}

		// 构造月统计数据
		result = make([]CountData, 12)
		for i := 0; i < 12; i++ {
			sql = ps("Select count(id) as num from `user` where unix<=%d and unix>%d;", month_unix[i], month_unix[i+1])

			_, err := db.Raw(sql).Values(&res)
			if err == nil && len(res) > 0 {
				if res[0]["num"] != nil {
					result[i].Number, _ = strconv.Atoi(res[0]["num"].(string))
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
			sql = ps("Select count(id) as num from `user` where unix<=%d and unix>%d;", week_unix[i], week_unix[i+1])

			_, err := db.Raw(sql).Values(&res)
			if err == nil && len(res) > 0 {
				if res[0]["num"] != nil {
					result[i].Number, _ = strconv.Atoi(res[0]["num"].(string))
				}
			}

			result[i].DatePd = week_unix[i+1]
		}
	}

	this.Rec = &Recv{5, "查询成功", result}
	return
}

// sid,pt_id,product_no(编号),phone(所有人手机号),classify(""-全部,eg.调度费,维护费,车辆维修费,电池报废等)
func (this *AccountController) ExpendQuery() {
	product_no := this.GetString("product_no")
	phone := this.GetString("phone")
	pt_id := this.GetString("pt_id")
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt64("counts")

	sql := ps("select ex.*,up.product_no,u.account from `expend` as ex,`user_product` as up,`enjoy_product` as ep,'user' as u where ex.up_id=up.id and up.ep_id=ep.id and up.user_id=u.id")
	sqlc := ps("select count(*) from `expend` as ex,`user_product` as up,`enjoy_product` as ep,'user' as u where ex.ep_id=up.id and up.ep_id=ep.id and up.user_id=u.id")
	if CheckArg(product_no) {
		sql += ps(" where up.product_no='%s'", product_no)
		sqlc += ps(" where up.product_no='%s'", product_no)
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

	if CheckArg(pt_id) {
		if strings.Contains(sql, "where") {
			sql += ps(" and ep.pt_id='%d'", pt_id)
			sqlc += ps(" and ep.pt_id='%d'", pt_id)
		} else {
			sql += ps(" where ep.pt_id='%d'", pt_id)
			sqlc += ps(" where ep.pt_id='%d'", pt_id)
		}
	}

	sql += ps(" order by ex.unix desc limit %d,%d;", begidx, counts)
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
	this.Rec = &Recv{3, "查询成功!", &RecvEx{total, result}}

	return
}
