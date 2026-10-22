package httpserver

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

var businessSummaries = map[string]string{
	"GET /api/v1/status":                                                "查询服务状态",
	"GET /api/v1/account/context":                                       "获取身份站登录上下文",
	"POST /api/v1/account/login":                                        "使用邮箱和密码登录",
	"POST /api/v1/account/verification":                                 "发送注册邮箱验证码",
	"POST /api/v1/account/register":                                     "注册账号",
	"GET /api/v1/account/sessions":                                      "列出身份站会话",
	"DELETE /api/v1/account/sessions/{id}":                              "撤销身份站会话",
	"POST /api/v1/account/logout":                                       "退出身份站当前会话",
	"GET /api/v1/account/authorizations/{id}":                           "获取授权确认详情",
	"POST /api/v1/account/authorizations/{id}":                          "批准或拒绝授权",
	"GET /api/v1/account/logout-requests/{id}":                          "获取中央退出确认详情",
	"POST /api/v1/account/logout-requests/{id}":                         "确认或取消中央退出",
	"GET /api/v1/account/federation/providers":                          "列出可用的第三方登录方式",
	"POST /api/v1/account/federation/{provider}":                        "发起第三方登录",
	"GET /api/v1/account/federation/registration":                       "获取第三方注册补全信息",
	"POST /api/v1/account/federation/registration":                      "完成第三方账号注册",
	"POST /api/v1/account/password-resets":                              "发送密码恢复邮件",
	"POST /api/v1/account/password-resets/confirm":                      "验证邮件并设置新密码",
	"POST /api/v1/account/deletion-cancellations":                       "发送注销取消验证码",
	"POST /api/v1/account/deletion-cancellations/confirm":               "验证并取消账号注销",
	"GET /api/v1/portal/context":                                        "获取用户站登录上下文",
	"POST /api/v1/portal/logout":                                        "退出用户站当前会话",
	"POST /api/v1/portal/central-logout":                                "发起用户站中央退出",
	"GET /api/v1/portal/sessions":                                       "列出用户站会话",
	"DELETE /api/v1/portal/sessions/{id}":                               "撤销用户站会话",
	"PATCH /api/v1/portal/profile":                                      "更新个人资料",
	"PUT /api/v1/portal/avatar":                                         "上传或删除头像",
	"POST /api/v1/portal/password":                                      "修改本地登录密码",
	"POST /api/v1/portal/email-changes":                                 "发送新邮箱验证邮件",
	"POST /api/v1/portal/email-changes/confirm":                         "确认更换邮箱",
	"GET /api/v1/portal/authorizations":                                 "列出个人授权记录",
	"DELETE /api/v1/portal/authorizations/{id}":                         "撤销个人授权",
	"GET /api/v1/portal/security-events":                                "查询个人安全记录",
	"GET /api/v1/portal/identities":                                     "列出已绑定的第三方身份",
	"POST /api/v1/portal/identities/{provider}":                         "发起第三方身份绑定",
	"DELETE /api/v1/portal/identities/{provider}":                       "解除第三方身份绑定",
	"POST /api/v1/portal/deletion":                                      "申请注销账号",
	"GET /api/v1/developer/applications":                                "列出自己的应用",
	"POST /api/v1/developer/applications":                               "创建测试应用",
	"GET /api/v1/developer/applications/{id}":                           "获取自己的应用详情",
	"PUT /api/v1/developer/applications/{id}/draft":                     "保存应用配置草稿",
	"POST /api/v1/developer/applications/{id}/submit":                   "提交应用配置审核",
	"POST /api/v1/developer/applications/{id}/availability":             "启用或停用自己的应用",
	"POST /api/v1/developer/applications/{id}/secret":                   "轮换应用客户端密钥",
	"DELETE /api/v1/developer/applications/{id}":                        "删除自己的应用",
	"GET /api/v1/developer/applications/{id}/versions":                  "查询应用配置版本历史",
	"GET /api/v1/developer/applications/{id}/statistics":                "查询应用协议使用统计",
	"GET /api/v1/developer/applications/{id}/test-members":              "列出应用测试成员",
	"POST /api/v1/developer/applications/{id}/test-members":             "添加应用测试成员",
	"DELETE /api/v1/developer/applications/{id}/test-members/{user_id}": "移除应用测试成员",
	"GET /api/v1/admin/context":                                         "获取管理员登录与 MFA 状态",
	"POST /api/v1/admin/logout":                                         "退出管理台当前会话",
	"POST /api/v1/admin/central-logout":                                 "发起管理台中央退出",
	"GET /api/v1/admin/sessions":                                        "列出管理台会话",
	"DELETE /api/v1/admin/sessions/{id}":                                "撤销管理台会话",
	"POST /api/v1/admin/mfa/enrollment":                                 "使用当前账号密码开始绑定验证器",
	"POST /api/v1/admin/mfa/enrollment/confirm":                         "使用验证器动态码完成绑定",
	"POST /api/v1/admin/mfa/verify":                                     "验证管理员动态码",
	"POST /api/v1/admin/mfa/recover":                                    "使用单次恢复码重新绑定验证器",
	"POST /api/v1/admin/mfa/recovery-codes":                             "重新生成管理员恢复码",
	"POST /api/v1/admin/mfa/emergency":                                  "消费操作员应急凭据",
	"GET /api/v1/admin/applications":                                    "查询平台应用列表",
	"GET /api/v1/admin/applications/{id}":                               "获取平台应用详情",
	"GET /api/v1/admin/applications/{id}/versions":                      "查询平台应用配置历史",
	"POST /api/v1/admin/applications/{id}/review":                       "批准或拒绝应用配置",
	"POST /api/v1/admin/applications/{id}/suspension":                   "暂停或解除暂停应用",
	"GET /api/v1/admin/users":                                           "检索平台用户",
	"POST /api/v1/admin/users/{id}/availability":                        "停用或恢复普通用户",
	"GET /api/v1/admin/registration-policy":                             "获取注册准入策略",
	"PUT /api/v1/admin/registration-policy":                             "切换开放或邀请注册",
	"GET /api/v1/admin/invitations":                                     "查询邀请码记录",
	"POST /api/v1/admin/invitations":                                    "发行单次邀请码",
	"DELETE /api/v1/admin/invitations/{id}":                             "撤销邀请码",
	"GET /api/v1/admin/audit-events":                                    "查询平台安全审计",
	"GET /api/v1/admin/principals":                                      "列出管理员主体",
	"POST /api/v1/admin/principals/{id}/availability":                   "授予或撤回管理员角色",
	"GET /api/v1/admin/statistics":                                      "查询平台概览与注册趋势",
	"GET /api/v1/admin/notifications/failed":                            "查询可重投的失败邮件",
	"POST /api/v1/admin/notifications/{id}/retry":                       "将失败邮件重新排队",
}

func documentBusinessAPI(doc *huma.OpenAPI) {
	for path, item := range doc.Paths {
		for _, op := range []*huma.Operation{item.Get, item.Post, item.Put, item.Patch, item.Delete} {
			if op == nil {
				continue
			}
			if summary, ok := businessSummaries[op.Method+" "+path]; ok {
				op.Summary = summary
			}
			tag := "服务状态"
			for prefix, name := range map[string]string{"account": "身份与注册", "portal": "用户中心", "developer": "开发者应用", "admin": "管理控制台"} {
				if strings.HasPrefix(path, "/api/v1/"+prefix+"/") {
					tag = name
				}
			}
			op.Tags = []string{tag}
		}
	}

	mfa := doc.Paths["/api/v1/admin/mfa/enrollment"].Post
	mfa.Description = "开始绑定管理员 TOTP 验证器。current_password 是当前 Moetopia 账号的本地登录密码，不是独立后台密码、邮箱验证码或验证器动态码。先完成管理员登录；提交后使用验证器扫描返回的 otpauth_uri 对应二维码（或手动输入 secret），再向 /api/v1/admin/mfa/enrollment/confirm 提交验证器生成的六位 code。只有确认成功才完成 MFA 并返回单次恢复码。纯第三方账号须先经邮箱密码恢复设置本地密码。需要独立管理会话、Origin 与 CSRF；此阶段不要求已验证 MFA。秘密仅用于本次绑定，禁止日志或持久化到浏览器存储。"
}
