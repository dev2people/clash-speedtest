package speedtester

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	stdlog "log"
	"math/rand/v2"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/component/proxydialer"
	"github.com/metacubex/mihomo/constant"
	"gopkg.in/yaml.v3"
)

var (
	relayPoolCacheMu   sync.RWMutex
	relayPoolCacheURL  string
	relayPoolCacheExp  time.Time
	relayPoolCacheData []*CProxy
)

// LoadRelayPool 从指定 URL 或本地文件加载中继代理池
func LoadRelayPool(sourceURL string) ([]*CProxy, error) {
	sourceURL = strings.Trim(strings.TrimSpace(sourceURL), "\"'")
	if sourceURL == "" {
		return nil, nil
	}

	// 缓存机制（有效期 5 分钟），避免 Web 模式每次请求重复拉取远程 URL
	relayPoolCacheMu.RLock()
	if relayPoolCacheURL == sourceURL && time.Now().Before(relayPoolCacheExp) && len(relayPoolCacheData) > 0 {
		cached := make([]*CProxy, len(relayPoolCacheData))
		copy(cached, relayPoolCacheData)
		relayPoolCacheMu.RUnlock()
		return cached, nil
	}
	relayPoolCacheMu.RUnlock()

	var body []byte
	var err error

	if strings.HasPrefix(sourceURL, "http://") || strings.HasPrefix(sourceURL, "https://") {
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("拉取中继代理池失败 (%s): %w", sourceURL, err)
		}
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取中继代理池内容失败: %w", err)
		}
	} else {
		body, err = os.ReadFile(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("读取中继代理池文件失败 (%s): %w", sourceURL, err)
		}
	}

	body = CleanYAMLControlCharacters(body)
	proxies := parseRelayPoolContent(body)
	if len(proxies) == 0 {
		return nil, fmt.Errorf("中继代理池中未找到有效可用代理: %s", sourceURL)
	}

	stdlog.Printf("【中继代理池】已成功加载 %d 个可用中继节点 (来源: %s)", len(proxies), sourceURL)

	relayPoolCacheMu.Lock()
	relayPoolCacheURL = sourceURL
	relayPoolCacheExp = time.Now().Add(5 * time.Minute)
	relayPoolCacheData = proxies
	relayPoolCacheMu.Unlock()

	return proxies, nil
}

// parseRelayPoolContent 解析内容（支持 Clash YAML 格式与逐行代理 URL 格式）
func parseRelayPoolContent(data []byte) []*CProxy {
	// 尝试优先按照 Clash YAML 解析
	var rawCfg RawConfig
	if err := yaml.Unmarshal(data, &rawCfg); err == nil && len(rawCfg.Proxies) > 0 {
		var list []*CProxy
		for i, cfg := range rawCfg.Proxies {
			// 中继节点只使用 socks5，过滤掉无关或不兼容协议
			if t, ok := cfg["type"].(string); ok {
				t = strings.ToLower(strings.TrimSpace(t))
				if t != "socks5" && t != "socks" {
					continue
				}
			} else {
				continue
			}

			p, err := adapter.ParseProxy(cfg)
			if err != nil {
				continue
			}
			name := p.Name()
			if name == "" {
				name = fmt.Sprintf("relay-yaml-%d", i+1)
			}
			list = append(list, &CProxy{Proxy: p, Config: cfg})
		}
		if len(list) > 0 {
			return list
		}
	}

	// 逐行解析 URL 格式 (仅支持 socks5://...)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var list []*CProxy
	idx := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		proxyCfg, err := parseProxyFromURL(line, idx)
		if err != nil || proxyCfg == nil {
			continue
		}

		proxy, err := adapter.ParseProxy(proxyCfg)
		if err != nil {
			continue
		}

		list = append(list, &CProxy{Proxy: proxy, Config: proxyCfg})
		idx++
	}
	return list
}

// parseProxyFromURL 将单行代理 URL 解析为 Mihomo 代理配置 Map
func parseProxyFromURL(line string, idx int) (map[string]any, error) {
	u, err := url.Parse(line)
	if err != nil {
		return nil, err
	}

	scheme := strings.ToLower(u.Scheme)
	host := u.Hostname()
	portStr := u.Port()
	if host == "" || portStr == "" {
		return nil, fmt.Errorf("缺少主机或端口: %s", line)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("端口无效: %s", portStr)
	}

	// 中继节点仅支持 socks5
	var proxyType string
	switch scheme {
	case "socks5", "socks":
		proxyType = "socks5"
	default:
		return nil, fmt.Errorf("中继仅支持 socks5 协议，忽略非 socks5: %s", scheme)
	}

	proxyCfg := map[string]any{
		"name":   fmt.Sprintf("relay-%d-%s:%d", idx+1, host, port),
		"type":   proxyType,
		"server": host,
		"port":   port,
	}

	if scheme == "https" {
		proxyCfg["tls"] = true
	}

	if u.User != nil {
		proxyCfg["username"] = u.User.Username()
		if pwd, ok := u.User.Password(); ok {
			proxyCfg["password"] = pwd
		}
	}

	return proxyCfg, nil
}

// relayTestResult 单个中继测试结果
type relayTestResult struct {
	latency      time.Duration
	relayedProxy constant.Proxy
	relayName    string
	err          error
}

// selectRelaySample 从中继代理池中随机抽取 sampleCount 个中继节点（不重复）
func selectRelaySample(relays []*CProxy, sampleCount int) []*CProxy {
	n := len(relays)
	if n == 0 {
		return nil
	}
	// sampleCount <= 0 或大于等于总数时全部使用，并随机打乱顺序
	if sampleCount <= 0 || sampleCount >= n {
		selected := make([]*CProxy, n)
		copy(selected, relays)
		rand.Shuffle(n, func(i, j int) {
			selected[i], selected[j] = selected[j], selected[i]
		})
		return selected
	}

	perm := rand.Perm(n)
	selected := make([]*CProxy, sampleCount)
	for i := 0; i < sampleCount; i++ {
		selected[i] = relays[perm[i]]
	}
	return selected
}

// isUDPOnlyProxy 判断目标节点协议是否必须依赖 UDP 传输 (QUIC 等)
func isUDPOnlyProxy(proxy *CProxy) bool {
	var pType string
	if proxy.Config != nil {
		if t, ok := proxy.Config["type"].(string); ok {
			pType = strings.ToLower(t)
		}
	}
	if pType == "" && proxy.Proxy != nil {
		pType = strings.ToLower(proxy.Type().String())
	}
	switch pType {
	case "hysteria", "hysteria2", "hy", "hy2", "tuic", "wireguard", "wg":
		return true
	default:
		return false
	}
}

// relaySupportsUDP 判断中继节点是否支持转发 UDP 数据包
func relaySupportsUDP(relay *CProxy) bool {
	var pType string
	if relay.Config != nil {
		if t, ok := relay.Config["type"].(string); ok {
			pType = strings.ToLower(t)
		}
	}
	if pType == "" && relay.Proxy != nil {
		pType = strings.ToLower(relay.Type().String())
	}
	// HTTP / HTTPS 代理协议为标准 CONNECT 隧道，不支持 UDP 转发 (Mihomo 会直接返回 no support)
	if pType == "http" || pType == "https" {
		return false
	}
	return true
}

// simplifyRelayError 提取简明易懂的中继失败错误原因
func simplifyRelayError(err error) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "no support") {
		return "中继协议不支持UDP"
	}
	if strings.Contains(errMsg, "context deadline exceeded") || strings.Contains(errMsg, "timeout") {
		return "连接超时"
	}
	if strings.Contains(errMsg, "connection refused") {
		return "连接被拒绝"
	}
	if strings.Contains(errMsg, "no route to host") {
		return "无法路由到主机"
	}
	if strings.Contains(errMsg, "proxyconnect tcp") {
		return "中继代理握手失败"
	}
	if len(errMsg) > 60 {
		return errMsg[:60] + "..."
	}
	return errMsg
}

// testProxyWithRelayPool 使用中继代理池测试目标节点连通性
func (st *SpeedTester) testProxyWithRelayPool(name string, proxy *CProxy) *Result {
	result := &Result{
		ProxyName:   name,
		ProxyType:   proxy.Type().String(),
		ProxyConfig: proxy.Config,
		Proxy:       proxy,
	}

	if len(st.relayProxies) == 0 {
		return result
	}

	availableRelays := st.relayProxies
	if isUDPOnlyProxy(proxy) {
		var udpRelays []*CProxy
		for _, r := range st.relayProxies {
			if relaySupportsUDP(r) {
				udpRelays = append(udpRelays, r)
			}
		}
		if len(udpRelays) == 0 {
			result.RelayUsed = true
			result.RelayTestCount = 0
			result.RelaySuccessCount = 0
			result.RelayFailureReason = fmt.Sprintf("目标节点为 %s (需UDP)，但中继池无支持UDP的中继(均为HTTP代理)", proxy.Type().String())
			stdlog.Printf("【中继过滤】节点 [%s] 协议为 %s (需UDP)，但中继池中无支持UDP的中继节点(均为HTTP代理)，跳过测试", name, proxy.Type().String())
			return result
		}
		availableRelays = udpRelays
	}

	sampleCount := st.config.RelaySampleCount
	if sampleCount == 0 {
		sampleCount = 10
	}
	// 若显式配置为负数（如 -1），则测试全部可用中继
	if sampleCount < 0 || sampleCount > len(availableRelays) {
		sampleCount = len(availableRelays)
	}

	candidates := selectRelaySample(availableRelays, sampleCount)

	requiredSuccess := st.config.RelaySuccessCount
	if requiredSuccess <= 0 {
		requiredSuccess = 1
	}
	if requiredSuccess > len(candidates) {
		requiredSuccess = len(candidates)
	}

	// 快速连接测试目标 URL
	testURL := fmt.Sprintf("%s/__down?bytes=0", st.config.ServerURL)
	if st.config.FastMode {
		if st.config.ServerURL != "" && st.config.ServerURL != "https://speed.cloudflare.com" {
			testURL = st.config.ServerURL
		} else {
			testURL = "https://www.google.com/generate_204"
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type relayJob struct {
		index int
		relay *CProxy
	}

	jobs := make(chan relayJob, len(candidates))
	for i, r := range candidates {
		jobs <- relayJob{index: i, relay: r}
	}
	close(jobs)

	// 并发中继探测 worker 数量（每个节点最多并发探测 8 个中继，避免瞬间网络拥塞）
	workerCount := 8
	if workerCount > len(candidates) {
		workerCount = len(candidates)
	}

	var (
		mu            sync.Mutex
		successCount  int32
		bestLatency   time.Duration
		bestRelayed   constant.Proxy
		bestRelayName string
		latencies     []time.Duration
		failReasons   []string
		wg            sync.WaitGroup
	)

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// 检查是否已达到目标成功数或已取消
				if atomic.LoadInt32(&successCount) >= int32(requiredSuccess) || ctx.Err() != nil {
					return
				}

				latency, relayedTarget, err := st.testSingleRelay(ctx, proxy, job.relay, testURL)
				if err == nil {
					mu.Lock()
					latencies = append(latencies, latency)
					if bestLatency == 0 || latency < bestLatency {
						bestLatency = latency
						bestRelayed = relayedTarget
						bestRelayName = job.relay.Name()
					}
					newCount := atomic.AddInt32(&successCount, 1)
					if newCount >= int32(requiredSuccess) {
						cancel() // 已达标，立刻取消其余探测任务
					}
					mu.Unlock()
				} else {
					mu.Lock()
					failReasons = append(failReasons, fmt.Sprintf("%s: %s", job.relay.Name(), simplifyRelayError(err)))
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	result.RelayUsed = true
	result.RelaySuccessCount = int(atomic.LoadInt32(&successCount))
	result.RelayTestCount = len(candidates)
	result.BestRelayName = bestRelayName

	// 判断是否满足 N 个中继成功测试
	if int(atomic.LoadInt32(&successCount)) < requiredSuccess {
		// 未达到目标中继成功数量或全部中继连接不通，视为失败
		if len(failReasons) > 0 {
			sample := failReasons
			if len(sample) > 3 {
				sample = sample[:3]
			}
			result.RelayFailureReason = strings.Join(sample, "; ")
		} else {
			result.RelayFailureReason = "未达到达标中继数"
		}
		stdlog.Printf("【中继未达标】节点 [%s] (%d/%d 成功, 测试 %d 个中继), 原因: %s",
			name, result.RelaySuccessCount, requiredSuccess, len(candidates), result.RelayFailureReason)
		//未通过就是0
		result.Latency = 0
		return result
	}

	// 成功：记录延迟（取可用中继路径中的最优延迟）
	result.Latency = bestLatency

	// 快速测试模式直接返回
	if st.config.FastMode {
		return result
	}

	if st.config.MaxLatency > 0 && result.Latency > st.config.MaxLatency {
		return result
	}

	// 普通模式：基于成功的中继路径继续执行下载与上传测试
	if bestRelayed == nil {
		return result
	}

	// 并发下载测试
	var dlWg sync.WaitGroup
	var totalDownloadBytes, totalUploadBytes int64
	var totalDownloadTime, totalUploadTime time.Duration
	var downloadCount, uploadCount int

	downloadChunkSize := st.config.DownloadSize / st.config.Concurrent
	if downloadChunkSize > 0 {
		downloadResults := make(chan *downloadResult, st.config.Concurrent)
		for i := 0; i < st.config.Concurrent; i++ {
			dlWg.Add(1)
			go func() {
				defer dlWg.Done()
				downloadResults <- st.testDownload(bestRelayed, downloadChunkSize, st.config.Timeout)
			}()
		}
		dlWg.Wait()
		for i := 0; i < st.config.Concurrent; i++ {
			if dr := <-downloadResults; dr != nil {
				totalDownloadBytes += dr.bytes
				totalDownloadTime += dr.duration
				downloadCount++
			}
		}
		close(downloadResults)
		if downloadCount > 0 {
			result.DownloadSize = float64(totalDownloadBytes)
			result.DownloadTime = totalDownloadTime / time.Duration(downloadCount)
			result.DownloadSpeed = float64(totalDownloadBytes) / result.DownloadTime.Seconds()
		}
		if result.DownloadSpeed < st.config.MinDownloadSpeed {
			return result
		}
	}

	// 上传测试
	uploadChunkSize := st.config.UploadSize / st.config.Concurrent
	if uploadChunkSize > 0 {
		uploadResults := make(chan *downloadResult, st.config.Concurrent)
		for i := 0; i < st.config.Concurrent; i++ {
			dlWg.Add(1)
			go func() {
				defer dlWg.Done()
				uploadResults <- st.testUpload(bestRelayed, uploadChunkSize, st.config.Timeout)
			}()
		}
		dlWg.Wait()
		for i := 0; i < st.config.Concurrent; i++ {
			if ur := <-uploadResults; ur != nil {
				totalUploadBytes += ur.bytes
				totalUploadTime += ur.duration
				uploadCount++
			}
		}
		close(uploadResults)
		if uploadCount > 0 {
			result.UploadSize = float64(totalUploadBytes)
			result.UploadTime = totalUploadTime / time.Duration(uploadCount)
			result.UploadSpeed = float64(totalUploadBytes) / result.UploadTime.Seconds()
		}
	}

	return result
}

// testSingleRelay 通过单个中继测试目标节点
func (st *SpeedTester) testSingleRelay(ctx context.Context, targetProxy *CProxy, relayProxy *CProxy, testURL string) (time.Duration, constant.Proxy, error) {
	relayDialer := proxydialer.New(relayProxy.Proxy, false)
	relayedTarget, err := adapter.ParseProxy(targetProxy.Config, adapter.WithDialerForAPI(relayDialer))
	if err != nil {
		return 0, nil, fmt.Errorf("创建中继目标节点失败: %w", err)
	}

	// 限制单个中继的探测超时
	timeout := st.config.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	reqCtx, reqCancel := context.WithTimeout(ctx, timeout)
	defer reqCancel()

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			var u16Port uint16
			if port, err := strconv.ParseUint(port, 10, 16); err == nil {
				u16Port = uint16(port)
			}
			return relayedTarget.DialContext(ctx, &constant.Metadata{
				Host:    host,
				DstPort: u16Port,
			})
		},
		DisableKeepAlives: true,
	}
	defer tr.CloseIdleConnections()

	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 禁止跟随任何重定向（防止公共中继返回 302 劫持到认证页伪造 200 成功）
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(reqCtx, "GET", testURL, nil)
	if err != nil {
		return 0, nil, err
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if strings.Contains(testURL, "generate_204") {
		// generate_204 探测端点要求严格：必须返回 204 No Content，或者无 Body 的 200
		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			return 0, nil, fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode)
		}
		if resp.ContentLength > 0 {
			return 0, nil, fmt.Errorf("generate_204 受到中继页面劫持 (Content-Length: %d)", resp.ContentLength)
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, nil, fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode)
	}

	return time.Since(start), relayedTarget, nil
}
