package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/faceair/clash-speedtest/speedtester"
	"github.com/faceair/clash-speedtest/webserver"
	"github.com/google/uuid"
	"github.com/metacubex/mihomo/log"
	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/yaml.v3"
)

var (
	configPathsConfig = flag.String("c", "", "config file path, also support http(s) url")
	filterRegexConfig = flag.String("f", ".+", "filter proxies by name, use regexp")
	blockKeywords     = flag.String("b", "", "block proxies by keywords, use | to separate multiple keywords (example: -b 'rate|x1|1x')")
	serverURL         = flag.String("server-url", "https://speed.cloudflare.com", "server url")
	downloadSize      = flag.Int("download-size", 50*1024*1024, "download size for testing proxies")
	uploadSize        = flag.Int("upload-size", 20*1024*1024, "upload size for testing proxies")
	timeout           = flag.Duration("timeout", time.Second*5, "timeout for testing proxies")
	concurrent        = flag.Int("concurrent", 4, "download concurrent size")
	outputPath        = flag.String("output", "", "output config file path")
	stashCompatible   = flag.Bool("stash-compatible", false, "enable stash compatible mode")
	maxLatency        = flag.Duration("max-latency", 800*time.Millisecond, "filter latency greater than this value")
	minDownloadSpeed  = flag.Float64("min-download-speed", 5, "filter download speed less than this value(unit: MB/s)")
	minUploadSpeed    = flag.Float64("min-upload-speed", 2, "filter upload speed less than this value(unit: MB/s)")
	renameNodes       = flag.Bool("rename", false, "rename nodes with IP location and speed")
	fastMode          = flag.Bool("fast", false, "fast mode, only test latency")
	webMode           = flag.Bool("web", false, "enable web server mode")
	webPort           = flag.Int("port", 8080, "web server port (only used in web mode)")
	relayPoolURL      = flag.String("relay-pool-url", "", "relay pool url or file path (also reads RELAY_POOL_URL env)")
	relaySuccessCount = flag.Int("relay-success-count", 0, "minimum successful relays required (default: 1, also reads RELAY_SUCCESS_COUNT env)")
	relaySampleCount  = flag.Int("relay-sample-count", 0, "sample count of relays to randomly test per node (default: 10, also reads RELAY_SAMPLE_COUNT env)")
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

func main() {
	flag.Parse()
	log.SetLevel(log.SILENT)

	// Web 模式
	if *webMode {
		server, err := webserver.New(*webPort)
		if err != nil {
			log.Fatalln("初始化 Web 服务器失败: %v", err)
		}

		if err := server.Start(); err != nil {
			log.Fatalln("启动 Web 服务器失败: %v", err)
		}
		return
	}

	// CLI 模式
	if *configPathsConfig == "" {
		log.Fatalln("please specify the configuration file")
	}

	speedTester := speedtester.New(&speedtester.Config{
		ConfigPaths:       *configPathsConfig,
		FilterRegex:       *filterRegexConfig,
		BlockRegex:        *blockKeywords,
		ServerURL:         *serverURL,
		DownloadSize:      *downloadSize,
		UploadSize:        *uploadSize,
		Timeout:           *timeout,
		Concurrent:        *concurrent,
		MaxLatency:        *maxLatency,
		MinDownloadSpeed:  *minDownloadSpeed * 1024 * 1024,
		MinUploadSpeed:    *minUploadSpeed * 1024 * 1024,
		FastMode:          *fastMode,
		RelayPoolURL:      *relayPoolURL,
		RelaySuccessCount: *relaySuccessCount,
		RelaySampleCount:  *relaySampleCount,
	})

	allProxies, err := speedTester.LoadProxies(*stashCompatible)
	if err != nil {
		log.Fatalln("load proxies failed: %v", err)
	}

	if speedTester.RelayProxyCount() > 0 {
		fmt.Printf("【中继模式】已启用，中继池可用节点数: %d，单节点随机测试中继数: %d，要求达标中继数: %d\n", speedTester.RelayProxyCount(), speedTester.Config().RelaySampleCount, speedTester.Config().RelaySuccessCount)
	} else {
		fmt.Println("【直连模式】未配置中继代理池，使用直连测速")
	}

	bar := progressbar.Default(int64(len(allProxies)), "测试中...")
	results := make([]*speedtester.Result, 0)
	speedTester.TestProxies(allProxies, func(result *speedtester.Result) {
		bar.Add(1)
		bar.Describe(result.ProxyName)
		results = append(results, result)
	})

	sort.Slice(results, func(i, j int) bool {
		return results[i].DownloadSpeed > results[j].DownloadSpeed
	})

	printResults(results)

	if *outputPath != "" {
		err = saveConfig(results, speedTester)
		if err != nil {
			log.Fatalln("save config file failed: %v", err)
		}
		fmt.Printf("\nsave config file to: %s\n", *outputPath)
	}
}

func printResults(results []*speedtester.Result) {
	table := tablewriter.NewWriter(os.Stdout)

	var headers []string
	if *fastMode {
		headers = []string{
			"序号",
			"节点名称",
			"类型",
			"延迟",
		}
	} else {
		headers = []string{
			"序号",
			"节点名称",
			"类型",
			"延迟",
			"抖动",
			"丢包率",
			"下载速度",
			"上传速度",
		}
	}
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.SetColMinWidth(0, 4)  // 序号
	table.SetColMinWidth(1, 20) // 节点名称
	table.SetColMinWidth(2, 8)  // 类型
	table.SetColMinWidth(3, 8)  // 延迟
	if !*fastMode {
		table.SetColMinWidth(4, 8)  // 抖动
		table.SetColMinWidth(5, 8)  // 丢包率
		table.SetColMinWidth(6, 12) // 下载速度
		table.SetColMinWidth(7, 12) // 上传速度
	}

	for i, result := range results {
		idStr := fmt.Sprintf("%d.", i+1)

		// 延迟颜色
		latencyStr := result.FormatLatency()
		if result.Latency > 0 {
			if result.Latency < 800*time.Millisecond {
				latencyStr = colorGreen + latencyStr + colorReset
			} else if result.Latency < 1500*time.Millisecond {
				latencyStr = colorYellow + latencyStr + colorReset
			} else {
				latencyStr = colorRed + latencyStr + colorReset
			}
		} else {
			latencyStr = colorRed + latencyStr + colorReset
		}
		jitterStr := result.FormatJitter()
		if result.Jitter > 0 {
			if result.Jitter < 800*time.Millisecond {
				jitterStr = colorGreen + jitterStr + colorReset
			} else if result.Jitter < 1500*time.Millisecond {
				jitterStr = colorYellow + jitterStr + colorReset
			} else {
				jitterStr = colorRed + jitterStr + colorReset
			}
		} else {
			jitterStr = colorRed + jitterStr + colorReset
		}
		// 丢包率颜色
		packetLossStr := result.FormatPacketLoss()
		if result.PacketLoss < 10 {
			packetLossStr = colorGreen + packetLossStr + colorReset
		} else if result.PacketLoss < 20 {
			packetLossStr = colorYellow + packetLossStr + colorReset
		} else {
			packetLossStr = colorRed + packetLossStr + colorReset
		}
		// 下载速度颜色 (以MB/s为单位判断)
		downloadSpeed := result.DownloadSpeed / (1024 * 1024)
		downloadSpeedStr := result.FormatDownloadSpeed()
		if downloadSpeed >= 10 {
			downloadSpeedStr = colorGreen + downloadSpeedStr + colorReset
		} else if downloadSpeed >= 5 {
			downloadSpeedStr = colorYellow + downloadSpeedStr + colorReset
		} else {
			downloadSpeedStr = colorRed + downloadSpeedStr + colorReset
		}
		// 上传速度颜色
		uploadSpeed := result.UploadSpeed / (1024 * 1024)
		uploadSpeedStr := result.FormatUploadSpeed()
		if uploadSpeed >= 5 {
			uploadSpeedStr = colorGreen + uploadSpeedStr + colorReset
		} else if uploadSpeed >= 2 {
			uploadSpeedStr = colorYellow + uploadSpeedStr + colorReset
		} else {
			uploadSpeedStr = colorRed + uploadSpeedStr + colorReset
		}
		var row []string
		if *fastMode {
			row = []string{
				idStr,
				result.ProxyName,
				result.ProxyType,
				latencyStr,
			}
		} else {
			row = []string{
				idStr,
				result.ProxyName,
				result.ProxyType,
				latencyStr,
				jitterStr,
				packetLossStr,
				downloadSpeedStr,
				uploadSpeedStr,
			}
		}

		table.Append(row)
	}
	fmt.Println()
	table.Render()
	fmt.Println()
}

func saveConfig(results []*speedtester.Result, speedTester *speedtester.SpeedTester) error {
	proxies := make([]map[string]any, 0)

	// Filter results first
	var validResults []*speedtester.Result
	for _, result := range results {
		if *maxLatency > 0 && result.Latency > *maxLatency {
			continue
		}
		if result.Latency == 0 {
			continue
		}
		if !*fastMode {
			if *downloadSize > 0 && *minDownloadSpeed > 0 && result.DownloadSpeed < *minDownloadSpeed*1024*1024 {
				continue
			}
			if *uploadSize > 0 && *minUploadSpeed > 0 && result.UploadSpeed < *minUploadSpeed*1024*1024 {
				continue
			}
		}
		validResults = append(validResults, result)
	}
	if *renameNodes {
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, *concurrent)
		for _, result := range validResults {
			wg.Add(1)
			go func(r *speedtester.Result) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				location, err := speedTester.GetIPLocation(r.Proxy)
				countryCode := "UNKNOWN"
				if err == nil && location.CountryCode != "" {
					countryCode = location.CountryCode
				}
				proxyConfig := r.ProxyConfig
				if *fastMode {
					newUUID := uuid.New().String()
					proxyConfig["name"] = fmt.Sprintf("%s|%s|%s|%dms|%s",
						countryNames[countryCode],
						countryCode,
						countryFlags[countryCode],
						r.Latency.Milliseconds(),
						newUUID)
				} else {
					proxyConfig["name"] = generateNodeName(countryCode, r.DownloadSpeed)
				}
			}(result)
		}
		wg.Wait()
	}

	for _, result := range validResults {
		proxies = append(proxies, result.ProxyConfig)
	}

	config := &speedtester.RawConfig{
		Proxies: proxies,
	}
	if len(proxies) == 0 {
		log.Warnln("No proxy available,No output!")
		return nil
	}
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(*outputPath, yamlData, 0o644)
}

func generateNodeName(countryCode string, downloadSpeed float64) string {
	flag, exists := countryFlags[strings.ToUpper(countryCode)]
	if !exists {
		flag = "🏳️"
	}

	speedMBps := downloadSpeed / (1024 * 1024)
	return fmt.Sprintf("%s %s | ⬇️ %.2f MB/s", flag, strings.ToUpper(countryCode), speedMBps)
}

var countryFlags = map[string]string{
	"US": "🇺🇸", "CN": "🇨🇳", "GB": "🇬🇧", "UK": "🇬🇧", "JP": "🇯🇵", "DE": "🇩🇪", "FR": "🇫🇷", "RU": "🇷🇺",
	"SG": "🇸🇬", "HK": "🇭🇰", "TW": "🇹🇼", "KR": "🇰🇷", "CA": "🇨🇦", "AU": "🇦🇺", "NL": "🇳🇱", "IT": "🇮🇹",
	"ES": "🇪🇸", "SE": "🇸🇪", "NO": "🇳🇴", "DK": "🇩🇰", "FI": "🇫🇮", "CH": "🇨🇭", "AT": "🇦🇹", "BE": "🇧🇪",
	"BR": "🇧🇷", "IN": "🇮🇳", "TH": "🇹🇭", "MY": "🇲🇾", "VN": "🇻🇳", "PH": "🇵🇭", "ID": "🇮🇩", "UA": "🇺🇦",
	"TR": "🇹🇷", "IL": "🇮🇱", "AE": "🇦🇪", "SA": "🇸🇦", "EG": "🇪🇬", "ZA": "🇿🇦", "NG": "🇳🇬", "KE": "🇰🇪",
	"RO": "🇷🇴", "PL": "🇵🇱", "CZ": "🇨🇿", "HU": "🇭🇺", "BG": "🇧🇬", "HR": "🇭🇷", "SI": "🇸🇮", "SK": "🇸🇰",
	"LT": "🇱🇹", "LV": "🇱🇻", "EE": "🇪🇪", "PT": "🇵🇹", "GR": "🇬🇷", "IE": "🇮🇪", "LU": "🇱🇺", "MT": "🇲🇹",
	"CY": "🇨🇾", "IS": "🇮🇸", "MX": "🇲🇽", "AR": "🇦🇷", "CL": "🇨🇱", "CO": "🇨🇴", "PE": "🇵🇪", "VE": "🇻🇪",
	"EC": "🇪🇨", "UY": "🇺🇾", "PY": "🇵🇾", "BO": "🇧🇴", "CR": "🇨🇷", "PA": "🇵🇦", "GT": "🇬🇹", "HN": "🇭🇳",
	"SV": "🇸🇻", "NI": "🇳🇮", "BZ": "🇧🇿", "JM": "🇯🇲", "TT": "🇹🇹", "BB": "🇧🇧", "GD": "🇬🇩", "LC": "🇱🇨",
	"VC": "🇻🇨", "AG": "🇦🇬", "DM": "🇩🇲", "KN": "🇰🇳", "BS": "🇧🇸", "CU": "🇨🇺", "DO": "🇩🇴", "HT": "🇭🇹",
	"PR": "🇵🇷", "VI": "🇻🇮", "GU": "🇬🇺", "AS": "🇦🇸", "MP": "🇲🇵", "PW": "🇵🇼", "FM": "🇫🇲", "MH": "🇲🇭",
	"KI": "🇰🇮", "TV": "🇹🇻", "NR": "🇳🇷", "WS": "🇼🇸", "TO": "🇹🇴", "FJ": "🇫🇯", "VU": "🇻🇺", "SB": "🇸🇧",
	"PG": "🇵🇬", "NC": "🇳🇨", "PF": "🇵🇫", "WF": "🇼🇫", "CK": "🇨🇰", "NU": "🇳🇺", "TK": "🇹🇰", "SC": "🇸🇨",
	"UNKNOWN": "🏳️",
}
var countryNames = map[string]string{
	"US": "美国", "CN": "中国", "GB": "英国", "UK": "英国", "JP": "日本", "DE": "德国", "FR": "法国", "RU": "俄罗斯",
	"SG": "新加坡", "HK": "香港", "TW": "台湾", "KR": "韩国", "CA": "加拿大", "AU": "澳大利亚", "NL": "荷兰", "IT": "意大利",
	"ES": "西班牙", "SE": "瑞典", "NO": "挪威", "DK": "丹麦", "FI": "芬兰", "CH": "瑞士", "AT": "奥地利", "BE": "比利时",
	"BR": "巴西", "IN": "印度", "TH": "泰国", "MY": "马来西亚", "VN": "越南", "PH": "菲律宾", "ID": "印度尼西亚", "UA": "乌克兰",
	"TR": "土耳其", "IL": "以色列", "AE": "阿联酋", "SA": "沙特阿拉伯", "EG": "埃及", "ZA": "南非", "NG": "尼日利亚", "KE": "肯尼亚",
	"RO": "罗马尼亚", "PL": "波兰", "CZ": "捷克", "HU": "匈牙利", "BG": "保加利亚", "HR": "克罗地亚", "SI": "斯洛文尼亚", "SK": "斯洛伐克",
	"LT": "立陶宛", "LV": "拉脱维亚", "EE": "爱沙尼亚", "PT": "葡萄牙", "GR": "希腊", "IE": "爱尔兰", "LU": "卢森堡", "MT": "马耳他",
	"CY": "塞浦路斯", "IS": "冰岛", "MX": "墨西哥", "AR": "阿根廷", "CL": "智利", "CO": "哥伦比亚", "PE": "秘鲁", "VE": "委内瑞拉",
	"EC": "厄瓜多尔", "UY": "乌拉圭", "PY": "巴拉圭", "BO": "玻利维亚", "CR": "哥斯达黎加", "PA": "巴拿马", "GT": "危地马拉", "HN": "洪都拉斯",
	"SV": "萨尔瓦多", "NI": "尼加拉瓜", "BZ": "伯利兹", "JM": "牙买加", "TT": "特立尼达和多巴哥", "BB": "巴巴多斯", "GD": "格林纳达", "LC": "圣卢西亚",
	"VC": "圣文森特和格林纳丁斯", "AG": "安提瓜和巴布达", "DM": "多米尼克", "KN": "圣基茨和尼维斯", "BS": "巴哈马", "CU": "古巴", "DO": "多米尼加", "HT": "海地",
	"PR": "波多黎各", "VI": "美属维尔京群岛", "GU": "关岛", "AS": "美属萨摩亚", "MP": "北马里亚纳群岛", "PW": "帕劳", "FM": "密克罗尼西亚", "MH": "马绍尔群岛",
	"KI": "基里巴斯", "TV": "图瓦卢", "NR": "瑙鲁", "WS": "萨摩亚", "TO": "汤加", "FJ": "斐济", "VU": "瓦努阿图", "SB": "所罗门群岛",
	"PG": "巴布亚新几内亚", "NC": "新喀里多尼亚", "PF": "法属波利尼西亚", "WF": "瓦利斯和富图纳", "CK": "库克群岛", "NU": "纽埃", "TK": "托克劳", "SC": "塞舌尔",
	"UNKNOWN": "未知",
}
