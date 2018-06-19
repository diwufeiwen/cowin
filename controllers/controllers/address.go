package controllers

import (
	"github.com/astaxie/beego/orm"
)

type AddressController struct {
	OnlineController
}

// sid,recver,address,phone,default(1-默认地址,0-一般地址)
func (this *AddressController) ShipAddrAdd() {
	recver := this.GetString("recver")
	address := this.GetString("address")
	phone := this.GetString("phone")
	defaults, _ := this.GetInt32("default")

	//检查参数
	if !CheckArg(recver, address, phone) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	var sql string
	db := orm.NewOrm()
	if defaults == 1 {
		sql = ps("update `ship_address` set `default`=0 where uid=%d;", this.User.UserId)
		_, err := db.Raw(sql).Exec()
		if err != nil {
			log("更改默认地址失败:[%v]", err)
			this.Rec = &Recv{5, "添加收货地址失败", nil}
			return
		}
	}
	ps("insert into `ship_address` (uid,recver,address,phone,`default`,unix) values ('%d','%s','%s','%s','%d','%d');",
		this.User.UserId, recver, address, phone, defaults, TimeNow)

	sql = ps("insert into `ship_address` (uid,recver,address,phone,`default`,unix) values ('%d','%s','%s','%s','%d','%d');",
		this.User.UserId, recver, address, phone, defaults, TimeNow)

	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("添加收货地址失败:[%v]", err)
		this.Rec = &Recv{5, "添加收货地址失败", nil}
		return
	}

	this.Rec = &Recv{3, "添加收货地址成功", nil}
}

// sid,id,recver,address,phone,default
func (this *AddressController) ShipAddrModify() {
	id, _ := this.GetInt("id")
	recver := this.GetString("recver")
	address := this.GetString("address")
	phone := this.GetString("phone")
	defaults, _ := this.GetInt32("default")

	//检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	//业务逻辑
	var sql = "update ship_address set "
	if recver != "" {
		sql += ps("recver='%s',", recver)
	}
	if address != "" {
		sql += ps("address='%s',", address)
	}
	if phone != "" {
		sql += ps("phone='%s',", phone)
	}
	if defaults > 0 {
		sql += ps("`default`='%d',", defaults)
	}

	sql += ps("unix='%d' where id=%d", TimeNow, id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("修改收货地址失败:[%v]", err)
		this.Rec = &Recv{5, "修改收货地址失败", nil}
		return
	}
	this.Rec = &Recv{3, "修改收货地址成功", nil}
}

// sid,id
func (this *AddressController) ShipAddrDel() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("delete from ship_address where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("删除收货地址失败:[%v]", err)
		this.Rec = &Recv{5, "删除收货地址失败", nil}
		return
	}
	this.Rec = &Recv{3, "删除收货地址成功", nil}
	return
}

// sid,id
func (this *AddressController) ShipAddrDefault() {
	id, _ := this.GetInt64("id")

	// 检查参数
	if !CheckArg(id) {
		this.Rec = &Recv{5, "id不能为空", nil}
		return
	}

	// 业务逻辑
	var sql = ps("update ship_address set `default`=1 where id='%d';", id)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("设置默认地址失败:[%v]", err)
		this.Rec = &Recv{5, "设置默认地址失败", nil}
		return
	}

	_, err = db.Raw("update ship_address set `default`=0 where id!=? and uid=?;", id, this.User.UserId).Exec()
	if err != nil {
		log("取消默认设置失败:[%v]", err)
		this.Rec = &Recv{5, "设置默认地址失败", nil}
		return
	}

	this.Rec = &Recv{3, "设置默认地址成功", nil}
	return
}

// sid
func (this *AddressController) ShipAddrQuery() {
	sql := ps("select * from ship_address where uid=%d;", this.User.UserId)
	var result []orm.Params
	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询收货地址失败:err[%v]", err)
		this.Rec = &Recv{5, ps("查询收货地址失败"), nil}
		return
	}

	this.Rec = &Recv{3, ps("查询收货地址成功!"), result}
}
