# Clash-SpeedTest

基于 Clash/Mihomo 核心的高性能测速与节点检测工具，支持批量测速、延迟探测、可用性过滤以及基于中继代理池的境内可用性检测。

## 特性亮点

1. **零额外依赖**：无需常驻后台或启动额外的 Clash/Mihomo 进程实例，单一二进制工具即可完成全部测试。
2. **格式全兼容**：完整支持 Clash / Mihomo 配置中的 `proxies` 和 `proxy-providers`，涵盖 Shadowsocks、ShadowsocksR、VMess、VLESS、Trojan、Snell、SOCKS5、HTTP、Hysteria2、TUIC 等几乎所有协议。
3. **支持中继代理池检测**：支持通过中继节点（如国内代理池）级联测试目标节点在中国境内的实际可用性，支持设置达标中继数阈值。
4. **灵活过滤与筛选**：支持正则白名单、黑名单关键词过滤、延迟过滤、上行/下行速率过滤，并支持将达标节点重新导出为可用配置。
5. **智能重命名**：可根据 IP 物理归属地（国旗 Emoji、国家代码）及测速结果自动重命名节点。
6. **双运行模式**：既支持终端 CLI 命令行模式，也内置轻量级 Web API 服务模式，方便与自动化工作流集成。

<img width="1332" alt="image" src="https://github.com/user-attachments/assets/fdc47ec5-b626-45a3-a38a-6d88c326c588">

---

## 安装方法

### 方式 1：Go 一键安装
```bash
go install github.com/faceair/clash-speedtest@latest
```

### 方式 2：从源码编译
```bash
git clone https://github.com/faceair/clash-speedtest.git
cd clash-speedtest
go build -o clash-speedtest .
```

---

## 参数详解

运行 `clash-speedtest -h` 可查看完整参数列表：

| 参数 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| **`-c`** | `string` | `""` | 待测试的配置文件路径，**支持本地文件路径、HTTP(S) 订阅链接**，多个路径可用英文逗号 `,` 分隔（必填，Web 模式除外） |
| **`-f`** | `string` | `".+"` | 节点名称正则**白名单**过滤，仅测试名称匹配该正则的节点 |
| **`-b`** | `string` | `""` | 节点名称**黑名单**关键词，使用 `\|` 分隔多个关键词，如 `-b 'rate\|x1\|1x'` |
| **`-fast`** | `bool` | `false` | **快速模式**：仅测试连通性与握手延迟（RTT），跳过后续的下载和上传吞吐测试，大幅节省流量和时间 |
| **`-server-url`** | `string` | `https://speed.cloudflare.com` | 测速服务器地址，默认使用 Cloudflare Speedtest，支持配置自建测速端 |
| **`-download-size`** | `int` | `52428800` (50MB) | 测速时下载测试文件的大小（字节） |
| **`-upload-size`** | `int` | `20971520` (20MB) | 测速时上传测试文件的大小（字节） |
| **`-timeout`** | `duration` | `5s` | 每个测速环节的超时时间，如 `5s`、`10s` |
| **`-concurrent`** | `int` | `4` | 下载/上传并发连接数，快速模式下作为并发探测工作协程数（默认 `10`） |
| **`-max-latency`** | `duration` | `800ms` | 延迟过滤阈值，延迟高于此值的节点将被判定为不合格 |
| **`-min-download-speed`** | `float` | `5` | 下载速度过滤阈值（单位：**MB/s**），低于此值的节点被过滤 |
| **`-min-upload-speed`** | `float` | `2` | 上传速度过滤阈值（单位：**MB/s**），低于此值的节点被过滤 |
| **`-output`** | `string` | `""` | 过滤后达标节点的导出配置文件路径（YAML 格式），为空时不导出文件 |
| **`-rename`** | `bool` | `false` | 按照节点实际出口 IP 归属地与下载速度重命名节点（如 `🇺🇸 US \| ⬇️ 15.67 MB/s`） |
| **`-stash-compatible`** | `bool` | `false` | 开启 Stash 客户端兼容模式，过滤 Stash 不支持的加密算法或协议 |
| **`-relay-pool-url`** | `string` | `""` | **中继代理池地址**（支持远程 HTTP(S) URL 或本地文件），亦可通过环境变量 `RELAY_POOL_URL` 指定。用于境内中继可用性检测 |
| **`-relay-success-count`** | `int` | `1` | 中继模式下，每个目标节点判定成功所需的最少中继连通数 $N$，亦可通过环境变量 `RELAY_SUCCESS_COUNT` 指定 |
| **`-web`** | `bool` | `false` | 启用 Web API 服务模式 |
| **`-port`** | `int` | `8080` | Web API 监听端口（仅在 `-web` 模式下生效） |

---

## 环境变量

除了命令行参数外，程序还支持通过环境变量进行配置（优先级：命令行参数 > 环境变量 > 默认值）：

| 环境变量 | 适用模式 | 默认值 | 支持单位 / 格式 | 说明 |
| :--- | :--- | :--- | :--- | :--- |
| **`MAX_LATENCY`** | CLI / Web | `5000ms` (Web) / `800ms` (CLI) | `600ms`, `1s`, `2.5s`, `800` | **延迟过滤门槛**。超过此延迟的节点将被直接过滤剔除（别名 `SPEEDTEST_MAX_LATENCY`） |
| **`SPEEDTEST_TIMEOUT`** | CLI / Web | `2s` (直连) / `5s` (中继) | `600ms`, `5s`, `10s`, `5` | **测速通信超时**。控制单次中继探测、下载/上传分块的最大等待时间（别名 `TIMEOUT`） |
| **`RELAY_POOL_URL`** | CLI / Web | `""` | URL / 本地路径 | 中继代理池地址。**未设置时为直连模式**；设置后自动开启中继检测模式 |
| **`RELAY_SUCCESS_COUNT`** | CLI / Web | `1` | 整数 | 中继检测模式下，每个节点必须成功的最小中继数量（别名 `RELAY_COUNT` 亦可识别） |
| **`RELAY_SAMPLE_COUNT`** | CLI / Web | `10` | 整数 (`-1` 为全部) | 中继模式下单节点随机抽样的中继数量（别名 `RELAY_TEST_COUNT` / `RELAY_RANDOM_COUNT`） |
| **`AUTH_KEY`** | Web 模式 | `""` | 字符串 | Web API 身份验证密钥，Web 模式启动时**必须设置**（通过请求头 `Authorization: Bearer <AUTH_KEY>` 校验） |
| **`IS_NAME_APPEND`** | Web 模式 | `""` | 字符串 | 是否拼接原始名称 |

---

### `SPEEDTEST_TIMEOUT` 与 `MAX_LATENCY` 的区别与联系

很多用户在配置 Docker 容器或自动化脚本时容易混淆这两个参数，以下是它们的定位与协作关系：

#### 1. 核心区别

| 维度 | `MAX_LATENCY`（最大延迟过滤阈值） | `SPEEDTEST_TIMEOUT`（测速通信超时时间） |
| :--- | :--- | :--- |
| **本质定位** | **质量过滤指标（业务裁决）** | **底层通信保护（防挂死机制）** |
| **核心作用** | 决定节点延迟**多快才算合格**。延迟高于此值的节点直接被丢弃。 | 决定网络请求**最长等待多久**。防止因网络丢包或节点无响应而无限阻塞。 |
| **主要生效环节** | 1. 节点最终结果过滤阶段（`filterResults`）<br>2. 直连延迟探测时的 HTTP Client 硬超时 | 1. 中继模式下单个中继节点的探测握手限时<br>2. 全量测速模式下的下载/上传分块传输限时 |
| **典型场景示例** | 设为 `600ms`：测速耗时 850ms 的节点会被**判定不合格并剔除**。 | 设为 `5s`：若节点 5 秒内毫无响应则直接**中断本次网络请求**。 |

#### 2. 两者的内在联系与协同工作机制

* **直连模式下的协同**：直连探测延迟时，HTTP 客户端直接以 `MAX_LATENCY` 作为最长等待时间。如果在该时间内未收到响应，请求直接报超时失败；若收到响应但延迟大于 `MAX_LATENCY`，同样在结果过滤阶段被剔除。
* **中继模式下的协同**：`SPEEDTEST_TIMEOUT` 负责约束测试端经过多跳中继链时的单次探测超时；而探测成功后汇总出的最优中继延迟，最终由 `MAX_LATENCY` 进行质量把关，超过门槛的节点依然会被剔除。
* **单位格式完全一致**：两者均全面支持 Go 标准时间单位（如 `600ms`、`1.5s`、`5s`、`1m`，支持带引号），并且兼容纯数字输入（大于 100 自动按毫秒 `ms` 计算，小于等于 100 自动按秒 `s` 计算）。

---

## 常见使用场景与实战示例

### 1. 基础全量测速（订阅 URL / 本地文件）
```bash
# 订阅地址建议携带 flag=meta 参数，以便识别全量 Mihomo 扩展协议
clash-speedtest -c 'https://domain.com/api/v1/client/subscribe?token=secret&flag=meta'

# 本地文件测速
clash-speedtest -c ~/.config/clash/config.yaml

# 混合传入多个订阅与本地文件（用英文逗号分隔）
clash-speedtest -c 'https://domain.com/sub1.yaml,/home/user/sub2.yaml'
```

### 2. 快速可用性与延迟测试 (`-fast`)
仅测试节点是否可用以及延迟大小，跳过吞吐测速，耗时少、流量消耗极低：
```bash
clash-speedtest -c config.yaml -fast
```

### 3. 正则与关键词过滤 (`-f` 与 `-b`)
```bash
# 仅测香港或日本节点（名称正则白名单）
clash-speedtest -c config.yaml -f 'HK|港|JP|日'

# 排除倍率、流媒体解锁等特征节点（黑名单关键词）
clash-speedtest -c config.yaml -b '0.1x|rate|x1|游戏'
```

### 4. 筛选合格节点并导出新配置文件 (`-output`)
筛选出延迟 $\le 600\text{ms}$ 且下载速度 $\ge 10\text{MB/s}$ 的优质节点，并另存为新配置：
```bash
clash-speedtest -c config.yaml \
  -max-latency 600ms \
  -min-download-speed 10 \
  -output filtered.yaml
```
> 生成的 `filtered.yaml` 可以直接导入 Clash/Mihomo 客户端，或通过 Gist/自建服务作为 Proxy Provider 引用。

### 5. 自动重命名节点 (`-rename`)
根据测速时的出口 IP 地区与下载速度，自动格式化节点名称：
```bash
clash-speedtest -c config.yaml -rename -output renamed.yaml
```
重命名效果示例：`🇭🇰 HK | ⬇️ 35.80 MB/s`、`🇺🇸 US | ⬇️ 18.25 MB/s`。

---

### 6. 中国境内中继可用性检测（中继代理池模式）

#### 为什么需要中继检测？
很多测试机运行在海外服务器（如 GitHub Actions、海外 VPS 等），直接测试目标节点无法反映**中国境内真实网络环境下**该节点是否可用或是否被阻断。

通过指定一个中国大陆的代理池（例如公共 SOCKS5/HTTP 代理池），工具会以中继链方式发起探测：
$$\text{测试端} \longrightarrow \text{CN 中继节点} \longrightarrow \text{目标代理节点} \longrightarrow \text{目标测速站点}$$

#### 代理池格式支持
中继代理池支持以下两种格式（程序自动识别并缓存 5 分钟）：
1. **逐行代理 URL 格式**（支持 `socks5://`、`http://`、`https://`，支持带用户名密码，自动跳过注释和空行）：
   ```text
   socks5://101.5.21.225:7894
   http://117.72.15.196:8080
   socks5://username:password@123.45.67.89:1080
   ```
2. **标准 Clash YAML 格式**（包含 `proxies:` 列表）。

#### 使用方法
```bash
# 方式 A：通过环境变量设置（推荐在脚本/容器中使用）
export RELAY_POOL_URL="https://raw.githubusercontent.com/dev2people/cn-proxy-node-pool/refs/heads/main/filter-data/allnode-url.txt"
# 要求至少 1 个中继节点连通即视为合格（默认为 1）
export RELAY_SUCCESS_COUNT=1
# 单个节点随机抽取 X 个中继进行测试（默认为 10；若全部失败则视为失败节点；设为 -1 表示不限制测试全部）
export RELAY_SAMPLE_COUNT=10

clash-speedtest -c config.yaml -fast

# 方式 B：通过命令行参数指定
clash-speedtest -c config.yaml \
  -relay-pool-url "https://raw.githubusercontent.com/dev2people/cn-proxy-node-pool/refs/heads/main/filter-data/allnode-url.txt" \
  -relay-success-count 1 \
  -relay-sample-count 10 \
  -fast
```

- **随机抽样与耗时保护**：当代理池节点数量较多时，为避免不可用节点长时间等待全部探测，系统为每个目标节点从代理池中**随机无重复抽取 $X$ 个中继**（默认 10 个，可通过 `RELAY_SAMPLE_COUNT` 或 `-relay-sample-count` 修改）。
- **全协议中继支持**：中继代理池支持 `socks5`、`http`、`https` 代理节点。若使用 HTTP 中继测试基于 UDP 的节点（如 `hysteria2`、`tuic` 等），底层探测报错将自然判定为该中继尝试超时/不可达，若未达标则判定节点整体超时（`N/A`）。
- **并发与早期熔断**：每个目标节点并发调度中继进行探测。一旦满足 $N$ 个中继成功连通，立即取消剩余中继探测，毫秒级返回最优中继与延迟。
- **失败判定与原因明细**：若抽样的 $X$ 个中继全部失败（或成功数 $< N$），则判定目标节点在中国境内不可用（延迟记为 `N/A`），并在日志中打印失败原因明细（如连接超时、连接被拒绝、中继不支持UDP等）。
- **超时与服务端配置**：支持配置 `TIMEOUT` 或 `SPEEDTEST_TIMEOUT`（如 `5s`），启用中继模式时默认放宽至 5 秒以包容多跳嵌套握手耗时；支持通过 `SERVER_URL` 自定义探测目标地址。

---

## Web API 服务模式

提供 HTTP API 接口，适合集成到订阅转换器、定时测速面板或 CI/CD 流水线中。

### Docker / Docker Compose 部署

```yaml
services:
  clash-speedtest:
    restart: unless-stopped
    image: dev2people/clashsub-tools-clashspeedtest-v0
    container_name: clash-speedtest
    ports:
      - "57890:8080"
    environment:
      - TZ=Asia/Shanghai
      - AUTH_KEY=epYeobK02c6ee9ELuCkPAplZYyqAjOJN
      - MAX_LATENCY=600ms          # 延迟门槛（如 600ms, 1s），过滤超限节点
      - SPEEDTEST_TIMEOUT=2s       # 测速/中继超时时间
      # - RELAY_POOL_URL="https://raw.githubusercontent.com/dev2people/cn-proxy-node-pool/refs/heads/main/filter-data/allnode-url.txt"
      # - RELAY_SUCCESS_COUNT=1
      # - RELAY_SAMPLE_COUNT=10
```

### 启动服务
```bash
# 启动 Web 模式需要指定 AUTH_KEY 环境变量
export AUTH_KEY="your-secret-token"
export RELAY_POOL_URL="https://raw.githubusercontent.com/dev2people/cn-proxy-node-pool/refs/heads/main/filter-data/allnode-url.txt" # 可选中继池

clash-speedtest -web -port 8080
```

### API 接口清单

所有需要认证的接口均需在 Header 中携带：
```http
Authorization: Bearer <AUTH_KEY>
```

| 路径 | 方法 | 请求体 | 说明 |
| :--- | :--- | :--- | :--- |
| **`/speedtest`** | `POST` | Clash 配置 YAML | 执行连通性与测速，过滤掉无效节点，返回测速合格节点的 YAML 配置 |
| **`/speedtest_append_name`** | `POST` | Clash 配置 YAML | 执行测速并在每个节点名称后追加测速信息（如速度/延迟），返回更新后的 YAML |
| **`/speedtest_config_filter`** | `POST` | Clash 配置 YAML | **仅做静态语法与协议有效性校验**，不连接网络，快速剔除不合规节点并返回干净 YAML |
| **`/health`** | `GET` | 无 | 服务健康检查接口，成功返回 `{"status":"ok"}`，无需认证 |

#### API 调用示例（cURL）
```bash
# 1. 测速并过滤节点
curl -X POST http://127.0.0.1:8080/speedtest \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: text/yaml" \
  --data-binary @config.yaml -o result.yaml

# 2. 仅过滤格式无效节点（无需网络连通测试，毫秒级响应）
curl -X POST http://127.0.0.1:8080/speedtest_config_filter \
  -H "Authorization: Bearer your-secret-token" \
  --data-binary @raw-subscription.yaml -o clean.yaml
```

---

## 测速与中继原理

### 1. 直连测速原理
1. **延迟（TTFB）**：通过 HTTP GET 向测试端点发起请求，计算从发送请求到接收到首字节的时间。
2. **下载速度**：并发分块通过代理节点下载指定大小的文件（默认 50MB），计算总用时得出传输带宽。
3. **上传速度**：并发分块向服务端提交空数据流（默认 20MB），计算上传带宽。

### 2. 自建测速服务端
默认使用 Cloudflare 测速网络（`https://speed.cloudflare.com`）。如需在特定地域搭建私有测速端：

```bash
# 在测速服务器上安装并运行
go install github.com/faceair/clash-speedtest/download-server@latest
download-server

# 客户端测速时指定自建服务器地址
clash-speedtest -c config.yaml -server-url "http://your-server-ip:8080"
```

---

## License

本项目遵循 [GPL-3.0](LICENSE) 开源协议。
