package controllers

import (
	"github.com/astaxie/beego/orm"
	"time"
	"strconv"
	"path/filepath"
)

type RechargeController struct {
	OnlineController
}

func (this *RechargeController) RechargeGold() {
	pay_method, _ := this.GetInt("pay_method")
	total_amount, _ := this.GetFloat("amount")
	if total_amount < 0 {
		this.Rec = &Recv{5, "充值金额不正确", nil}
	}

	// 参数检测
	switch pay_method {
	case 1:

		goto alipay

	default:
		this.Rec = &Recv{5, "充值方式异常", nil}
	}

alipay:
	//添加到数据库
	out_trade_no := ps("ZFB_%s_%s", time.Now().Format("20060102150405"), GetRandomString(5))
	sql := ps("insert `order_recharger` (recd,amount,pay_type,result,unix,account) values('%v','%v','%v','%v','%v','%v');", out_trade_no, total_amount, 1, 1, TimeNow, this.User.Account)
	db := orm.NewOrm()
	_, err := db.Raw(sql).Exec()
	if err != nil {
		log("提交充值订单失败:[%v]", err)
		this.Rec = &Recv{5, "提交充值订单失败", nil}

	}
	request, _ := CreateAlipayOrder(this.User.Platform, out_trade_no, total_amount, 1, "")
	if request != "" && err == nil {
		this.Rec = &Recv{3, "充值订单处理成功,请用支付宝在15分钟内完成支付", request}
	} else {
		log("支付宝订单处理错误:%s", err.Error())
		this.Rec = &Recv{5, "订单处理失败", nil}
	}

	return
}

func (this *RechargeController) QuaryRechargeHistory() {
	begidx, _ := this.GetInt32("begidx")
	counts, _ := this.GetInt32("counts")
	reType, _ := this.GetInt32("type") //0-所有历史订单;1-支付成功；2-未支付；3-支付失败
	//检查参数
	if !CheckArg(counts) {
		this.Rec = &Recv{5, "总数不能为空", nil}
		return
	}

	var sql string
	switch reType {
	case 0:
		sql = ps("select * from `order_recharger` order by unix desc limit %d,%d;", begidx, counts)
	case 1:
		sql = ps("select * from `order_recharger` where `result`='0' order by unix desc limit %d,%d;", begidx, counts)
	case 2:
		sql = ps("select * from `order_recharger` where `result`='1' order by unix desc limit %d,%d;", begidx, counts)
	case 3:
		sql = ps("select * from `order_recharger` where `result`='2' order by unix desc limit %d,%d;", begidx, counts)
	default:
		this.Rec = &Recv{5, "查询类型不存在", nil}
		return
	}

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询充值订单失败：[%v]", err.Error())
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}
	this.Rec = &Recv{3, "成功", result}
	return
}

func (this *RechargeController) QuaryRechargeByCode() {

	reCode := this.GetString("recode")
	//检查参数
	if !CheckArg(reCode) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	var sql string
	sql = ps("select * from `order_recharger` where `recd`='%v' ", reCode)

	db := orm.NewOrm()
	var result []orm.Params
	_, err := db.Raw(sql).Values(&result)
	if err != nil {
		log("查询充值订单失败：[%v]", err.Error())
		this.Rec = &Recv{5, "查询失败", nil}
		return
	}
	this.Rec = &Recv{3, "成功", result}
	return
}


//提现
func (this *RechargeController)  WithDraw(){
	money , _ := this.GetFloat("money")
	realname := this.GetString("realname")
	bankno := this.GetString("bankno")

	//检查参数
	if !CheckArg(money , realname , bankno) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	if money < 1 {
		this.Rec = &Recv{5, "提现金额不能小于1元", nil}
		return
	}
	
	var result []orm.Params
	db := orm.NewOrm()
	sql := ps("select * from `user` where `id`='%d' ", this.User.UserId)

	_ , err := db.Raw(sql).Values(&result)
	if err != nil {
		log("提现查询user表错误：[%v]", err.Error())
		this.Rec = &Recv{5, "提现申请失败", nil}
		return
	}

	if len(result)==0{
		log("提现查询user表错误：查无此人")
		this.Rec = &Recv{5, "提现申请失败", nil}
		return
	}

	ver_states , _ := strconv.Atoi(result[0]["verify_status"].(string))

	if ver_states !=3 {
		this.Rec = &Recv{5, ps("[%s]用户未实名认证", this.User.Account), nil}
		return
	}

	canWallet := GetCanWallet(this.User.Wallet , this.User.UserId , this.User.Account)
	id , _ := strconv.Atoi(result[0]["id"].(string))
	phone := result[0]["account"].(string)
	nick := result[0]["nick"].(string)
	name := result[0]["realname"].(string)
	bankcard := result[0]["bankcard"].(string)
	wallet, _ := strconv.ParseFloat(result[0]["wallet"].(string), 64)

	if realname != name || bankcard != bankno {
		this.Rec = &Recv{5, ps("[%s]用户实名认证信息不匹配", this.User.Account), nil}
		return
	}

	if money > canWallet {
		this.Rec = &Recv{5, "提现金额不能大于可提余额", nil}
		return
	}

	sql = ps("insert `withdraw` (phone,nick,realname,bankno,wallet,canwallet,pick_wallet,userid,unix) values('%s','%s','%s','%s','%v','%v','%v','%v','%v');",
		phone, nick, name, bankcard, wallet, canWallet , money , id , TimeNow)
	_ , err = db.Raw(sql).Exec()
	if err != nil {
		log("提现插入withdraw表错误：[%v]", err.Error())
		this.Rec = &Recv{5, "提现申请失败", nil}
		return
	}

	//扣除用户余额
	sql = ps("update `user` set wallet='%v' where id=%d;", this.User.Wallet-money, this.User.UserId)
	_, err = db.Raw(sql).Values(&result)
	if err != nil {
		log("提现申请扣除金额失败:[%v]", err)
		this.Rec = &Recv{5, "提现申请失败", nil}
		return
	}
	this.User.Wallet -= money

	//插入资金变动流水
	AddMoneyFlow("商城提现", "", this.User.Wallet, -money, this.User.UserId, "余额支付")


	this.Rec = &Recv{3, ps("[%s]提现申请成功,七天内到账", this.User.Account), nil}
}


//查询提现
func (this *RechargeController)  QueryWithDraw(){
	var result []orm.Params
	db := orm.NewOrm()
	sql := "select * from `withdraw`"

	_ , err := db.Raw(sql).Values(&result)

	if err != nil {
		log("提现查询withdraw表错误：[%v]", err.Error())
		this.Rec = &Recv{5, "提现查询失败", nil}
		return
	}

	this.Rec = &Recv{3, "提现查询成功", result}
}


//提现确认
func (this *RechargeController) ConfirmWithDraw(){

	remark := this.GetString("remark")
	id, _ := this.GetInt32("id")

	//检查参数
	if !CheckArg(remark , id) {
		this.Rec = &Recv{5, "参数存在空值", nil}
		return
	}

	var imgurl string = ""
	f, h, err := this.GetFile("file")
	if f != nil {
		defer f.Close()
		if err != nil {
			log("凭证文件上传失败:err[%v]", err)
			this.Rec = &Recv{5, "凭证文件上传失败,请重新尝试", nil}
		} else {
			// 保存位置在 static/tmpdown,没有文件夹要先创建
			filename := GetSid()
			filename += filepath.Ext(h.Filename)
			err = this.SaveToFile("file", filepath.Join(conf("tmppath"), filename))
			if err != nil {
				log("凭证文件保存失败:err[%v]", err)
				this.Rec = &Recv{5, "凭证文件上传失败", nil}
			}else{
				imgurl += ps("https://%s/%s;", conf("tmpdown"), filename)
			}
		}
	} else {
		log("上传文件为空")
	}

	sql := ps("update `withdraw` set remark='%s',certificate='%s',re_account='%s',state=1 where id='%d';", remark , imgurl , this.User.Account , id)
	db := orm.NewOrm()
	_, err = db.Raw(sql).Exec()
	if err != nil {
		log("提现确认失败:err[%v]", err)
		this.Rec = &Recv{5, "提现确认失败", nil}
		return
	}

	this.Rec = &Recv{3, "提现确认成功", nil}
}
