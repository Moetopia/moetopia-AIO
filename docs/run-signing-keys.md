# 签名密钥运行与轮换

状态：已有常规轮换、维护窗口应急轮换和持久化运行路径，开发验证见 [实施进度](implementation-status.md)。不默认执行轮换，不代表生产下游已经完成验收。

## 常规操作

初次部署先迁移，再按现有 `pnpm api:keys:init` 初始化；已有 active key 时初始化明确拒绝，不用它轮换。更换已存在的正常密钥使用：

```powershell
pnpm api:keys:rotate
```

命令从当前 `.env` 读取目标，执行前核对环境与数据库；不得指向其他项目或用临时测试库结果推断日常库已经轮换。运行者必须拥有主机、数据库及加密根密钥访问权，不提供公网轮换接口。日志不输出私钥。密钥 CLI 的审计主体为 system:keys-cli，操作人员归属需结合主机访问记录，不把该值当作已验证的人类管理员。

新 RSA 3072 位密钥在持锁前生成并加密；事务锁定当前 active 行，停用旧密钥、保留十五分钟重叠窗口，创建唯一 active 新密钥并写入审计。失败整笔回滚。并发轮换中竞争失败者需要重新检查并重试，不自动连续生成多个版本。没有新迁移，复用原有 signing_keys.active/retire_after。

## 运行一致性

真实 API 使用 NewPersistentProvider；固定密钥构造器只服务显式注入密钥的隔离测试。Token 事务先用共享行锁读取并解密 active 密钥，固定在请求 context；JWT access token 和 ID Token 最终签名入口强制使用同一密钥的 kid。即使授权码和刷新快照保存旧 header，也不能签出新私钥配旧 kid 的令牌。

轮换更新等待已开始的签发事务完成，因此重叠窗口从旧签发结束后开始计算。新请求从持久化状态选密钥，无需重启 API；短暂竞争、无 active、错误加密根或损坏密钥时失败关闭，不退回进程启动时的旧私钥。Fosite 继续负责令牌生成/格式与协议校验，项目签名适配仅提供请求密钥和按 kid 选择的受控验证公钥。

JWKS 每次读取当前有效公钥，不加载或返回私钥；保留 active 与 retire_after 尚未到期的旧公钥，带 no-store。UserInfo/内省按 kid 读取同一有效公钥集合，拒绝未知 kid、错误算法及过期密钥；内省存储失败继续返回 503，不伪装为 inactive。

当前 access/ID Token 最长十分钟，常规旧公钥保留十五分钟。若后续提高 TTL 或时钟容差，必须同步调整窗口并重新验证。刷新令牌和授权码是独立 HMAC 凭据，常规 RSA 轮换不使合法刷新族失效，旧授权码兑换和刷新后签出当前密钥的新令牌。自有 Cookie 与加密根密钥均不随该命令更换。

worker 在旧密钥 retire_after 到期后清空加密私钥材料；保留非秘密公钥和时间元数据用于运维核对，停止发布与在线接受该 kid。测试中提前推进到期仅用于验证，不提供日常任意缩短窗口的参数。

## 性能与发布边界

当前规模优先一致性：每次签发多一次索引查询、私钥解密及解析；验签多一次按 kid 公钥查询。不引入无法证明失效时序的私钥缓存。尚未测量生产硬件的吞吐和延迟；如数据库访问成为瓶颈，应测量后设计带失效与版本证据的缓存，而不是删除实时状态检查。

轮换前必须将所有 API 实例升级到持久化密钥路径。旧进程若仍固定持有启动私钥，会继续签旧令牌，十五分钟窗口无法保护长期遗留实例；不能混跑后宣称热轮换成功。确认下游 RP/资源服务器在未知 kid 时刷新 JWKS、不永久固定公钥；多副本及实际资源服务器的验证尚未完成。

## 维护窗口应急轮换

私钥泄露不能使用常规重叠窗口。先停止全部 issuer API 实例（包含内置 worker）及其他相关维护任务，确认数据库/根密钥目标，再显式执行：

```powershell
pnpm api:keys:emergency --acknowledge-global-revocation --reason "Confirmed signing key exposure"
```

生产停止 API 后可通过 `sh deploy/compose.sh run --rm --no-deps --entrypoint /app/keys api emergency --acknowledge-global-revocation --reason "Confirmed signing key exposure"` 执行。命令不自动停止进程，不能在仍有写入者时当作在线撤销使用。

新 RSA 密钥、全部旧公钥立即退出 JWKS、旧私钥清空、全局瞬态凭据撤销和原因审计同事务提交；失败全部回滚。瞬态撤销范围由[恢复指南](run-backup-recovery.md)定义，长期密码、TOTP 和外部绑定保持，用户必须重新登录/授权。只启动全部升级后的实例，核验新 JWKS 并通知下游刷新信任。未对日常开发库执行此命令。

下游缓存的旧 JWKS、离线 JWT 及 RP 自己的会话不会被服务器命令瞬间删除，实际应急仍需协调下游。根加密密钥泄露涉及另一组长期秘密，不能由 RSA 轮换解决。当前测试证明本项目事务和在线协议行为，不代表所有外部 RP 已完成生产事故演练。

协议依据：[OIDC Core 10.1.1](https://openid.net/specs/openid-connect-core-1_0.html#RotateSigKeys)。
