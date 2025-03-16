package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	"github.com/shirou/gopsutil/cpu"
)

// 定义图标文件路径列表
var iconPaths []string

// 当前 CPU 使用率
var cpuUsage float64

// 加载图标文件
func loadIcon(path string) []byte {
	// fmt.Println("[+] 开始加载图标文件: " + path)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("加载 ico 文件错误，请检查资源列表: %v", err)
	}
	// fmt.Println("[+] 图标加载成功")
	return data
}

// 获取 CPU 使用率
func getCPUUsage() float64 {
	percent, err := cpu.Percent(5*time.Second, false)
	if err != nil {
		log.Fatalf("没救了，获取不到 cpu 信息: %v", err)
	}
	return percent[0]
}

// 读取 resources 文件夹下的所有 .ico 文件
func readIconFiles() {
	fmt.Println("[+] 开始读取图标目录")

	entries, err := os.ReadDir("resources")
	if err != nil {
		log.Fatalf("文件没权限，寄: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ico" {
			iconPaths = append(iconPaths, filepath.Join("resources", entry.Name()))
			fmt.Println("[+] 找到文件: " + entry.Name())

		}
	}
}

// 计算动画间隔时间
func calculateAnimationInterval() time.Duration {
	// fmt.Printf("CPU 使用率: %s\n", cpuUsage)
	if cpuUsage < 20 {
		return 200 * time.Millisecond
	} else if cpuUsage < 40 {
		// 20% - 40% 之间线性过渡可以自行添加逻辑，这里简单按低区间处理
		return 200 * time.Millisecond
	} else if cpuUsage < 60 {
		return 100 * time.Millisecond
	} else if cpuUsage < 80 {
		return 50 * time.Millisecond
	} else if cpuUsage < 100 {
		return 20 * time.Millisecond
	}
	return 10 * time.Millisecond

}

// 系统托盘初始化
func onReady() {

	fmt.Println("[+] 初始化系统托盘")

	if len(iconPaths) == 0 {
		log.Fatal("找不到文件呐~")
	}

	// 初始化图标索引
	currentIconIndex := 0

	// 加载第一个图标
	iconData := loadIcon(iconPaths[currentIconIndex])
	systray.SetIcon(iconData)
	systray.SetTitle("RunCat")
	systray.SetTooltip("请等待第一次扫描")

	// 创建退出菜单
	mOpenTaskMgr := systray.AddMenuItem("任务管理器", "打开任务管理器")
	mQuit := systray.AddMenuItem("退下吧", "退出应用程序")

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
	go func() {
		<-mOpenTaskMgr.ClickedCh
		// 打开任务管理器
		if err := openTaskManager(); err != nil {
			log.Printf("Failed to open task manager: %v", err)
		}
	}()

	// CPU 使用率更新定时器
	cpuTicker := time.NewTicker(1 * time.Second)
	// 动画定时器
	animationTicker := time.NewTicker(calculateAnimationInterval())

	go func() {
		defer cpuTicker.Stop()
		defer animationTicker.Stop()
		for range cpuTicker.C {
			// 获取 CPU 使用率
			cpuUsage = getCPUUsage()

			// cpu大于99.5直接显示100
			if cpuUsage >= 99.5 {
				cpuUsage = 100
			}
			systray.SetTooltip(fmt.Sprintf("CPU 使用率: %.1f%%", cpuUsage))
			// 根据新的 CPU 使用率调整动画间隔
			animationTicker.Reset(calculateAnimationInterval())
		}
	}()

	for range animationTicker.C {
		// 切换图标
		currentIconIndex = (currentIconIndex + 1) % len(iconPaths)
		iconData = loadIcon(iconPaths[currentIconIndex])
		systray.SetIcon(iconData)
	}

}
func openTaskManager() error {
	var shell string
	var args []string

	// 根据不同的操作系统设置相应的命令和参数
	switch runtime.GOOS {
	case "windows":
		shell = "cmd"
		args = []string{"/c", "start", "taskmgr.exe"}
	case "darwin":
		shell = "open"
		args = []string{"/System/Library/CoreServices/TaskManager.app"}
	default:
		return fmt.Errorf("[-] 不支持的操作系统: %s", runtime.GOOS)
	}

	// 执行命令来打开任务管理器
	cmd := exec.Command(shell, args...)
	return cmd.Start()
}

func main() {
	// 读取图标文件
	readIconFiles()

	// 处理系统信号，确保程序可以正常退出
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		systray.Quit()
	}()

	// 启动系统托盘
	systray.Run(onReady, func() {
		// 程序退出时的清理操作
	})
}
