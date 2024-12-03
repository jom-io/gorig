# Gorig CLI

Gorig 是一个基于 Go 语言的 Web 综合服务框架，提供了一套完整的开发模式和工具链，你可以使用gorig-cli快速创建一个新项目或新模块。

## 安装

使用 npm 全局安装：

```sh
npm install -g gorig-cli
```

或者使用 npx 直接运行：

```sh
npx gorig-cli@latest <command>
```

## 快速开始

### 初始化新项目

使用 `init` 命令创建一个新项目：

```sh
gorig-cli init my-new-project
```

或者使用 npx：

```sh
npx gorig-cli@latest init my-new-project
```

这将在当前目录下创建一个新项目，包含 `_cmd/main.go`、`domain/init.go`、`cron/cron.go` 等基本文件和目录。

### 创建新模块

在项目根目录下使用 `create` 命令创建一个新模块：

```sh
gorig-cli create user
```

或者使用 npx：

```sh
npx gorig-cli@latest create user
```

这将在项目中创建一个名为 `user` 的模块，包含 `api/`、`internal/`、`model/` 等文件夹和必要的代码。

### 运行项目

进入项目目录后，可以使用以下命令运行项目：

```sh
cd my-new-project
go run _cmd/main.go
```

或者编译后运行：

```sh
go build -o my-new-project _cmd/main.go && ./my-new-project
```

