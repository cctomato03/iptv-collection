package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type ConfigInfo struct {
	UrlList      []string       `json:"url"`
	CategoryList []CategoryInfo `json:"categoryList"`
}

type CategoryInfo struct {
	CategoryName string   `json:"categoryName"`
	ChannelList  []string `json:"channelList"`
}

type ChannelInfo struct {
	ChannelName string
	ChannelList []string
}

type DataInfo struct {
	CategoryName string
	ChannelList  []*ChannelInfo
}

func isIpv6(url string) bool {
	match, err := regexp.MatchString(`^http://[[0-9a-fA-F:]+]`, url)
	if err != nil {
		return false
	}
	return match
}

func fetchUrl(url string) (map[string][]string, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	result := string(b)

	channels := make(map[string][]string)

	lines := strings.Split(result, "\n")
	isM3u := strings.Contains(result, "#EXTM3U")

	var channelName string

	if isM3u {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.Contains(line, "http") {
				continue
			}
			if strings.HasPrefix(line, "#EXTINF") {
				re := regexp.MustCompile(`group-title="(.*?)",(.*)`)
				matches := re.FindAllStringSubmatch(line, -1)
				if len(matches) == 1 && len(matches[0]) == 3 {
					//currentCategory = strings.TrimSpace(matches[0][1])
					channelName = strings.TrimSpace(matches[0][2])
					if _, ok := channels[channelName]; !ok {
						channels[channelName] = make([]string, 0)
					}
				}
			} else if !strings.HasPrefix(line, "#") {
				channelUrl := strings.TrimSpace(line)

				if len(channelName) > 0 {
					channels[channelName] = append(channels[channelName], channelUrl)
				}
			}
		}
	} else {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.Contains(line, "#genre#") && strings.Contains(line, "http") {
				channelString := strings.Split(line, ",")
				if len(channelString) == 2 {
					channelName = strings.TrimSpace(channelString[0])
					channelUrl := strings.TrimSpace(channelString[1])
					if len(channelName) > 0 {
						if _, ok := channels[channelName]; !ok {
							channels[channelName] = make([]string, 0)
							channels[channelName] = append(channels[channelName], channelUrl)
						} else {
							channels[channelName] = append(channels[channelName], channelUrl)
						}
					}
				}
			}
		}
	}
	return channels, nil
}

func checkUrl(url string) bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", fmt.Sprintf(".\\ffprobe.exe -v error -show_format -show_streams %s", url))
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("bash", "-c", fmt.Sprintf("./ffprobe -v error -show_format -show_streams %s", url))
	} else {
		return false
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return false
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	after := time.After(5 * time.Second)
	select {
	case <-after:
		_ = cmd.Process.Kill()
		return false
	case err := <-done:
		if err != nil {
			return false
		}
	}

	return len(stdout.String()) > 0
}

var sourceType = "ipv6"
var configFile = "config.json"
var isCheckUrl = "yes"

func main() {
	// 只筛选特定网络链接
	flag.StringVar(&sourceType, "type", "ipv6", "filter url, only ipv4、ipv6、all")
	// 指定config配置文件
	flag.StringVar(&configFile, "config", "config.json", "config file path")
	// 是否检查url有效性
	flag.StringVar(&isCheckUrl, "check", "yes", "check url valid, only yes or no")
	flag.Parse()

	sourceFile, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("open config file error")
		return
	}

	var configInfo ConfigInfo

	if err := json.Unmarshal(sourceFile, &configInfo); err != nil {
		fmt.Println("parse config file error")
		return
	} else {
		if len(configInfo.UrlList) == 0 {
			fmt.Println("url is empty")
			return
		}

		if len(configInfo.CategoryList) == 0 {
			fmt.Println("category is empty")
			return
		}

		allChannelList := make(map[string][]string)
		for _, url := range configInfo.UrlList {
			fetchedData, err := fetchUrl(url)
			if err == nil {
				for categoryName, channelList := range fetchedData {
					if _, ok := allChannelList[categoryName]; ok {
						allChannelList[categoryName] = channelList
					} else {
						allChannelList[categoryName] = append(allChannelList[categoryName], channelList...)
					}
				}
			}
		}

		var dataInfoList []*DataInfo
		for _, category := range configInfo.CategoryList {
			var channelInfoList = make([]*ChannelInfo, 0)
			for _, channelName := range category.ChannelList {
				if _, ok := allChannelList[channelName]; ok {
					channelInfo := &ChannelInfo{
						ChannelName: channelName,
						ChannelList: allChannelList[channelName],
					}
					channelInfoList = append(channelInfoList, channelInfo)
				}
			}
			var dataInfo = &DataInfo{
				CategoryName: category.CategoryName,
				ChannelList:  channelInfoList,
			}

			dataInfoList = append(dataInfoList, dataInfo)
		}

		liveV4 := ""
		liveV6 := ""
		for _, dataInfo := range dataInfoList {
			fmt.Println(dataInfo.CategoryName)
			liveV4 = fmt.Sprintf("%s%s, #genre#\n", liveV4, dataInfo.CategoryName)
			liveV6 = fmt.Sprintf("%s%s, #genre#\n", liveV6, dataInfo.CategoryName)

			for _, channelInfo := range dataInfo.ChannelList {
				fmt.Println(channelInfo.ChannelName)
				for _, channelUrl := range channelInfo.ChannelList {
					if isIpv6(channelUrl) {
						if sourceType == "ipv6" || sourceType == "all" {
							if isCheckUrl == "yes" && !checkUrl(channelUrl) {
								continue
							}
							fmt.Println(channelUrl)
							channelString := fmt.Sprintf("%s,%s\n", channelInfo.ChannelName, channelUrl)
							if !strings.Contains(liveV6, channelString) {
								liveV6 = fmt.Sprintf("%s%s", liveV6, channelString)
							}
						}
					} else {
						if sourceType == "ipv4" || sourceType == "all" {
							if isCheckUrl == "yes" && !checkUrl(channelUrl) {
								continue
							}
							fmt.Println(channelUrl)
							channelString := fmt.Sprintf("%s,%s\n", channelInfo.ChannelName, channelUrl)
							if !strings.Contains(liveV4, channelString) {
								liveV4 = fmt.Sprintf("%s%s", liveV4, channelString)
							}
						}
					}
				}
			}
		}

		if sourceType == "ipv6" || sourceType == "all" {
			_ = os.WriteFile("live_v6.txt", []byte(liveV6), os.ModePerm)
		}

		if sourceType == "ipv4" || sourceType == "all" {
			_ = os.WriteFile("live_v4.txt", []byte(liveV4), os.ModePerm)
		}

		fmt.Println("end")
	}
}
