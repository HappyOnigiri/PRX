# PRX

[English](README.md) | [日本語](README.ja.md) | 简体中文

**PRX** 让分散在大量 GitHub 拉取请求中的项目进展，在你自己的机器上一目了然。
只要登记好任务之间的依赖关系，PRX 就会自动区分现在可以着手的任务，以及正在等待其他任务的任务。

## 特性

- **知道可以着手哪些任务** — 只取出依赖项全部完成的任务，无需每次重新推敲执行顺序。
- **知道卡在哪里** — 以图的形式俯瞰整个项目，追溯每个任务在等待什么，也能列出等待评审的拉取请求和停滞的任务。
- **反映拉取请求的状态** — 从 GitHub 获取评审中、冲突、已合并等状态，并反映到任务的进展中。
- **生成交给智能体的指令** — 从模板组装出包含任务内容和依赖关系的提示词，可直接复制使用。
- **完全在本地完成** — 数据保存在你自己的机器上，服务器默认只接受本地连接。

## 安装

### macOS

面向 Apple Silicon 提供预编译的二进制文件。

```sh
curl -fsSL https://github.com/HappyOnigiri/PRX/releases/latest/download/install.sh | bash
```

更新也使用同一条命令。

### Linux / WSL2

从源码构建，macOS 上也可以使用相同的步骤。不支持原生 Windows。

```sh
git clone https://github.com/HappyOnigiri/PRX.git
cd PRX
make install
```

`make install` 会连同 WebUI 一起构建，并安装到 `~/.local/bin/prx`。要安装到其他位置，请指定 `INSTALL_DIR`。
管理常驻服务的 `prx daemon` 和 `prx open` 仅在 macOS 上可用，因此请使用 `prx serve` 启动服务器。

## 快速开始

在浏览器中打开 http://localhost:7331/ 。如果该端口已被占用而服务器换到了其他端口，`prx open` 会打开实际正在监听的地址。

`prx` 命令可以读写同样的数据，因此可以让 AI 智能体查看当前状况，或登记任务和依赖关系。

```sh
prx ready      # 取出可以着手的任务
prx graph F-1  # 查看整个项目的任务与依赖关系
prx prompt T-1 # 组装交给任务的指令
```

命令和选项的详细用法，请参阅 `prx -h` 和 `prx <command> -h`。

## 更多功能

- **与 GitHub 同步：** 通过 `prx config`、`GITHUB_TOKEN`、`GH_TOKEN` 或已登录的 `gh` CLI 提供凭据。即使不同步，任务和依赖关系的管理也照常可用。
- **固定端口：** `prx config server update PORT`。要把正在运行的服务器迁移到新端口，请接着执行 `prx daemon restart`。
- **在前台启动：** `prx serve` 会在任意操作系统上于前台启动服务器。
- **演示：** `prx serve --demo` 会启动装有示例数据的演示环境。它不会影响你自己的数据，适合先试用一下。

## 卸载

```sh
curl -fsSL https://github.com/HappyOnigiri/PRX/releases/latest/download/uninstall.sh | bash
```

## 开发

```sh
make dev  # 启动开发服务器：http://127.0.0.1:7331
make demo # 同上，使用隔离的演示数据
make ci   # 在交付变更前运行全部检查
```

`make demo` 会在 Go 代码变更时重启 API，因此每次都会重新生成演示数据。

## 文档

- [docs/cli/prx.md](docs/cli/prx.md)：Markdown 版的 CLI 参考（英文）。
- [docs/design/](docs/design/README.md)：设计决策及其理由（日文）。
- [docs/development.md](docs/development.md)：验证与发布规则（日文）。

## 参与贡献

欢迎参与贡献！
你可以通过 [Issues](https://github.com/HappyOnigiri/PRX/issues) 报告问题或提出想法，也可以提交 [Pull Request](https://github.com/HappyOnigiri/PRX/pulls)。
同样欢迎改进文档和翻译。
