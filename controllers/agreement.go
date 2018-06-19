package controllers

import (
	"github.com/astaxie/beego/orm"
	"path/filepath"
	"strconv"
	"strings"
)

type AgreementController struct {
	OnlineController
}

// sid,ep_id(订单id),text
func (this *AgreementController) AgreementUpdate() {
	ep_id, _ := this.GetInt32("ep_id")
	text := this.GetString("text")

	//检查参数
	if !CheckArg(ep_id) {
		this.Rec = &Recv{5, "订单id不能为空", nil}
		return
	}
	if !CheckArg(text) {
		this.Rec = &Recv{5, "合同正文为空", nil}
		return
	}
	text = strings.Replace(text, "'", "''", -1)

	sql := ps("select count(id) as num from `agreement` where ep_id='%d';", ep_id)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询合同出错:%s", err.Error())
		this.Rec = &Recv{5, "查询合同失败", nil}
		return
	} else {
		num, _ := strconv.Atoi(result[0]["num"].(string))
		if num > 0 {
			sql = ps("insert into `agreement` (ep_id,text,unix) values ('%d','%s','%d');", ep_id, text, TimeNow)
		} else {
			sql = ps("update `agreement` set text='%s',unix='%d' where ep_id=%d;", text, TimeNow, ep_id)
		}
	}

	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("添加合同失败:err[%v]", err)
		this.Rec = &Recv{5, "添加合同失败", nil}
		return
	}
	this.Rec = &Recv{3, "添加合同成功!", nil}
}

// sid,ep_id(订单id),realname,idnumber,pimg(正面照),nimg(反面照)
func (this *AgreementController) AgreementUpload() {
	ep_id, _ := this.GetInt32("ep_id")
	realname := this.GetString("realname")
	idnumber := this.GetString("idnumber")

	//检查参数
	if !CheckArg(ep_id) {
		this.Rec = &Recv{5, "订单id不能为空", nil}
		return
	}
	if !CheckArg(realname, idnumber) {
		this.Rec = &Recv{5, "身份信息不能为空", nil}
		return
	}

	var positive_img, negative_img string = "", ""
	f, h, err := this.GetFile("pimg")
	if f == nil { //空文件
		log("图片为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件失败:err[%v]", err)
		} else {
			// 保存位置在 static/personal
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("pimg", filepath.Join(conf("personalpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				positive_img = ps("https://%s/%s;", conf("personaldown"), filename)
			}
		}
	}

	f, h, err = this.GetFile("nimg")
	if f == nil { //空文件
		log("图片为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件失败:err[%v]", err)
		} else {
			// 保存位置在 static/personal
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("nimg", filepath.Join(conf("personalpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				negative_img = ps("https://%s/%s;", conf("personaldown"), filename)
			}
		}
	}

	sql := "update `agreement` set "
	if realname != "" {
		sql += ps("realname='%s',", realname)
	}
	if idnumber != "" {
		sql += ps("idnumber='%s',", idnumber)
	}
	if positive_img != "" {
		sql += ps("positive_img='%s',", positive_img)
	}
	if negative_img != "" {
		sql += ps("negative_img='%s',", negative_img)
	}

	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("上传合同失败:err[%v]", err)
		this.Rec = &Recv{5, "上传合同失败", nil}
		return
	}
	this.Rec = &Recv{3, "上传合同成功!", nil}
}

// sid,realname,idnumber,pimg(正面照),nimg(反面照)
func (this *AgreementController) UserInfoAuth() {
	realname := this.GetString("realname")
	idnumber := this.GetString("idnumber")

	//检查参数
	if !CheckArg(realname, idnumber) {
		this.Rec = &Recv{5, "身份信息不能为空", nil}
		return
	}

	var positive_img, negative_img string = "", ""
	f, h, err := this.GetFile("pimg")
	if f == nil { //空文件
		log("图片为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件失败:err[%v]", err)
		} else {
			// 保存位置在 static/personal
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("pimg", filepath.Join(conf("personalpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				positive_img = ps("https://%s/%s", conf("personaldown"), filename)
			}
		}
	}

	f, h, err = this.GetFile("nimg")
	if f == nil { //空文件
		log("图片为空")
	} else {
		defer f.Close()
		if err != nil {
			log("上传文件失败:err[%v]", err)
		} else {
			// 保存位置在 static/personal
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("nimg", filepath.Join(conf("personalpath"), filename))
			if err != nil {
				log("文件保存失败:err[%v]", err)
			} else {
				negative_img = ps("https://%s/%s", conf("personaldown"), filename)
			}
		}
	}

	sql := "update `user` set "
	if realname != "" {
		sql += ps("realname='%s',", realname)
	}
	if idnumber != "" {
		sql += ps("idnumber='%s',", idnumber)
	}
	if positive_img != "" {
		sql += ps("positive_img='%s',", positive_img)
	}
	if negative_img != "" {
		sql += ps("negative_img='%s',", negative_img)
	}

	sql += ps("verify_status=1,verify_deadline=0 where id=%d", this.User.UserId)

	//log("%s", sql)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("认证信息上传失败:err[%v]", err)
		this.Rec = &Recv{5, "认证信息上传失败", nil}
		return
	}
	this.Rec = &Recv{3, "认证信息上传成功!", nil}
}

func (this *AgreementController) UserAuthInfoQuery() {
	sql := ps("select realname,idnumber,positive_img,negative_img,verify_status,verify_reason,bankcard from user where id=%d", this.User.UserId)
	var result []orm.Params

	db := orm.NewOrm()
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询认证信息失败:%s", err.Error())
		this.Rec = &Recv{5, "查询认证信息失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询认证信息成功", result[0]}
}

// sid,id(订单id)
func (this *AgreementController) AgreementDown() {
	id, _ := this.GetInt64("id")

	if !CheckArg(id) {
		this.Rec = &Recv{5, "订单id不能为空", nil}
		return
	}

	sql := ps("select * from `agreement` where ep_id='%d';", id)
	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询合同出错:%s", err.Error())
		this.Rec = &Recv{5, "查询合同失败", nil}
		return
	}

	this.Rec = &Recv{3, "查询成功", result}
	return
}
