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
				origName := ""
				if r.ProxyConfig != nil {
					if n, ok := r.ProxyConfig["name"].(string); ok {
						origName = n
					}
				}
				if origName == "" && r.Proxy != nil {
					origName = r.Proxy.Name()
				}

				location, err := speedTester.GetIPLocation(r.Proxy)
				countryCode, countryName, countryFlag := speedtester.ResolveCountry(location, err, origName)

				proxyConfig := r.ProxyConfig
				if *fastMode {
					newUUID := uuid.New().String()
					proxyConfig["name"] = fmt.Sprintf("%s|%s|%s|%dms|%s",
						countryName,
						countryCode,
						countryFlag,
						r.Latency.Milliseconds(),
						newUUID)
				} else {
					proxyConfig["name"] = generateNodeName(countryCode, countryFlag, r.DownloadSpeed)
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

func generateNodeName(countryCode, flag string, downloadSpeed float64) string {
	if flag == "" {
		flag = speedtester.GetCountryFlag(countryCode)
	}

	speedMBps := downloadSpeed / (1024 * 1024)
	return fmt.Sprintf("%s %s | ⬇️ %.2f MB/s", flag, strings.ToUpper(countryCode), speedMBps)
}
