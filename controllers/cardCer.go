package controllers

import (
	"net/http"
	"net/url"
	"io/ioutil"
	"encoding/json"
	"github.com/astaxie/beego/orm"
)

type CardCertificationController struct {
	OnlineController
}

func (this *CardCertificationController) CheckCardCA() {

	bankcard := this.GetString("bankcard")
	realName := this.GetString("realName")
	cardNo := this.GetString("cardNo")

	//检查参数
	if !CheckArg(bankcard, realName, cardNo) {
		this.Rec = &Recv{5, "此接口参数均不能为空", nil}
		return
	}


	form := url.Values{}
	form.Set("key", "3e58c48caf73f9a4a58faab5b30229f4")
	form.Set("bankcard", bankcard)
	form.Set("realName", realName)
	form.Set("cardNo", cardNo)
	form.Set("cardtype", "CD")
	form.Set("information", "1")

	res, err := http.Get(ps("http://v.apistore.cn/api/bank/v3/?%s", form.Encode()))
	if err != nil {
		this.Rec = &Recv{5, "服务器异常:" + err.Error(), nil}
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		this.Rec = &Recv{5, "服务器异常:" + err.Error(), nil}
		return
	}

	type RetBody struct {
		ErrorCode int         `json:"error_code"`
		Reason    string      `json:"reason"`
		Result    interface{} `json:"result"`
	}

	//解析body
	var rBody RetBody
	err = json.Unmarshal(body, &rBody)
	if err != nil {
		log("body:%s", string(body))
		this.Rec = &Recv{5, "服务器异常:" + err.Error(), nil}
		return
	}

	if rBody.ErrorCode != 0 {
		this.Rec = &Recv{5, rBody.Reason, nil}
		return
	}

	db := orm.NewOrm()
	var result []orm.Params
	//更新user表
	sql := ps("update `user` set realname='%s',idnumber='%s',bankcard='%s',verify_status='3' where id=%d;",
		 realName, cardNo,bankcard, this.User.UserId)

	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("认证信息更新失败:%s", err.Error())
		this.Rec = &Recv{5, "审核认证信息失败", nil}
		return
	}

	// 对于待认证的合同写入认证信息
	_, err = db.Raw("select realname,idnumber,positive_img,negative_img from `user` where id=?", this.User.UserId).Values(&result)
	if err != nil {
		log("认证信息更新失败:%s", err.Error())
		this.Rec = &Recv{5, "查询认证信息失败", nil}
		return
	}

	sql = ps("update `agreement` set text='%s',realname='%s',idnumber='%s',positive_img='%s',negative_img='%s',unix='%d',status=2 where status=1;",
		"", realName, cardNo, result[0]["positive_img"].(string), result[0]["negative_img"].(string), TimeNow)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("签署待认证合同失败:%s", err.Error())
		this.Rec = &Recv{5, "签署待认证合同失败", nil}
		return
	}

	sql = ps("update `user` set verify_status='3',verify_deadline='0' where id='%d';", this.User.UserId)
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("审核认证信息失败:%s", err.Error())
		this.Rec = &Recv{5, "审核认证信息失败", nil}
		return
	}
	this.Rec = &Recv{3, rBody.Reason, nil}
	return
}
