package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/ts721521/DevToolbox/internal/desktop"
	"github.com/ts721521/DevToolbox/internal/docs"
	"github.com/ts721521/DevToolbox/internal/launcher"
	"github.com/ts721521/DevToolbox/internal/platform"
	"github.com/ts721521/DevToolbox/internal/registry"
	"github.com/ts721521/DevToolbox/internal/server"
	"github.com/ts721521/DevToolbox/internal/version"
)

//go:embed web/*
var webFS embed.FS

//go:embed AGENTS.md
var agentsMD []byte

//go:embed HOW_TO_REGISTER.md
var howToMD []byte

//go:embed tool.schema.json
var schemaJSON []byte

func guideFiles() map[string][]byte {
	return map[string][]byte{
		"AGENTS.md":          agentsMD,
		"HOW_TO_REGISTER.md": howToMD,
		"tool.schema.json":   schemaJSON,
	}
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 && strings.HasPrefix(args[0], "-psn_") {
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(runServe())
	}
	switch args[0] {
	case "serve", "gui", "start":
		os.Exit(runServe())
	case "list":
		os.Exit(runList())
	case "open", "launch":
		os.Exit(runOpen(args[1:]))
	case "stop", "kill":
		os.Exit(runStop(args[1:]))
	case "register":
		os.Exit(runRegister(args[1:]))
	case "unregister", "rm":
		os.Exit(runUnregister(args[1:]))
	case "install-desktop":
		os.Exit(runInstallDesktop())
	case "install-cli":
		os.Exit(runInstallCLI())
	case "version", "-v", "--version":
		fmt.Printf("devtoolbox %s commit=%s date=%s\n", version.Version, version.Commit, version.Date)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", args[0])
		printHelp()
		os.Exit(2)
	}
}

func printHelp() {
	fmt.Printf("开发工具箱 (devtoolbox) %s — macOS / Windows 本地工具启动器\n\n", version.Version)
	fmt.Print(`用法:
  devtoolbox                     打开图形界面
  devtoolbox list                列出已注册工具
  devtoolbox open <id>           启动某个工具
  devtoolbox stop <id>           关闭工具并杀掉后台进程
  devtoolbox register --file f   从 JSON 注册（其他项目 / 其他 AI 用这个）
  devtoolbox unregister <id>     移除注册
  devtoolbox install-desktop     把快捷方式装到桌面（并写入 AGENTS.md）
  devtoolbox install-cli         安装命令行到 ~/bin
  devtoolbox version             显示版本

其他 AI：请读 AGENTS.md（源码目录、安装目录、http://127.0.0.1:17890/AGENTS.md）
`)
}

func runServe() int {
	_ = docs.Publish(guideFiles())
	if server.AlreadyRunning() {
		_ = platform.OpenInChromeApp(server.URL())
		return 0
	}
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	go func() {
		_ = platform.OpenInChromeApp(server.URL())
	}()
	fmt.Println("开发工具箱", server.URL())
	h := server.Handler(http.FileServer(http.FS(sub)), server.Options{Guides: guideFiles()})
	if err := http.ListenAndServe(server.Addr, h); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runList() int {
	tools, err := registry.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(tools) == 0 {
		fmt.Println("(empty)")
		return 0
	}
	for _, t := range tools {
		st := launcher.Probe(t)
		state := "stop"
		if st.Running {
			state = "run "
		}
		fmt.Printf("%s  %-18s  %-8s  %s\n", state, t.ID, strings.Join(t.Platforms, ","), t.Name)
	}
	return 0
}

func runOpen(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devtoolbox open <id>")
		return 2
	}
	t, err := registry.Get(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := launcher.Launch(t); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runStop(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devtoolbox stop <id>")
		return 2
	}
	t, err := registry.Get(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := launcher.Stop(t); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("stopped", t.ID)
	return 0
}

func runUnregister(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: devtoolbox unregister <id>")
		return 2
	}
	if err := registry.Remove(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runRegister(args []string) int {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	file := fs.String("file", "", "JSON 文件（单个工具对象）")
	id := fs.String("id", "", "唯一 ID")
	name := fs.String("name", "", "显示名称")
	kind := fs.String("kind", "", "web|command|app|url")
	group := fs.String("group", "", "分组")
	workdir := fs.String("workdir", "", "工作目录")
	execLine := fs.String("exec", "", "启动命令（空格分隔）")
	url := fs.String("url", "", "打开的网址")
	health := fs.String("health-contains", "", "健康检查页面需包含的文字")
	appPath := fs.String("app", "", ".app / .exe 路径")
	platforms := fs.String("platforms", "darwin,windows", "darwin,windows")
	terminal := fs.Bool("terminal", true, "在终端运行")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		var t registry.Tool
		if err := json.Unmarshal(data, &t); err != nil {
			fmt.Fprintln(os.Stderr, "invalid json:", err)
			return 1
		}
		if err := registry.Save(t); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("registered", t.ID)
		return 0
	}
	var plats []string
	for _, p := range strings.Split(*platforms, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			plats = append(plats, p)
		}
	}
	t := registry.Tool{
		ID:             *id,
		Name:           *name,
		Kind:           *kind,
		Group:          *group,
		Workdir:        *workdir,
		URL:            *url,
		HealthURL:      *url,
		HealthContains: *health,
		AppPath:        *appPath,
		Terminal:       *terminal,
		Platforms:      plats,
	}
	if strings.TrimSpace(*execLine) != "" {
		t.Command = strings.Fields(*execLine)
	}
	if err := registry.Save(t); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("registered", t.ID)
	return 0
}

func runInstallDesktop() int {
	self, err := desktop.SelfPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = docs.Publish(guideFiles())
	path, err := desktop.Install(self, guideFiles())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(path)
	return 0
}

func runInstallCLI() int {
	self, err := desktop.SelfPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	path, err := desktop.InstallCLI(self)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(path)
	return 0
}
