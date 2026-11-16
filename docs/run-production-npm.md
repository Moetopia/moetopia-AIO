# 生产部署：NPM 与共享 PostgreSQL

使用 Docker Compose 2.24.4 或更新版本。此模式复用 `proxy-network` 与 `database-network`；默认启动 API、用户站、管理台和私有 Valkey。 [Compose 合并规则](https://docs.docker.com/reference/compose-file/merge/) · [NPM 共享网络](https://nginxproxymanager.com/advanced-config/#best-practice-use-a-docker-network)

隔离验证已在 PostgreSQL 17.12 上以非超管账号应用迁移 1–16，验证运行角色可读取业务表、不能建 schema；管理台镜像构建及容器内深链接探测通过。实际 NPM 配置、正式证书和浏览器登录需在目标主机验收。

## 数据库和容器

1. 在 `common-postgres` 执行 `SHOW server_version;` 确认版本。创建 Moetopia 专属数据库 `moetopia` 和两个登录角色 `moetopia_migrator`、`moetopia_app`。迁移角色拥有该库；两个角色均无需超管、建库或建角色权限。通过数据库管理工具分别设置密码。
2. 按 [首次安装](run-production.md#首次安装) 创建 `.env.production` 和秘密文件。将 `migration_database_url` 与 `app_database_url` 分别指向 `common-postgres:5432/moetopia` 上的对应角色，TLS 按共享数据库实际配置填写。设置正式双域名、SMTP 和发布标签。基础 Compose 解析所需的 `MOETOPIA_TLS_EMAIL` 与 `MOETOPIA_PROXY_SUBNET` 可保留模板值。
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

首位管理员沿用 [首装流程](run-production.md#初始化首位管理员)，将脚本替换为 `deploy/compose-npm.sh`。数据库备份和恢复沿用共享 PostgreSQL 的运维流程，需能单独恢复 Moetopia 库；另行加密保存原始 `encryption_key`。

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

NPM 前面若还有 CDN，先在 NPM 配好可信来源，使 `$remote_addr` 代表实际客户端。验收时检查两个发现地址、管理台深链接、首位管理员注册和 MFA、独立客户端 OIDC 回跳。 [生产前端交接](coordination/production-frontend.md)
