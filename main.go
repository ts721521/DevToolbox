package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/ts721521/DevToolbox/internal/applog"
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
	if err := applog.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "log init: %v\n", err)
	}
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
	case "dir", "reveal", "folder":
		os.Exit(runRevealDir(args[1:]))
	case "app":
		os.Exit(runRevealApp(args[1:]))
	case "git", "repo":
		os.Exit(runRevealGit(args[1:]))
	case "stop", "kill":
		os.Exit(runStop(args[1:]))
	case "register":
		os.Exit(runRegister(args[1:]))
	case "unregister", "rm":
		os.Exit(runUnregister(args[1:]))
	case "restore":
		os.Exit(runRestore(args[1:]))
	case "blocked", "trash":
		os.Exit(runBlocked())
	case "unblock":
		os.Exit(runUnblock(args[1:]))
	case "install-desktop":
		os.Exit(runInstallDesktop())
	case "install-cli":
		os.Exit(runInstallCLI())
	case "pack":
		os.Exit(runPack(args[1:]))
	case "logs":
		os.Exit(runLogs())
	case "version", "-v", "--version":
		fmt.Printf("工坞 ToolDock %s\ncommit %s\ndate %s\n", version.Display(), version.Commit, version.Date)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", args[0])
		printHelp()
		os.Exit(2)
	}
}

func printHelp() {
	fmt.Printf("工坞 ToolDock %s — macOS / Windows 本地工具启动器\n\n", version.Display())
	fmt.Print(`用法:
  tooldock                       打开界面
  tooldock list                  列出已注册工具
  tooldock open <id>             启动某个工具（含 depends / services）
  tooldock dir <id>              打开开发目录（访达 / 资源管理器）
  tooldock app <id>              打开原始程序（.app / .exe）
  tooldock git <id>              打开 Git 仓库页
  tooldock stop <id>             关闭工具并杀掉后台进程
  tooldock register --file f     从 JSON 注册
  tooldock unregister <id>       移到垃圾桶（以后不会再被注册）
  tooldock blocked               列出垃圾桶
  tooldock restore <id>          从垃圾桶恢复
  tooldock unblock <id>          取消屏蔽（允许以后再注册）
  tooldock install-desktop       安装到「应用程序」并在桌面创建快捷方式
  tooldock install-cli           安装命令行到 ~/bin
  tooldock pack [--out dist]     封装带版本号的安装包
  tooldock version               显示版本
  tooldock logs                  显示日志路径并打印最近记录

兼容旧命令名 devtoolbox。其他 AI 请读 AGENTS.md。
`)
}

func runServe() int {
	_ = docs.Publish(guideFiles())
	self, _ := desktop.SelfPath()
	handoff, err := server.ClaimHub(self, version.Display(), version.Commit)
	if err != nil {
		applog.Error("claim_hub", "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if handoff {
		applog.Info("already_running", "url", server.URL(), "version", version.Display())
		if !platform.HubLaunchedByCocoa(self) {
			_ = platform.OpenInChromeApp(server.URL())
		}
		return 0
	}
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		applog.Error("embed_web", "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	logFile, _ := applog.Path()
	applog.Info("serve", "url", server.URL(), "version", version.Display(), "log", logFile)
	fmt.Println("工坞 ToolDock", server.URL())
	if logFile != "" {
		fmt.Println("日志", logFile)
	}
	h := server.Handler(http.FileServer(http.FS(sub)), server.Options{Guides: guideFiles()})
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		live := server.PeekIdentity()
		if live.Dock {
			applog.Info("listen_busy_handoff", "url", server.URL())
			if !platform.HubLaunchedByCocoa(self) {
				_ = platform.OpenInChromeApp(server.URL())
			}
			return 0
		}
		name := live.Name
		if name == "" {
			name = "unknown"
		}
		msg := fmt.Sprintf("%s 已被其他程序占用（%s）", server.Addr, name)
		applog.Error("listen", "err", msg)
		fmt.Fprintln(os.Stderr, msg)
		return 1
	}
	if err := server.WriteLock(self, version.Display(), version.Commit); err != nil {
		_ = ln.Close()
		applog.Error("hub_lock", "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !platform.HubLaunchedByCocoa(self) {
		go func() {
			_ = platform.OpenInChromeApp(server.URL())
		}()
	}
	if err := http.Serve(ln, h); err != nil {
		applog.Error("listen", "err", err)
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
		st := launcher.Alive(t)
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
		fmt.Fprintln(os.Stderr, "usage: tooldock open <id>")
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

func runRevealDir(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: tooldock dir <id>")
		return 2
	}
	t, err := registry.Get(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := launcher.RevealDir(t); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runRevealApp(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: tooldock app <id>")
		return 2
	}
	t, err := registry.Get(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := launcher.RevealProgram(t); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runRevealGit(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: tooldock git <id>")
		return 2
	}
	t, err := registry.Get(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := launcher.RevealGit(t); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runStop(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: tooldock stop <id>")
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
		fmt.Fprintln(os.Stderr, "usage: tooldock unregister <id>")
		return 2
	}
	if err := registry.Remove(args[0]); err != nil {
		applog.Warn("cli_unregister", "id", args[0], "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	applog.Info("cli_unregister", "id", args[0])
	fmt.Println("trashed", args[0])
	return 0
}

func runBlocked() int {
	ents, err := registry.ListBlocked()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(ents) == 0 {
		fmt.Println("(empty)")
		return 0
	}
	for _, e := range ents {
		fmt.Printf("%s\t%s\n", e.ID, e.Name)
	}
	return 0
}

func runRestore(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: tooldock restore <id>")
		return 2
	}
	if err := registry.Restore(args[0]); err != nil {
		applog.Warn("cli_restore", "id", args[0], "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	applog.Info("cli_restore", "id", args[0])
	fmt.Println("restored", args[0])
	return 0
}

func runUnblock(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: tooldock unblock <id>")
		return 2
	}
	if err := registry.Unblock(args[0]); err != nil {
		applog.Warn("cli_unblock", "id", args[0], "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	applog.Info("cli_unblock", "id", args[0])
	fmt.Println("unblocked", args[0])
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
			applog.Error("cli_register", "file", *file, "err", err)
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		applog.Info("cli_register", "id", t.ID, "file", *file)
		fmt.Println("registered", t.ID)
		if h := registry.ExtrasHint(t); h != "" {
			fmt.Fprintln(os.Stderr, "提示:", h)
		}
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
		applog.Error("cli_register", "id", t.ID, "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	applog.Info("cli_register", "id", t.ID, "name", t.Name)
	fmt.Println("registered", t.ID)
	if h := registry.ExtrasHint(t); h != "" {
		fmt.Fprintln(os.Stderr, "提示:", h)
	}
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
		applog.Error("install_desktop", "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	applog.Info("install_desktop", "path", path)
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
		applog.Error("install_cli", "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	applog.Info("install_cli", "path", path)
	fmt.Println(path)
	return 0
}

func runPack(args []string) int {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	out := fs.String("out", "dist", "输出目录")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	self, err := desktop.SelfPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	files, err := desktop.Pack(*out, self)
	if err != nil {
		applog.Error("pack", "err", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("工坞 ToolDock %s\n", version.Display())
	for _, f := range files {
		fmt.Println(f)
	}
	return 0
}

func runLogs() int {
	p, err := applog.Path()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(p)
	text, err := applog.Tail(300)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(text)
	return 0
}
