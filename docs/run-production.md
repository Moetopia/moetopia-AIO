# Linux 容器发布

状态：已编写发布工程；本次隔离演练结果见[实施进度](implementation-status.md)。这不是已上线的生产环境。正式域名、SMTP、GitHub/Google、备份存放位置和机器预算仍需实际配置。

## 运行结构

`compose.production.yml` 启动 PostgreSQL、Valkey、Go API、account SSR 和 Caddy。admin 是构建进 Caddy 镜像的静态 CSR 应用；developer 属于 account。邮件、过期清理和注销任务嵌入 API 进程，**没有独立 worker 启动命令**。API 退出时一并停止任务领取。

仅 edge 发布 80/443。数据库、缓存和 API 不发布宿主机端口。private 网络隔离持久化服务；专用 proxy 网络只连接 API 与 edge；outbound 提供外部邮件、供应商请求和证书签发。API 只信任配置 CIDR 内直接代理覆写的 `X-Moetopia-Client-IP`；忽略任意外部 X-Forwarded-For。不得让其他项目加入 proxy 网络。

`deploy/images.env` 固定基础镜像摘要；修改后重新构建和验证。应用镜像标签使用不可变提交标识。API scratch 镜像以非 root 身份运行，account 以 node 用户运行；运行服务配置只读文件系统、资源限制和轮转日志。当前资源上限是初始配置，不是已测得的容量承诺。

多阶段构建将 SDK 与运行环境分开，依据 [Docker 官方说明](https://docs.docker.com/build/building/multi-stage/)。构建 Go 默认使用官方代理；网络限制时可设置 `MOETOPIA_BUILD_GOPROXY`，仍保留 Go checksum 验证，不设置 `GOSUMDB=off`。

## 首次安装

以下命令在独立 Linux 项目目录执行，需要 Docker Engine、Compose v2、Bash、OpenSSL 和 curl。生产域名应解析到入口，80/443 可供 Caddy 申请证书。示例 `.test` 域名不可直接用于生产。此流程不在现有 Ubuntu 开发依赖上自动上线。

```bash
cp deploy/production.env.example .env.production
sh deploy/init-secrets.sh
mkdir -m 700 deploy/backups deploy/key-backups
```

编辑 `.env.production` 的域名、发布标签、真实邮件地址及空闲 proxy 子网，填写实际操作员 UID/GID；备份目录由该 UID 拥有。使用秘密管理工具填写 `deploy/secrets/` 中的 SMTP/provider 文件。未配置的供应商保持 client ID 与 secret 同时为空。不要把真实秘密写进 `.env.production`、构建参数或命令行。

初始化脚本只接受不存在的秘密目录；默认数据库名 `moetopia`，生成运行和迁移两个不同角色的 DSN，以及数据库密码、Valkey 密码和根加密密钥。调整数据库名时必须同时调整两个 DSN 和 `MOETOPIA_DATABASE_NAME`。恢复根密钥按[恢复指南](run-backup-recovery.md)，不可用随机新密钥代替。

文件经 Compose secrets 挂入容器。普通 Compose 的文件 secret 不提供自动加密存储或可靠的 UID 重映射；依据 [Compose secrets 定义](https://docs.docker.com/reference/compose-file/secrets/)。本工程使用父目录 `0700`、文件 `0444` 允许非 root 容器读取绑定文件；宿主机访问受父目录保护。目录需位于受保护磁盘，不得共享给其他本机用户。Windows ACL 与 Linux 权限不等价。

```bash
sh deploy/compose.sh config --quiet
sh deploy/compose.sh build api account edge backup
sh deploy/compose.sh up -d --wait postgres valkey
sh deploy/compose.sh --profile maintenance run --rm migrate
sh deploy/compose.sh --profile maintenance run --rm permissions
sh deploy/compose.sh run --rm --no-deps --entrypoint /app/keys api init
sh deploy/compose.sh run --rm --no-deps --entrypoint /app/firstparty api
sh deploy/compose.sh run --rm --no-deps --entrypoint /app/administrator api init-client
sh deploy/compose.sh up -d --wait api account edge
sh deploy/compose.sh ps
```

迁移使用 `moetopia_migrator`；普通 API 使用 `moetopia_app`，仅获模块 schema 的 CRUD/序列权限，不能迁移。permissions 任务根据私密密码文件显式设置运行角色密码及默认表权限，必须与 DSN 一致。脚本没有自动 bootstrap 管理员或建立测试账号；首管理员按[管理员指南](run-administration.md)显式指定已有验证账号。

## 升级及故障

先生成并确认可解密的数据库备份与独立根密钥备份，保留上一版本镜像。停止旧 API（同时停止内置任务），更新发布标签并构建镜像；独立运行 migrate、permissions，再启动 API/account/edge。生产不执行 Down；代码回退前先评估新 schema、注销状态和 MFA 状态兼容性。当前升级策略允许短暂停机，不声称无中断滚动升级。

```bash
sh deploy/compose.sh stop api
sh deploy/compose.sh --profile maintenance run --rm migrate
sh deploy/compose.sh --profile maintenance run --rm permissions
sh deploy/compose.sh up -d --wait api account edge
sh deploy/compose.sh logs --tail 100 api
```

不要运行 `down --volumes` 清理生产数据。readyz 失败时检查迁移、密钥、数据库和缓存；未知错误拒绝放行。日志不得包含连接串或邮件正文。Caddy 前再加代理/CDN 时需另行设计直接 peer 的信任链，不能扩大到任意 CIDR。

## 可重复的发布演练

```bash
sudo bash deploy/verify-release.sh
```

该脚本创建随机命名的独立 Compose 项目、测试秘密和数据卷；只绑定宿主机回环 58443/58444，使用 `.test` 域和内部 CA，不操作日常开发库。确保端口及测试子网未占用，同一时间仅跑一轮。它构建全部镜像、验证迁移和运行角色隔离、启动入口、探测匿名接口，并实际执行加密备份/新库恢复/错误密钥拒绝/瞬态凭据失效。结束只清理本轮项目的数据卷，私密产物保留 `.data/release-<nonce>`；不提交或公开其中密钥、日志和备份。生产 TLS、外部邮件和供应商流程仍需真实环境验收。
