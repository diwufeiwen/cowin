package controllers

import (
	"github.com/astaxie/beego/orm"
	"strconv"
)

type LetterController struct {
	OnlineController
}

// sid,title,content,recv_uid(接收方账号)
func (this *LetterController) LetterSend() {
	title := this.GetString("title")
	content := this.GetString("content")
	recv_uid, _ := this.GetInt32("recv_uid")

	// 检查参数
	if !CheckArg(recv_uid) {
		this.Rec = &Recv{5, "接收方id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql string = ps("SELECT count(id) AS num from `user` WHERE id=%d;", recv_uid)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		_, str := ChecSQLerr(err)
		log("查询用户失败:err[%s]", str)
		this.Rec = &Recv{5, ps("发送失败:[%s]", str), nil}
		return
	} else {
		num, _ := strconv.Atoi(result[0]["num"].(string))
		if num <= 0 {
			this.Rec = &Recv{5, "接收用户id不存在", nil}
			return
		}
	}

	sql = ps("insert into `letter` (title,content,send_uid,recv_uid,unix) values ('%s','%s','%d','%d','%d')", title, content, this.User.UserId, recv_uid, TimeNow)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		_, str := ChecSQLerr(err)
		log("发送失败[%s]", str)
		this.Rec = &Recv{5, "发送失败", nil}
		return
	}

	this.Rec = &Recv{3, "发送成功", nil}
	return
}

// sid,status(0-未读,1-已读),begidx,counts
func (this *LetterController) LetterQuery() {
	begidx, _ := this.GetInt64("begidx")
	counts, _ := this.GetInt32("counts")
	status, _ := this.GetInt32("status")
	//检查参数
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	//业务逻辑
	db := orm.NewOrm()
	var result []orm.Params
	var (
		sql    string
		sqlc   string
		totals int = 0
	)

	if status < 0 {
		sql = ps("select * from `letter` where recv_uid=%d order by unix desc limit %d,%d;", this.User.UserId, begidx, counts)
		sqlc = ps("select count(id) as num from `letter` where  recv_uid=%d;", this.User.UserId)
	} else {
		sql = ps("select * from `letter` where recv_uid=%d and status=%d order by unix desc limit %d,%d;", this.User.UserId, status, begidx, counts)
		sqlc = ps("select count(id) as num from `letter` where  recv_uid=%d and status=%d;", this.User.UserId, status)
	}

	_, err := db.Raw(sqlc).Values(&result)
	if err == nil {
		totals, _ = strconv.Atoi(result[0]["num"].(string))
	}

	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("查询失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询失败"), nil}
		return
	}

	for _, item := range result {
		id, _ := strconv.Atoi(item["id"].(string))
		sql = ps("update letter set status='1' where id='%d';", id)
		_, err = db.Raw(sql).Exec()
		if err != nil {
			log("更新状态失败:%v", err)
		}
	}

	type RecvEx struct {
		Total  int
		Detail interface{}
	}
	this.Rec = &Recv{3, ps("查询成功!"), &RecvEx{totals, result}}
}
