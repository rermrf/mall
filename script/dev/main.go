package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// --------------- service registry ---------------

type service struct {
	Name string
	Port int
}

var allServices = []service{
	{"user", 8081}, {"tenant", 8082}, {"product", 8083}, {"inventory", 8084},
	{"order", 8085}, {"payment", 8086}, {"cart", 8087}, {"search", 8088},
	{"marketing", 8089}, {"logistics", 8090}, {"notification", 8091},
	{"consumer-bff", 8080}, {"merchant-bff", 8180}, {"admin-bff", 8280},
}

var serviceMap = func() map[string]service {
	m := make(map[string]service, len(allServices))
	for _, s := range allServices {
		m[s.Name] = s
	}
	return m
}()

// --------------- paths ---------------

func rootDir() string {
	exe, _ := os.Executable()
	// When run via "go run", exe is in a temp dir; use working directory instead
	wd, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd
	}
	return filepath.Dir(filepath.Dir(exe))
}

func devDir() string  { return filepath.Join(rootDir(), ".dev") }
func pidDir() string  { return filepath.Join(devDir(), "pids") }
func logDir() string  { return filepath.Join(devDir(), "logs") }
func pidFile(name string) string { return filepath.Join(pidDir(), name+".pid") }
func logFile(name string) string { return filepath.Join(logDir(), name+".log") }

func ensureDirs() {
	os.MkdirAll(pidDir(), 0o755)
	os.MkdirAll(logDir(), 0o755)
}

// --------------- pid helpers ---------------

func readPID(name string) (int, bool) {
	data, err := os.ReadFile(pidFile(name))
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	return pid, true
}

func writePID(name string, pid int) {
	os.WriteFile(pidFile(name), []byte(strconv.Itoa(pid)), 0o644)
}

func removePID(name string) {
	os.Remove(pidFile(name))
}

func isProcessAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; send signal 0 to check
	if runtime.GOOS != "windows" {
		return p.Signal(syscall.Signal(0)) == nil
	}
	// On Windows, FindProcess fails if process doesn't exist
	// If we got here, process exists. Double-check with signal.
	err = p.Signal(syscall.Signal(0))
	return err == nil
}

func isRunning(name string) bool {
	pid, ok := readPID(name)
	if !ok {
		return false
	}
	return isProcessAlive(pid)
}

// --------------- color output ---------------

const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[0;33m"
	colorCyan   = "\033[0;36m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

func useColor() bool {
	// Disable color on Windows cmd.exe (no ANSI support by default)
	if runtime.GOOS == "windows" {
		if os.Getenv("TERM") == "" && os.Getenv("WT_SESSION") == "" {
			return false
		}
	}
	return true
}

func c(code, text string) string {
	if !useColor() {
		return text
	}
	return code + text + colorReset
}

func logInfo(msg string)  { fmt.Printf("%s  %s\n", c(colorCyan, "[INFO]"), msg) }
func logOK(msg string)    { fmt.Printf("%s    %s\n", c(colorGreen, "[OK]"), msg) }
func logWarn(msg string)  { fmt.Printf("%s  %s\n", c(colorYellow, "[WARN]"), msg) }
func logErr(msg string)   { fmt.Printf("%s   %s\n", c(colorRed, "[ERR]"), msg) }

// --------------- global config ---------------

// configFile holds the --config flag value (empty = use service default).
var configFile string

// --------------- commands ---------------

func cmdStart(names []string) {
	if len(names) == 0 {
		names = serviceNames()
	}
	if err := validateNames(names); err != nil {
		logErr(err.Error())
		os.Exit(1)
	}

	logInfo(fmt.Sprintf("启动 %d 个服务...", len(names)))
	if configFile != "" {
		logInfo("配置文件: " + configFile)
	}
	fmt.Println()

	root := rootDir()
	for _, name := range names {
		if isRunning(name) {
			pid, _ := readPID(name)
			logWarn(fmt.Sprintf("%s 已在运行 (PID %d)", name, pid))
			continue
		}

		svcDir := filepath.Join(root, name)
		if _, err := os.Stat(filepath.Join(svcDir, "main.go")); err != nil {
			logErr(fmt.Sprintf("%s: main.go 不存在", name))
			continue
		}

		lf, err := os.Create(logFile(name))
		if err != nil {
			logErr(fmt.Sprintf("%s: 创建日志文件失败: %v", name, err))
			continue
		}

		args := []string{"run", "."}
		if configFile != "" {
			args = append(args, "--config", configFile)
		}
		cmd := exec.Command("go", args...)
		cmd.Dir = filepath.Join(root, name)
		cmd.Stdout = lf
		cmd.Stderr = lf

		if err := cmd.Start(); err != nil {
			lf.Close()
			logErr(fmt.Sprintf("%s: 启动失败: %v", name, err))
			continue
		}
		lf.Close()

		writePID(name, cmd.Process.Pid)

		svc := serviceMap[name]
		logOK(fmt.Sprintf("%s 已启动 (PID %d, port %d)", name, cmd.Process.Pid, svc.Port))

		// Detach: don't wait for process
		go cmd.Wait()
	}

	fmt.Println()
	logInfo("日志目录: " + logDir())
	logInfo("停止服务: go run ./script/dev stop")
	logInfo("查看状态: go run ./script/dev status")
}

func cmdStop(names []string) {
	if len(names) == 0 {
		names = serviceNames()
	}

	logInfo("停止服务...")
	for _, name := range names {
		pid, ok := readPID(name)
		if !ok {
			continue
		}

		p, err := os.FindProcess(pid)
		if err != nil {
			removePID(name)
			continue
		}

		// Try graceful shutdown
		if runtime.GOOS == "windows" {
			p.Kill()
		} else {
			p.Signal(syscall.SIGTERM)
		}

		// Wait up to 3 seconds
		done := make(chan struct{})
		go func() {
			for range 30 {
				if !isProcessAlive(pid) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			close(done)
		}()
		<-done

		// Force kill if still alive
		if isProcessAlive(pid) {
			p.Kill()
			time.Sleep(200 * time.Millisecond)
		}

		removePID(name)
		logOK(fmt.Sprintf("%s 已停止 (PID %d)", name, pid))
	}
	logOK("完成")
}

func cmdStatus() {
	fmt.Println()
	header := fmt.Sprintf("%-20s %-8s %-8s %-8s", "SERVICE", "PORT", "PID", "STATUS")
	if useColor() {
		header = colorBold + header + colorReset
	}
	fmt.Println(header)
	fmt.Printf("%-20s %-8s %-8s %-8s\n", "-------", "----", "---", "------")

	running, stopped := 0, 0

	for _, svc := range allServices {
		pidStr := "-"
		status := c(colorYellow, "stopped")

		if pid, ok := readPID(svc.Name); ok {
			if isProcessAlive(pid) {
				pidStr = strconv.Itoa(pid)
				status = c(colorGreen, "running")
				running++
			} else {
				status = c(colorRed, "dead")
				removePID(svc.Name)
				stopped++
			}
		} else {
			stopped++
		}

		fmt.Printf("%-20s %-8d %-8s %s\n", svc.Name, svc.Port, pidStr, status)
	}

	fmt.Println()
	fmt.Printf("  运行: %s  停止: %s  共: %d\n",
		c(colorGreen, strconv.Itoa(running)),
		c(colorYellow, strconv.Itoa(stopped)),
		len(allServices))
	fmt.Println()
}

func cmdLogs(names []string) {
	if len(names) == 0 {
		names = serviceNames()
	}

	// Collect existing log files
	var files []string
	for _, name := range names {
		lf := logFile(name)
		if _, err := os.Stat(lf); err == nil {
			files = append(files, lf)
		}
	}

	if len(files) == 0 {
		logWarn("没有找到日志文件")
		return
	}

	logInfo("Ctrl+C 退出日志查看")

	// Use tail -f (works on macOS/Linux; on Windows use PowerShell Get-Content)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// PowerShell: Get-Content -Wait -Path file1,file2
		args := []string{"-NoProfile", "-Command",
			fmt.Sprintf("Get-Content -Wait -Path %s", strings.Join(files, ","))}
		cmd = exec.Command("powershell", args...)
	} else {
		args := append([]string{"-f"}, files...)
		cmd = exec.Command("tail", args...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward Ctrl+C to child
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	cmd.Run()
}

func cmdRestart(names []string) {
	cmdStop(names)
	time.Sleep(500 * time.Millisecond)
	cmdStart(names)
}

// --------------- utils ---------------

func serviceNames() []string {
	names := make([]string, len(allServices))
	for i, s := range allServices {
		names[i] = s.Name
	}
	return names
}

func validateNames(names []string) error {
	for _, n := range names {
		if _, ok := serviceMap[n]; !ok {
			return fmt.Errorf("未知服务: %s\n  可用: %s", n, strings.Join(serviceNames(), " "))
		}
	}
	return nil
}

// extractConfig scans args for --config <path> or --config=<path>,
// sets configFile, and returns remaining positional args.
func extractConfig(args []string) []string {
	var rest []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--config" && i+1 < len(args):
			configFile = args[i+1]
			i++ // skip next
		case strings.HasPrefix(args[i], "--config="):
			configFile = strings.TrimPrefix(args[i], "--config=")
		default:
			rest = append(rest, args[i])
		}
	}
	return rest
}

func printHelp() {
	fmt.Println(`用法: go run ./script/dev <command> [service...] [--config <path>]

Commands:
  start [service]   启动服务 (不指定=全部)
  stop  [service]   停止服务 (不指定=全部)
  restart [service] 重启服务
  status            查看运行状态
  logs [service]    tail 日志

Options:
  --config <path>   指定配置文件 (默认: config/dev.yaml)

Services:
  ` + strings.Join(serviceNames(), " ") + `

Examples:
  go run ./script/dev start              # 启动全部 (默认配置)
  go run ./script/dev start order        # 只启动 order
  go run ./script/dev start --config config/example.yaml  # 使用本地配置
  go run ./script/dev stop               # 停止全部
  go run ./script/dev logs order         # 查看 order 日志

Makefile shortcuts:
  make dev-run-all                        # 启动全部
  make dev-run-order                      # 启动 order
  make dev-run-all CONF=config/example.yaml  # 使用本地配置
  make dev-stop-all                       # 停止全部
  make dev-status                         # 查看状态`)
}

// --------------- main ---------------

func main() {
	ensureDirs()

	if len(os.Args) < 2 {
		printHelp()
		return
	}

	cmd := os.Args[1]
	args := extractConfig(os.Args[2:])

	switch cmd {
	case "start":
		cmdStart(args)
	case "stop":
		cmdStop(args)
	case "restart":
		cmdRestart(args)
	case "status":
		cmdStatus()
	case "logs":
		cmdLogs(args)
	case "help", "--help", "-h":
		printHelp()
	default:
		logErr("未知命令: " + cmd)
		fmt.Println("  用法: go run ./script/dev {start|stop|restart|status|logs} [service...]")
		os.Exit(1)
	}
}
