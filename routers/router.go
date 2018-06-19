package routers

import (
	"cowin/controllers"
	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/cowin/moni", &controllers.BaseController{}, "post:MoniSrv")                                        // 检测服务
	beego.Router("/cowin/check", &controllers.UserController{}, "post:LogCheck")                                      // 检测用户登录
	beego.Router("/cowin/vcode/get", &controllers.LoginController{}, "post:SendVerCode")                              // 获取验证码
	beego.Router("/cowin/register", &controllers.LoginController{}, "post:Register")                                  // 注册
	beego.Router("/cowin/login", &controllers.LoginController{}, "post:Login")                                        // 登陆
	beego.Router("/cowin/wxlogin", &controllers.LoginController{}, "post:WxLogin")                                    // 微信登陆
	beego.Router("/cowin/wxbind", &controllers.LoginController{}, "post:WxBindPhone")                                 // 微信绑定手机号
	beego.Router("/cowin/wx/verify", &controllers.UserLoginController{}, "post:WxBind")                               // 微信绑定平台账号
	beego.Router("/cowin/wx/pay/ensure", &controllers.WxpayController{}, "post:Recharge")                             // 微信支付确认
	beego.Router("/cowin/logout", &controllers.UserLoginController{}, "post:Logout")                                  // 退出
	beego.Router("/cowin/userinfo", &controllers.UserController{}, "post:QueryUserInfo")                              // 查询在线用户信息
	beego.Router("/cowin/userinfo/update", &controllers.UserLoginController{}, "post:UpdateUserInfo")                 // 修改用户信息
	beego.Router("/cowin/pwd/modify", &controllers.UserLoginController{}, "post:ModifyPwd")                           // 修改密码
	beego.Router("/cowin/pwd/reset", &controllers.LoginController{}, "post:ResetPwd")                                 // 重置密码
	beego.Router("/cowin/upload/head", &controllers.UserController{}, "post:UploadHead")                              // 修改头像
	beego.Router("/cowin/upload/other", &controllers.UserController{}, "post:UploadTmpFile")                          // 上传其他文件
	beego.Router("/cowin/msg/query", &controllers.MsgController{}, "post:MsgQuery")                                   // 消息查询
	beego.Router("/cowin/talk/publish", &controllers.TalkaboutController{}, "post:TalkaboutPublish")                  // 发布说说
	beego.Router("/cowin/talk/del", &controllers.TalkaboutController{}, "post:TalkaboutDel")                          // 删除说说(仅限自己发布的)
	beego.Router("/cowin/talk/query", &controllers.TalkaboutBaseController{}, "post:TalkaboutQuery")                  // 查询说说
	beego.Router("/cowin/talk/view", &controllers.TalkaboutBaseController{}, "post:TalkaboutView")                    // 查看说说
	beego.Router("/cowin/talk/review", &controllers.TalkaboutController{}, "post:TalkaboutReview")                    // 评论说说
	beego.Router("/cowin/talk/review/query", &controllers.TalkaboutBaseController{}, "post:TalkaboutReviewQuery")     // 评论说说查询
	beego.Router("/cowin/talk/review/del", &controllers.TalkaboutController{}, "post:TalkaboutReviewDel")             // 评论删除
	beego.Router("/cowin/talk/secreview", &controllers.TalkaboutController{}, "post:TalkaboutSecReview")              // 发表二级评论
	beego.Router("/cowin/talk/secreview/del", &controllers.TalkaboutController{}, "post:TalkaboutSecReviewDel")       // 二级评论删除
	beego.Router("/cowin/talk/fans", &controllers.TalkaboutController{}, "post:TalkaboutFans")                        // 点赞说说
	beego.Router("/cowin/talk/fans/query", &controllers.TalkaboutBaseController{}, "post:TalkaboutFansQuery")         // 点赞查询
	beego.Router("/cowin/report/query", &controllers.TalkaboutBaseController{}, "post:ReportQuery")                   // 举报原因查询
	beego.Router("/cowin/talk/report", &controllers.TalkaboutController{}, "post:ReportTalk")                         // 举报说说
	beego.Router("/cowin/talk/report/query", &controllers.TalkaboutController{}, "post:ReportTalkQuery")              // 查询说说举报内容
	beego.Router("/cowin/talk/review/report", &controllers.TalkaboutController{}, "post:ReportReview")                // 举报一级评论
	beego.Router("/cowin/talk/secreview/report", &controllers.TalkaboutController{}, "post:ReportSecReview")          // 举报二级评论
	beego.Router("/cowin/carousel/query", &controllers.CarouselBaseController{}, "post:CarouselQuery")                // 轮播图查询
	beego.Router("/cowin/mall/willproduct/add", &controllers.MallBaseController{}, "post:WillProductAdd")             // 添加待审核(意愿)产品
	beego.Router("/cowin/mall/willproduct/query", &controllers.MallBaseController{}, "post:WillProductQuery")         // 意愿产品查询
	beego.Router("/cowin/mall/product/query", &controllers.MallBaseController{}, "post:ProductQuery")                 // 产品查询
	beego.Router("/cowin/mall/product/sales", &controllers.MallBaseController{}, "post:ProductSalesQuery")            // 产品销售情况查询
	beego.Router("/cowin/product/sold/query", &controllers.MallBaseController{}, "post:ProductSoldQuery")             // 产品已售数据统计
	beego.Router("/cowin/product/buyrd/query", &controllers.MallController{}, "post:ProductBuyrdQuery")               // 产品销量查询
	beego.Router("/cowin/product/pay/details/cdb", &controllers.MallController{}, "post:ProductPayDetailsCdb")        // 充电宝产品使用明细
	beego.Router("/cowin/product/review", &controllers.ProductController{}, "post:ProductReview")                     // 评论产品
	beego.Router("/cowin/product/review/fans", &controllers.ProductController{}, "post:ProductReviewFans")            // 产品评论点赞
	beego.Router("/cowin/product/review/query", &controllers.ProductBaseController{}, "post:ProductReviewQuery")      // 产品评论查询
	beego.Router("/cowin/isscoup/query", &controllers.CouponController{}, "post:IsscoupQuery")                        // 优惠券查询
	beego.Router("/cowin/extract", &controllers.ExtractController{}, "post:Extract")                                  // 提取现金
	beego.Router("/cowin/extract/query", &controllers.ExtractController{}, "post:ExtractQuery")                       // 提取记录查询
	beego.Router("/cowin/product/type/query", &controllers.ProductBaseController{}, "post:ProductTypeQuery")          // 产品类型查询
	beego.Router("/cowin/mall/hostmethod/query", &controllers.EnjoyBaseController{}, "post:HostMethodQuery")          // 托管方式查询
	beego.Router("/cowin/mall/product/buy", &controllers.EnjoyproductController{}, "post:ProductBuy")                 // 产品购买
	beego.Router("/cowin/mall/order/query", &controllers.EnjoyproductController{}, "post:ProductOrderQuery")          // 订单查询
	beego.Router("/cowin/userpd/order/query", &controllers.EnjoyproductController{}, "post:UserProductOrderQuery")    // 资产界面订单查询
	beego.Router("/cowin/mall/order/receipt", &controllers.EnjoyproductController{}, "post:ProductOrderReceipt")      // 购买订单确认收货
	beego.Router("/cowin/mall/order/cancel", &controllers.EnjoyproductController{}, "post:ProductOrderCancel")        // 取消订单
	beego.Router("/cowin/mall/order/pay", &controllers.EnjoyproductController{}, "post:ProductOrderPay")              // 订单支付确认
	beego.Router("/cowin/mall/order/paystatus", &controllers.EnjoyproductController{}, "post:ProductOrderPayStatus")  // 订单支付查询
	beego.Router("/cowin/product/agreement/sign", &controllers.EnjoyproductController{}, "post:OrderAgreementSign")   // 订单合同签署
	beego.Router("/cowin/product/agreement/query", &controllers.EnjoyproductController{}, "post:OrderAgreementQuery") // 订单合同查询
	beego.Router("/cowin/mall/order/flow", &controllers.EnjoyproductController{}, "post:MallOrderFlow")               // 取消订单
	beego.Router("/cowin/userpd/order/receipt", &controllers.UserProductController{}, "post:UserPdtPickReceipt")      // 提货订单确认收货
	beego.Router("/cowin/userpd/pickup", &controllers.UserProductController{}, "post:UserProductPickup")              // 运营产品提货
	beego.Router("/cowin/userpd/puorder/query", &controllers.UserProductController{}, "post:PickupProductOrderQuery") // 产品提货订单查询
	beego.Router("/cowin/product/repair", &controllers.UserProductController{}, "post:ProductRepair")                 // 产品报修
	beego.Router("/cowin/product/use/pay", &controllers.UserProductController{}, "post:ProductUsePay")                // 产品使用支付
	beego.Router("/cowin/userpd/general", &controllers.UserProductController{}, "post:UserProductGeneral")            // 投放中产品运营统计
	beego.Router("/cowin/userpd/query", &controllers.UserProductController{}, "post:UserProductQuery")                // 投放中产品详细统计
	beego.Router("/cowin/userpd/use/record", &controllers.UserProductController{}, "post:UserProductUseRecord")       // 产品详细使用记录
	beego.Router("/cowin/userproduct/baseinfo", &controllers.UserProductBaseController{}, "post:ProductBaseinfo")     // 产品基本信息查询
	beego.Router("/cowin/userproduct/useinfo", &controllers.UserProductController{}, "post:ProductUseinfo")           // 产品使用信息查询
	beego.Router("/cowin/transport/query", &controllers.UserProductController{}, "post:TransportQuery")               // 物流公司查询
	beego.Router("/cowin/income/brief", &controllers.IncomeController{}, "post:IncomeBrief")                          // 总收益,日收益查询
	beego.Router("/cowin/income/total", &controllers.IncomeController{}, "post:IncomeTotalQuery")                     // 总收益分类查询
	beego.Router("/cowin/income/detail", &controllers.IncomeController{}, "post:IncomeDetailQuery")                   // 收益明细查询
	beego.Router("/cowin/shipaddr/add", &controllers.AddressController{}, "post:ShipAddrAdd")                         // 收货地址添加
	beego.Router("/cowin/shipaddr/query", &controllers.AddressController{}, "post:ShipAddrQuery")                     // 收货地址查询
	beego.Router("/cowin/shipaddr/modify", &controllers.AddressController{}, "post:ShipAddrModify")                   // 收货地址修改
	beego.Router("/cowin/shipaddr/del", &controllers.AddressController{}, "post:ShipAddrDel")                         // 收货地址删除
	beego.Router("/cowin/shipaddr/default", &controllers.AddressController{}, "post:ShipAddrDefault")                 // 设置默认收货地址
	beego.Router("/cowin/letter/send", &controllers.LetterController{}, "post:LetterSend")                            // 发送私信
	beego.Router("/cowin/letter/recv", &controllers.LetterController{}, "post:LetterQuery")                           // 查询私信
	beego.Router("/cowin/product/agreement/update", &controllers.AgreementController{}, "post:AgreementUpdate")       // 产品合同更新
	beego.Router("/cowin/product/agreement/down", &controllers.AgreementController{}, "post:AgreementDown")           // 产品合同下载
	beego.Router("/cowin/product/agreement/upload", &controllers.AgreementController{}, "post:AgreementUpload")       // 产品合同上传
	beego.Router("/cowin/user/authen", &controllers.AgreementController{}, "post:UserInfoAuth")                       // 用户真实信息认证
	beego.Router("/cowin/user/authenEx", &controllers.CardCertificationController{}, "post:CheckCardCA")              // 用户提交认证信息
	beego.Router("/cowin/user/authen/query", &controllers.AgreementController{}, "post:UserAuthInfoQuery")            // 用户认证信息查询
	beego.Router("/cowin/shopcart/add", &controllers.ShopcartController{}, "post:ShopcartAdd")                        // 加入购物车
	beego.Router("/cowin/shopcart/query", &controllers.ShopcartController{}, "post:ShopcartQuery")                    // 查询购物车
	beego.Router("/cowin/shopcart/del", &controllers.ShopcartController{}, "post:ShopcartDel")                        // 购物车删除
	beego.Router("/cowin/shopcart/settle", &controllers.ShopcartController{}, "post:ShopcartSettle")                  // 购物车结算
	beego.Router("/cowin/product/city/query", &controllers.ProductBaseController{}, "post:ProductCityQuery")          // 产品投放城市查询
	beego.Router("/cowin/express/query", &controllers.ExpressbirdController{}, "post:ExpressInfoQuery")               // 产品投放城市查询
	beego.Router("/cowin/app/update", &controllers.SoftwareBaseController{}, "post:QuerySoftwareInfo")                // Android版本更新检查
	beego.Router("/cowin/product/serial/query", &controllers.UserProductBaseController{}, "post:ProductSerialQuery")  // 未匹配产品编号查询
	beego.Router("/cowin/product/serial/match", &controllers.UserProductBaseController{}, "post:ProductSerialMatch")  // 产品编号匹配
	beego.Router("/cowin/product/operation", &controllers.UserProductBaseController{}, "post:ProductOperations")      // 产品投入运营
	beego.Router("/cowin/recharge/gold", &controllers.RechargeController{}, "post:RechargeGold")                      // 提交充值订单
	beego.Router("/cowin/recharge/query", &controllers.RechargeController{}, "post:QuaryRechargeHistory")             // 查询充值订单
	beego.Router("/cowin/recharge/orderquery", &controllers.RechargeController{}, "post:QuaryRechargeByCode")         // 查询单个订单
	beego.Router("/cowin/withdraw/apply", &controllers.RechargeController{}, "post:WithDraw")                           // 提现申请
	beego.Router("/cowin/recharge/flow", &controllers.AlipayController{}, "post:QueryMoneyFlow")                      // 查询资金明细
	beego.Router("/cowin/alipay/notice", &controllers.AlipayBaseController{}, "post:AlipayNotice")                    // 支付宝异步通知接口
	beego.Router("/cowin/wxpay/notice", &controllers.WxpayBaseController{}, "post:WxpayNotice")                       // 微信异步通知接口
	beego.Router("/cowin/wxsharing", &controllers.WxpayBaseController{}, "post:WxSharing")                            // 微信公众号分享

	/******************************************************************后台接口********************************************************************/
	beego.Router("/cowin/admin/adduser", &controllers.AdminController{}, "post:AddUser")                         // 添加用户
	beego.Router("/cowin/admin/userquery", &controllers.AdminController{}, "post:QueryUser")                     // 查询所有后端用户
	beego.Router("/cowin/admin/user/flagquery", &controllers.AdminController{}, "post:FlagQueryUser")            // 分类查询用户
	beego.Router("/cowin/admin/modifyuser", &controllers.AdminController{}, "post:ModifyUser")                   // 修改用户
	beego.Router("/cowin/admin/deluser", &controllers.AdminController{}, "post:DelUser")                         // 删除用户
	beego.Router("/cowin/admin/modifylevel", &controllers.AdminController{}, "post:ModifyUserLevel")             // 修改前端用户等级
	beego.Router("/cowin/admin/specifyuser", &controllers.AdminController{}, "post:SpecifyUser")                 // 查询特定用户信息
	beego.Router("/cowin/user/authen/bkquery", &controllers.AdminController{}, "post:UserAuthInfoBkQuery")       // 用户认证信息查询
	beego.Router("/cowin/user/authen/check", &controllers.AdminController{}, "post:UserAuthInfoCheck")           // 用户认证信息审核
	beego.Router("/cowin/msg/add", &controllers.MsgController{}, "post:MsgAdd")                                  // 消息添加
	beego.Router("/cowin/msg/modify", &controllers.MsgController{}, "post:MsgModify")                            // 消息修改
	beego.Router("/cowin/msg/del", &controllers.MsgController{}, "post:MsgDel")                                  // 消息删除
	beego.Router("/cowin/msg/send", &controllers.MsgController{}, "post:MsgSend")                                // 消息发送
	beego.Router("/cowin/level/add", &controllers.LevelController{}, "post:AddLevel")                            // 等级添加
	beego.Router("/cowin/level/modify", &controllers.LevelController{}, "post:ModifyLevel")                      // 等级修改
	beego.Router("/cowin/level/del", &controllers.LevelController{}, "post:DelLevel")                            // 等级删除
	beego.Router("/cowin/report/add", &controllers.TalkaboutController{}, "post:ReportAdd")                      // 举报原因添加
	beego.Router("/cowin/report/del", &controllers.TalkaboutController{}, "post:ReportDel")                      // 举报原因删除
	beego.Router("/cowin/report/modify", &controllers.TalkaboutController{}, "post:ReportModify")                // 举报原因修改
	beego.Router("/cowin/talk/bs/query", &controllers.TalkaboutController{}, "post:TalkaboutBsQuery")            // 查询说说
	beego.Router("/cowin/talk/check", &controllers.TalkaboutController{}, "post:TalkaboutCheck")                 // 审核说说
	beego.Router("/cowin/talk/search", &controllers.TalkaboutController{}, "post:TalkaboutSearch")               // 搜索说说
	beego.Router("/cowin/talk/bs/del", &controllers.TalkaboutController{}, "post:TalkaboutBsDel")                // 删除说说
	beego.Router("/cowin/talk/review/search", &controllers.TalkaboutController{}, "post:TalkReviewSearch")       // 搜索评论
	beego.Router("/cowin/talk/review/sdel", &controllers.TalkaboutController{}, "post:TalkReviewBsDel")          // 删除评论
	beego.Router("/cowin/talk/secreview/search", &controllers.TalkaboutController{}, "post:TalkSecreviewSearch") // 搜索二级评论
	beego.Router("/cowin/talk/secreview/sdel", &controllers.TalkaboutController{}, "post:TalkSecreviewBsDel")    // 删除二级评论
	beego.Router("/cowin/carousel/add", &controllers.CarouselController{}, "post:CarouselAdd")                   // 轮播图添加
	beego.Router("/cowin/carousel/modify", &controllers.CarouselController{}, "post:CarouselModify")             // 轮播图修改
	beego.Router("/cowin/carousel/del", &controllers.CarouselController{}, "post:CarouselDel")                   // 轮播图删除
	beego.Router("/cowin/product/type/add", &controllers.ProductController{}, "post:ProductTypeAdd")             // 产品类型添加
	beego.Router("/cowin/product/type/modify", &controllers.ProductController{}, "post:ProductTypeModify")       // 产品类型修改
	beego.Router("/cowin/product/type/del", &controllers.ProductController{}, "post:ProductTypeDel")             // 产品类型删除
	beego.Router("/cowin/mall/product/add", &controllers.ProductController{}, "post:ProductAdd")                 // 产品添加
	beego.Router("/cowin/mall/product/edit", &controllers.ProductController{}, "post:ProductEdit")               // 产品编辑
	beego.Router("/cowin/mall/product/city", &controllers.ProductController{}, "post:ProductCity")               // 产品投放城市更新
	beego.Router("/cowin/mall/product/check", &controllers.ProductController{}, "post:ProductCheck")             // 产品上下架审核
	beego.Router("/cowin/product/review/reply", &controllers.ProductController{}, "post:ProductReviewReply")     // 产品评论回复
	beego.Router("/cowin/product/review/search", &controllers.ProductController{}, "post:ProductReviewSearch")   // 产品评论搜索
	beego.Router("/cowin/product/review/del", &controllers.ProductController{}, "post:ProductReviewDel")         // 产品评论删除
	beego.Router("/cowin/product/review/check", &controllers.ProductController{}, "post:ProductReviewCheck")     // 产品评论审核
	beego.Router("/cowin/product/review/auth", &controllers.ProductController{}, "post:ProductReviewAuth")       // 设置产品评论权限
	beego.Router("/cowin/mall/willproduct/check", &controllers.MallController{}, "post:WillProductCheck")        // 意愿产品操作
	beego.Router("/cowin/mall/order/search", &controllers.EnjoyproductController{}, "post:ProductOrderSearch")   // 订单搜索
	beego.Router("/cowin/coupon/add", &controllers.CouponController{}, "post:CouponAdd")                         // 优惠券添加
	beego.Router("/cowin/coupon/modify", &controllers.CouponController{}, "post:CouponModify")                   // 优惠券修改
	beego.Router("/cowin/coupon/del", &controllers.CouponController{}, "post:CouponDel")                         // 优惠券删除
	beego.Router("/cowin/coupon/query", &controllers.CouponBaseController{}, "post:CouponQuery")                 // 优惠券查询
	beego.Router("/cowin/extract/srvquery", &controllers.ExtractController{}, "post:ExtractSrvQuery")            // 提取记录查询
	beego.Router("/cowin/income/report", &controllers.IncomeController{}, "post:IncomeReport")                   // 周期报表
	beego.Router("/cowin/income/query", &controllers.IncomeController{}, "post:IncomeQuery")                     // 分类统计
	beego.Router("/cowin/expend/report", &controllers.ExpendController{}, "post:ExpendReport")                   // 周期报表
	beego.Router("/cowin/expend/query", &controllers.ExpendController{}, "post:ExpendQuery")                     // 分类统计
	beego.Router("/cowin/repair/deal", &controllers.UserProductController{}, "post:RepairDeal")                  // 报修处理
	beego.Router("/cowin/prfratio/add", &controllers.PrfratioController{}, "post:PrfratioAdd")                   // 收益比例添加
	beego.Router("/cowin/prfratio/modify", &controllers.PrfratioController{}, "post:PrfratioModify")             // 收益比例修改
	beego.Router("/cowin/prfratio/del", &controllers.PrfratioController{}, "post:PrfratioDel")                   // 收益比例删除
	beego.Router("/cowin/prfratio/query", &controllers.PrfratioController{}, "post:PrfratioQuery")               // 收益比例查询
	beego.Router("/cowin/assets/producting", &controllers.AssetsController{}, "post:AssetsProducting")           // 资产生产中
	beego.Router("/cowin/assets/putin", &controllers.AssetsController{}, "post:AssetsPutin")                     // 资产投放中
	beego.Router("/cowin/assets/order/pickingup", &controllers.AssetsController{}, "post:AssetsOrderPickingup")  // 订单提货中
	beego.Router("/cowin/assets/pdt/pickingup", &controllers.AssetsController{}, "post:AssetsPdtPickingup")      // 产品提货中
	beego.Router("/cowin/assets/scrap", &controllers.AssetsController{}, "post:AssetsScrap")                     // 资产模糊查询
	beego.Router("/cowin/withdraw/query", &controllers.RechargeController{}, "post:QueryWithDraw")               // 提现查询
	beego.Router("/cowin/withdraw/confirm", &controllers.RechargeController{}, "post:ConfirmWithDraw")           // 提现反馈确认
	beego.Router("/cowin/activity/add", &controllers.ActivityController{}, "post:ActivityAdd")                   // 活动添加
	beego.Router("/cowin/activity/del", &controllers.ActivityController{}, "post:ActivityDel")                   // 活动删除
	beego.Router("/cowin/activity/modify", &controllers.ActivityController{}, "post:ActivityModify")             // 活动修改
	beego.Router("/cowin/activity/query", &controllers.ActivityController{}, "post:ActivityQuery")               // 活动查询
	beego.Router("/cowin/activity/check", &controllers.ActivityController{}, "post:ActivityCheck")               // 活动审核

	// 生产厂商---------------------------------------------------------------------------------------------------------------------------------
	beego.Router("/cowin/manu/add", &controllers.ManufactController{}, "post:ManufactAdd")                          // 添加
	beego.Router("/cowin/manu/query", &controllers.ManufactController{}, "post:ManufactQuery")                      // 查询
	beego.Router("/cowin/manu/modify", &controllers.ManufactController{}, "post:ManufactModify")                    // 修改
	beego.Router("/cowin/manu/del", &controllers.ManufactController{}, "post:ManufactDel")                          // 删除
	beego.Router("/cowin/manu/apply/query", &controllers.ManufactController{}, "post:ManufactApplyQuery")           // 厂商未处理申请查询
	beego.Router("/cowin/manu/apply/deal", &controllers.ManufactController{}, "post:ManufactApplyDeal")             // 申请处理
	beego.Router("/cowin/manu/change/apply", &controllers.ManufactController{}, "post:ManufactChangeApply")         // 信息变更申请
	beego.Router("/cowin/manu/change/query", &controllers.ManufactController{}, "post:ManufactChangeQuery")         // 已提交信息变更申请查询
	beego.Router("/cowin/manu/order/undone", &controllers.ManufactController{}, "post:ManufactOrderUndone")         // 未完成订单查询
	beego.Router("/cowin/manu/order/history", &controllers.ManufactController{}, "post:ManufactOrderHistory")       // 历史订单查询
	beego.Router("/cowin/manu/order/producting", &controllers.ManufactController{}, "post:ManufactOrderProducting") // 订单开始生产
	beego.Router("/cowin/manu/order/producted", &controllers.ManufactController{}, "post:ManufactOrderProducted")   // 标识生产完成
	beego.Router("/cowin/manu/order/shipnum", &controllers.ManufactController{}, "post:ManufactShipnum")            // 添加物流单号
	beego.Router("/cowin/manu/order/productno", &controllers.ManufactController{}, "post:ManufactProductno")        // 查询订单下所有产品编号
	// 生产厂商---------------------------------------------------------------------------------------------------------------------------------

	// 服务厂商---------------------------------------------------------------------------------------------------------------------------------
	beego.Router("/cowin/svrpvd/add", &controllers.SvrpvdController{}, "post:SvrpvdAdd")                     // 添加
	beego.Router("/cowin/svrpvd/query", &controllers.SvrpvdController{}, "post:SvrpvdQuery")                 // 查询
	beego.Router("/cowin/svrpvd/modify", &controllers.SvrpvdController{}, "post:SvrpvdModify")               // 修改
	beego.Router("/cowin/svrpvd/del", &controllers.SvrpvdController{}, "post:SvrpvdDel")                     // 删除
	beego.Router("/cowin/svrpvd/apply/query", &controllers.SvrpvdController{}, "post:SvrpvdApplyQuery")      // 厂商未处理申请查询
	beego.Router("/cowin/svrpvd/apply/deal", &controllers.SvrpvdController{}, "post:SvrpvdApplyDeal")        // 申请处理
	beego.Router("/cowin/svrpvd/change/apply", &controllers.SvrpvdController{}, "post:SvrpvdChangeApply")    // 信息变更申请
	beego.Router("/cowin/svrpvd/change/query", &controllers.SvrpvdController{}, "post:SvrpvdChangeQuery")    // 已提交信息变更申请查询
	beego.Router("/cowin/svrpvd/product/ttsh", &controllers.SvrpvdController{}, "post:SvrpvdPrdttsh")        // 托管产品查询
	beego.Router("/cowin/svrpvd/product/pick", &controllers.SvrpvdController{}, "post:SvrpvdPrdpick")        // 自提产品查询
	beego.Router("/cowin/svrpvd/pick/pdtno", &controllers.SvrpvdController{}, "post:SvrpvdPickpdtno")        // 自提订单下产品编号查询
	beego.Router("/cowin/svrpvd/product/ship", &controllers.SvrpvdController{}, "post:SvrpvdPrdship")        // 用户自提产品发货
	beego.Router("/cowin/svrpvd/product/receipt", &controllers.SvrpvdController{}, "post:SvrpvdPrdReceiprt") // 厂商发货确认收货
	// 服务厂商---------------------------------------------------------------------------------------------------------------------------------

	// 经销商后台---------------------------------------------------------------------------------------------------------------------------------
	beego.Router("/cowin/dealer/add", &controllers.DealerController{}, "post:DealerAdd")                             // 添加角色
	beego.Router("/cowin/dealer/modify", &controllers.DealerController{}, "post:DealerModify")                       // 完善角色信息
	beego.Router("/cowin/dealer/del", &controllers.DealerController{}, "post:DealerDel")                             // 删除角色
	beego.Router("/cowin/dealer/query", &controllers.DealerController{}, "post:DealerQuery")                         // 查询角色信息
	beego.Router("/cowin/agents/brokerage", &controllers.DealerController{}, "post:DealerBrokerage")                 // 所有代理商累计佣金
	beego.Router("/cowin/agents/perform", &controllers.DealerController{}, "post:DealerPerform")                     // 特定代理商业务总体统计
	beego.Router("/cowin/agents/userparch", &controllers.DealerController{}, "post:DealerUserparch")                 // 特定代理商下客户购买统计
	beego.Router("/cowin/agents/ranking", &controllers.DealerController{}, "post:AgentsRanking")                     // 代理商上月销售额排行榜
	beego.Router("/cowin/agents/userorder", &controllers.DealerController{}, "post:DealerUserorder")                 // 特定客户订单详情
	beego.Router("/cowin/agents/count", &controllers.DealerController{}, "post:DealerCount")                         // 按月统计每个代理商应得佣金
	beego.Router("/cowin/agents/order", &controllers.DealerController{}, "post:AgentsOrderQuery")                    // 查询代理商下指定月所有订单
	beego.Router("/cowin/agents/personal/brokerage", &controllers.DealerController{}, "post:AgentsPersonalBrokeage") // 我的佣金-累计佣金
	beego.Router("/cowin/agents/personal/order", &controllers.DealerController{}, "post:AgentsPersonalOrder")        // 我的佣金-按月统计客户订单
	beego.Router("/cowin/peragents/perform", &controllers.DealerController{}, "post:PeragentsPerform")               // 个人代理业务总体统计
	beego.Router("/cowin/peragents/userparch", &controllers.DealerController{}, "post:PeragentsUserparch")           // 个人代理客户购买统计
	beego.Router("/cowin/peragents/ranking", &controllers.DealerController{}, "post:PeragentsRanking")               // 个人代理上月销售额排行榜
	beego.Router("/cowin/peragents/brokerage", &controllers.DealerController{}, "post:PeragentsBrokerage")           // 平台/一级代理下全部个人代理累积佣金
	beego.Router("/cowin/peragents/count", &controllers.DealerController{}, "post:PeragentsCount")                   // 按月统计每个个人代理应得佣金
	beego.Router("/cowin/peragents/mybrokerage", &controllers.DealerController{}, "post:PeragentsMybrokerage")       // 我的佣金-累计佣金
	beego.Router("/cowin/peragents/myorder", &controllers.DealerController{}, "post:PeragentsMyorder")               // 我的佣金-按月统计客户订单
	beego.Router("/cowin/sales/perform", &controllers.DealerController{}, "post:SalesPerform")                       // 销售业务总体统计
	beego.Router("/cowin/sales/userparch", &controllers.DealerController{}, "post:SalesUserparch")                   // 销售客户购买统计
	beego.Router("/cowin/sales/ranking", &controllers.DealerController{}, "post:SalesRanking")                       // 销售上月销售额排行榜

	// 经销商后台---------------------------------------------------------------------------------------------------------------------------------

	/******************************************************************后台接口********************************************************************/
}
