package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/desktop"
	"github.com/ts721521/DevToolbox/internal/registry"
	"github.com/ts721521/DevToolbox/internal/version"
)

const ForAIPath = "/for-ai.md"

func ForAIURL() string {
	return URL() + ForAIPath
}

func AIPrompt() string {
	return strings.TrimSpace(fmt.Sprintf(`请立刻读取这个本机地址（工坞 ToolDock 的注册说明书），然后把「你当前正在开发的这个项目」注册进去：

%s

读完后必须：
1. GET %s/api/tabs ，把当前项目放进最接近的已有标签（不要自己发明一堆新分类）
2. 在项目根写入 .devtoolbox.json（必须含 group；web/command/app 必须 workdir——界面「目录」/ tooldock dir；有安装好的 .app/.exe 再加 app_path——界面「程序」/ tooldock app）
3. 执行 tooldock register --file .devtoolbox.json
   （没有 tooldock 时用 curl -X POST http://127.0.0.1:17890/api/tools -d @.devtoolbox.json）
4. 不要再往桌面生成单独快捷方式`, ForAIURL(), URL()))
}

func handleAIHandoff(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"url":    ForAIURL(),
		"prompt": AIPrompt(),
	})
}

func handleForAI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(forAIMarkdown()))
}

func cliCandidates() []string {
	home, _ := os.UserHomeDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	out := []string{
		filepath.Join(home, "bin", "tooldock"+ext),
		filepath.Join(home, "bin", "devtoolbox"+ext),
	}
	if runtime.GOOS == "darwin" {
		out = append(out, "/Applications/工坞.app/Contents/MacOS/tooldock")
	}
	if self, err := desktop.SelfPath(); err == nil && self != "" {
		out = append([]string{self}, out...)
	}
	seen := map[string]struct{}{}
	var uniq []string
	for _, p := range out {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		uniq = append(uniq, p)
	}
	return uniq
}

func firstExisting(paths []string) string {
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return "tooldock"
}

func forAIMarkdown() string {
	root, _ := registry.RootDir()
	toolsDir, _ := registry.Dir()
	cli := firstExisting(cliCandidates())
	osName := registry.PlatformLabel(runtime.GOOS)
	var b strings.Builder
	fmt.Fprintf(&b, "# 工坞 / ToolDock — 给 AI 的注册说明书\n\n")
	fmt.Fprintf(&b, "你正在阅读**本机**工坞（ToolDock）的注册协议。\n")
	fmt.Fprintf(&b, "用户把这个链接发给你，是要你把**当前对话所在的项目**注册进 ToolDock，方便以后一键打开 / 关闭。\n\n")
	fmt.Fprintf(&b, "**不要**再给该项目做桌面 .command / .lnk / .app 快捷方式。\n")
	fmt.Fprintf(&b, "**不要**改 ToolDock 源码，除非用户当前项目就是 ToolDock 本身。\n\n")
	fmt.Fprintf(&b, "## 本机实况\n\n")
	fmt.Fprintf(&b, "- 产品：工坞 / ToolDock %s\n", version.Version)
	fmt.Fprintf(&b, "- 系统：%s / %s\n", osName, runtime.GOARCH)
	fmt.Fprintf(&b, "- 界面：%s\n", URL())
	fmt.Fprintf(&b, "- 注册 API：POST %s/api/tools\n", URL())
	fmt.Fprintf(&b, "- CLI：`%s`\n", cli)
	logFile, _ := applog.Path()
	fmt.Fprintf(&b, "- 工具 JSON 目录：`%s`\n", toolsDir)
	fmt.Fprintf(&b, "- 配置根：`%s`\n", root)
	tabs := registry.LoadTabs()
	fmt.Fprintf(&b, "- 当前标签页：%s\n", strings.Join(tabs, " / "))
	fmt.Fprintf(&b, "- 日志：`%s`（`tooldock logs` 或 GET %s/api/logs）\n\n", logFile, URL())
	fmt.Fprintf(&b, "## 你必须立刻做的 4 步\n\n")
	fmt.Fprintf(&b, "1. 弄清当前项目怎么启动（README、入口脚本、.app 等）。\n")
	fmt.Fprintf(&b, "2. **先 GET %s/api/tabs**，把项目放进最接近的已有标签。不要新建标签，除非完全对不上。\n", URL())
	fmt.Fprintf(&b, "3. 在**当前项目根目录**写入 `.devtoolbox.json`：`group` 用上一步标签；**必须**把 `workdir` 写成当前项目根绝对路径（界面「目录」/ `tooldock dir`）；若已安装 `.app`/`.exe` 再写 `app_path`（界面「程序」/ `tooldock app`）。\n")
	fmt.Fprintf(&b, "4. 注册（任选一种）：\n\n")
	fmt.Fprintf(&b, "    %s register --file .devtoolbox.json\n\n", cli)
	fmt.Fprintf(&b, "    curl -s -X POST %s/api/tools -H 'Content-Type: application/json' --data-binary @.devtoolbox.json\n\n", URL())
	fmt.Fprintf(&b, "    或把 JSON 复制到 `%s/<id>.json`\n\n", toolsDir)
	fmt.Fprintf(&b, "确认：`%s list`。启动：`%s open <id>`。开发目录：`%s dir <id>`。原始程序：`%s app <id>`。关闭：`%s stop <id>`。\n\n", cli, cli, cli, cli, cli)
	fmt.Fprintf(&b, "## JSON 模板（按项目改，不要用假路径）\n\n")
	fmt.Fprintf(&b, "Web 控制台：\n\n")
	fmt.Fprintf(&b, "    {\n")
	fmt.Fprintf(&b, "      \"id\": \"my-tool\",\n")
	fmt.Fprintf(&b, "      \"name\": \"显示名称\",\n")
	fmt.Fprintf(&b, "      \"description\": \"一句话\",\n")
	fmt.Fprintf(&b, "      \"group\": \"开发\",\n")
	fmt.Fprintf(&b, "      \"kind\": \"web\",\n")
	fmt.Fprintf(&b, "      \"platforms\": [\"darwin\", \"windows\"],\n")
	fmt.Fprintf(&b, "      \"workdir\": \"/absolute/path/to/this/project\",\n")
	fmt.Fprintf(&b, "      \"command\": [\"python3\", \"server.py\"],\n")
	fmt.Fprintf(&b, "      \"command_windows\": [\"python\", \"server.py\"],\n")
	fmt.Fprintf(&b, "      \"url\": \"http://localhost:端口\",\n")
	fmt.Fprintf(&b, "      \"health_url\": \"http://localhost:端口/\",\n")
	fmt.Fprintf(&b, "      \"health_contains\": \"页面里不会撞车的一段文字\",\n")
	fmt.Fprintf(&b, "      \"process_match\": \"server.py\",\n")
	fmt.Fprintf(&b, "      \"terminal\": true\n")
	fmt.Fprintf(&b, "    }\n\n")
	fmt.Fprintf(&b, "- `id`：`[a-zA-Z0-9][a-zA-Z0-9._-]*`\n")
	fmt.Fprintf(&b, "- `kind`：`web` | `command` | `app` | `url`\n")
	fmt.Fprintf(&b, "- `platforms`：`darwin`（macOS）和/或 `windows`\n")
	fmt.Fprintf(&b, "- **必须写 `workdir`**（`kind=url` 纯书签除外）：当前项目根的绝对路径。界面「目录」和 `tooldock dir` 靠它打开开发目录。\n")
	fmt.Fprintf(&b, "- 已安装的桌面程序再写 `app_path`（`.app` / `.exe`）。界面「程序」和 `tooldock app` 靠它打开原始程序。\n")
	fmt.Fprintf(&b, "- `kind=app` 时也要同时给 `workdir`（源码）和 `app_path`（安装好的程序），不要只写其中一个。\n")
	fmt.Fprintf(&b, "- 命令行工具用 `command`；纯网址用 `url`\n\n")
	fmt.Fprintf(&b, "## 分类（标签页 / group）— 必须自己选，不要堆新类\n\n")
	fmt.Fprintf(&b, "当前标签：**%s**\n\n", strings.Join(tabs, " / "))
	fmt.Fprintf(&b, "默认四类（能套上就套，不要新增第五类）：\n\n")
	fmt.Fprintf(&b, "- **工作**：销售、工程、许可证、情报、内部业务系统\n")
	fmt.Fprintf(&b, "- **财务**：报销、发票、报价\n")
	fmt.Fprintf(&b, "- **开发**：账号、仪表盘、本机开发辅助、AI 控制台\n")
	fmt.Fprintf(&b, "- **其他**：对不上上面三类才用\n\n")
	fmt.Fprintf(&b, "`group` 必须写成上述标签的**原文字**。同一项目只注册一次；重复副本选 SSD / 主仓库，不要把两份拷贝都注册进来。\n\n")
	fmt.Fprintf(&b, "## HTTP\n\n")
	fmt.Fprintf(&b, "- GET %s/api/system\n", URL())
	fmt.Fprintf(&b, "- GET %s/api/tabs\n", URL())
	fmt.Fprintf(&b, "- PUT %s/api/tabs\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tabs\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tabs/rename\n", URL())
	fmt.Fprintf(&b, "- GET %s/api/tools\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tools\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tools/move  （body: {\"ids\":[\"id\"],\"group\":\"开发\"}）\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tools/{id}/launch\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tools/{id}/dir     （打开开发目录 workdir）\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tools/{id}/app     （打开原始程序 app_path）\n", URL())
	fmt.Fprintf(&b, "- POST %s/api/tools/{id}/stop\n", URL())
	fmt.Fprintf(&b, "- GET %s/api/logs\n\n", URL())
	fmt.Fprintf(&b, "## English\n\n")
	fmt.Fprintf(&b, "Register the user's **current project** into ToolDock. First GET %s/api/tabs and set `group` to an existing tab. Always set `workdir` to the project root (Finder/Explorer 「目录」 / `tooldock dir`) except pure `kind=url` bookmarks. If there is an installed .app/.exe, also set `app_path` (「程序」 / `tooldock app`). Write `.devtoolbox.json` at the project root, then run `tooldock register --file .devtoolbox.json` or POST the JSON to %s/api/tools. Do not create extra desktop shortcuts or extra tab names.\n", URL(), URL())
	return b.String()
}
