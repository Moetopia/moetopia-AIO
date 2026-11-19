# 生产部署：NPM 与共享 PostgreSQL

使用 Docker Compose 2.24.4 或更新版本。此模式复用 `proxy-network` 与 `database-network`；默认启动 API、用户站、管理台和私有 Valkey。 [Compose 合并规则](https://docs.docker.com/reference/compose-file/merge/) · [NPM 共享网络](https://nginxproxymanager.com/advanced-config/#best-practice-use-a-docker-network)

隔离验证已在 PostgreSQL 17.12 上以非超管账号应用迁移 1–16，验证运行角色可读取业务表、不能建 schema；管理台镜像构建及容器内深链接探测通过。实际 NPM 配置、正式证书和浏览器登录需在目标主机验收。

## 数据库和容器

1. 在 `common-postgres` 执行 `SHOW server_version;` 确认版本。创建 Moetopia 专属数据库 `moetopia` 和两个登录角色 `moetopia_migrator`、`moetopia_app`。迁移角色拥有该库；两个角色均无需超管、建库或建角色权限。通过数据库管理工具分别设置密码。
2. 在 Linux 部署目录首次执行 `cp deploy/production.env.example .env.production` 和 `sh deploy/init-secrets.sh`。已有部署保留原配置和加密密钥。将 `deploy/secrets/migration_database_url` 与 `deploy/secrets/app_database_url` 分别指向 `common-postgres:5432/moetopia` 上的对应角色，TLS 按共享数据库实际配置填写。设置正式双域名、TLS SMTP 和发布标签。SMTP 密码填入 `deploy/secrets/smtp_password`。基础 Compose 解析所需的 `MOETOPIA_TLS_EMAIL` 与 `MOETOPIA_PROXY_SUBNET` 可保留模板值。
3. 查询 NPM 容器的代理网地址：`docker inspect --format '{{(index .NetworkSettings.Networks "proxy-network").IPAddress}}' nginx-proxy-manager`。将该 IPv4 地址加 `/32` 填入 `MOETOPIA_NPM_PEER_CIDR`，禁用对整个共享网段的信任。给 NPM 保留固定地址，或在它重建后同步该值并重建 API。

```sh
sh deploy/compose-npm.sh config --services
sh deploy/compose-npm.sh build api account admin
sh deploy/compose-npm.sh --profile maintenance run --rm migrate
```

以迁移角色连接 Moetopia 专属库，执行 `psql -X -v ON_ERROR_STOP=1 -f deploy/grants.sql`；连接参数通过 `PGHOST`、`PGDATABASE`、`PGUSER` 和私有密码文件提供。以后升级迁移后也执行此步。

```sh
sh deploy/compose-npm.sh run --rm --no-deps --entrypoint /app/keys api init
sh deploy/compose-npm.sh run --rm --no-deps --entrypoint /app/firstparty api
sh deploy/compose-npm.sh run --rm --no-deps --entrypoint /app/administrator api init-client
sh deploy/compose-npm.sh up -d --wait
```

数据库备份和恢复沿用共享 PostgreSQL 的运维流程，需能单独恢复 Moetopia 库。另行加密保存原始 `encryption_key`。

## NPM 路由

用户站转发到 `http://moetopia-account:3000`，管理台转发到 `http://moetopia-admin:3000`。两个域名分别配置证书和 Force SSL。两个 Proxy Host 的 Advanced 配置均加入以下路由：

```nginx
location ~ ^/(?:api/|oauth/|\.well-known/(?:openid-configuration|oauth-authorization-server)$) {
    resolver 127.0.0.11 valid=10s;
    set $moetopia_api http://moetopia-api:8080;
    proxy_pass $moetopia_api;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Moetopia-Client-IP $remote_addr;
}
```

NPM 前面若还有 CDN，先在 NPM 配好可信来源，使 `$remote_addr` 代表实际客户端。

## 注册首个账号并初始化管理员

在 Windows PowerShell 操作 WSL 实例时，见 [WSL 命令示例](live-development.md#在-windows-powershell-中操作-wsl-容器)。

1. 确认用户站和管理台均可通过正式 HTTPS 地址打开，TLS SMTP 能向首位管理员的邮箱投递。执行下面的命令获取注册表单所需的首装邀请码。有效期一小时，单次使用。已有有效的邀请码或管理员初始化记录时，命令会拒绝再次签发。

```sh
sh deploy/compose-npm.sh run --rm --no-deps --entrypoint /app/administrator api bootstrap-invite
```

2. 打开用户站 `/auth/register`，填入邀请码、邮箱、昵称和本地登录密码。点击发送验证码，从实际邮箱收取验证码并提交注册。邀请码用于注册准入，邮件验证码用于验证邮箱。已有启用且已验证的账号时，可直接进入下一步。
3. 将命令中的示例邮箱换成该账号的邮箱，授予首位管理员角色。

```sh
sh deploy/compose-npm.sh run --rm --no-deps --entrypoint /app/administrator api bootstrap --email admin@example.com
```

4. 从管理台域名发起登录。绑定验证器时填当前 Moetopia 本地登录密码，扫码添加 TOTP 后输入六位动态码。将一次性展示的恢复码离线保存，再验证可以访问管理页面。后续在管理台发放邀请码、切换注册模式和管理角色。恢复和应急处理见 [管理员控制](run-admin-controls.md)。
5. 对外开放前，从客户端网络检查两个发现地址及 issuer，打开管理台深链接，并用独立应用完成 OIDC 登录和回跳。 [OIDC 配置](configure-oidc.md) · [前端验收](coordination/production-frontend.md)
