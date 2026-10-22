# 加密备份、恢复与安全切换

状态：工具已实现，真实隔离演练结果见[实施进度](implementation-status.md)。不自动为实际生产配置备份调度、保留期或异地存储；这些参数尚未指定，不宣称已有 RPO/RTO 保证。

## 两份独立备份

数据库使用 `pg_dump` custom archive 经管道直接交给 age 加密，落盘只有 `.dump.age`、校验和；根加密密钥使用另一个 age recipient、另一目录单独备份。两个恢复私钥在服务器外分别保管，不放入 Git、聊天、镜像或同一备份包。签名私钥/TOTP/outbox 在库内仍有应用层加密，丢失原根密钥不能靠数据库备份恢复。

```bash
sh deploy/compose.sh --profile maintenance run --rm backup
sh deploy/compose.sh --profile maintenance run --rm backup-key
```

事先在 `.env.production` 填写 `MOETOPIA_BACKUP_RECIPIENT` 和不同的 `MOETOPIA_KEY_BACKUP_RECIPIENT`，创建由操作员 UID 拥有的私有输出目录。脚本拒绝缺失 recipient，使用随机后缀避免覆盖；失败保留 `.partial` 而不会伪装完成。校验和用于检测传输变化，不代替可信来源或 age 验证。将成功产物复制到独立受控存储，按实际政策定期检查恢复；本工程不擅自删除旧备份。

`pg_dump` 是一致性快照，不包含快照后提交的数据，也不备份集群角色；新环境先建立迁移角色和数据库，再用 permissions 建立运行角色。参考 [PostgreSQL dump 文档](https://www.postgresql.org/docs/18/backup-dump.html)。age 固定为 [v1.3.2](https://github.com/FiloSottile/age/releases/tag/v1.3.2)。

## 恢复到全新数据库

先进入维护窗口，停止所有连接此 issuer 的 API、内置任务和其他写入者，保留源库。核对 archive 来源、校验和、根密钥托管记录和镜像版本；不恢复来源不明的 dump，因为恢复可执行归档中的 SQL，见 [pg_restore 文档](https://www.postgresql.org/docs/18/app-pgrestore.html)。

显式挂载离线恢复身份文件（`MOETOPIA_RESTORE_IDENTITY_FILE`，私有 Linux 文件权限），配置下列非秘密参数；文件名替换为实际成功备份：

```bash
export MOETOPIA_RESTORE_DATABASE="moetopia_restore_$(openssl rand -hex 8)"
export MOETOPIA_RESTORE_ARCHIVE='moetopia-REPLACE-WITH-ACTUAL-BACKUP.dump.age'
sh deploy/compose.sh --profile restore run --rm restore
```

只有 `moetopia_restore_<16 位小写十六进制>` 可作为目标；目标必须尚不存在。脚本先 createdb，再将 age 解密流交给单事务 pg_restore，无 `--clean`、无覆盖源库。失败时保留目标供检查，重新尝试使用另一个新目标，不把失败库自动当作可切换实例。

通过秘密管理工具创建 `secrets/restore_database_url`，指向**该新库**的迁移角色 DSN。不要在控制台输出或把秘密放进命令行。恢复的应用根密钥必须来自原密钥托管，正确填写 `encryption_key`；`restore-check` 严格检查实际解析的数据库名，拒绝 DSN 查询参数绕过目标限制。

```bash
sh deploy/compose.sh --profile restore run --rm restore-check
sh deploy/compose.sh --profile restore run --rm restore-check --invalidate-transient
```

第一条只检查当前要求的 schema 与全部仍保留的加密签名私钥；不自动改表。旧快照缺迁移时，先把独立迁移任务的秘密 DSN 指向该新库，按版本说明补齐迁移，再检查。错误根密钥在删除凭据前失败；第二条在单事务内撤销恢复快照的浏览器/RP/协议会话、授权族、授权/退出交互、验证码/挑战、外部登录事务、未用邀请码、邮件队列和 MFA 提升，并记录恢复审计。普通账号密码、外部绑定、长期 TOTP 和应用配置保留；尚未完成的 MFA 重新绑定保持受限，不能借恢复绕过 MFA。

Valkey 在切换时使用新的本项目专属空实例/命名空间，保持敏感限流 fail-closed；不要清空同机共享缓存。清除短期凭据不能消除快照后的密码/邮箱/绑定/注销差异；需按可信安全记录对账，尤其不能未经检查恢复应已删除的账号。备份保留、注销政策与事故处理需共同评审。

确认新库检查成功后，修改 `MOETOPIA_DATABASE_NAME`、app_database_url、migration_database_url 为同一新库，重新运行 permissions，再启动全部新版本服务。保留旧库和镜像至验收结束，验证真实登录、MFA、邮件和下游重新授权。原 issuer/domain 必须保持受控；不要让旧实例同时写源库形成分叉。离线 JWT、缓存 JWKS 与第三方 RP 自己的会话不会凭中央恢复立即消失，应按风险协调下游撤销。

## 签名密钥泄露

常规轮换保留十五分钟兼容窗口；泄露场景使用[签名密钥指南](run-signing-keys.md)中的维护命令。应急轮换立即退出旧公钥并清除旧私钥，同时撤销全部瞬态凭据；必须先停止所有 issuer 实例。根加密密钥泄露还影响 TOTP、外部事务和备份，不等同于只更换签名密钥，需另行处理受影响长期秘密。
