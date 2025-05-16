# Gorig [![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/jom-io/gorig)

**Gorig** is a comprehensive web service framework based on the Go programming language. It provides a complete development model and toolchain. You can quickly create a new project or module using `gorig-cli`.

📚 **Project Wiki**: [https://deepwiki.com/jom-io/gorig](https://deepwiki.com/jom-io/gorig)  
🔧 **Operations Dashboard**: [https://github.com/jom-io/gorig-om](https://github.com/jom-io/gorig-om)

## Direct Installation

```sh
go get github.com/jom-io/gorig@latest
```

## Using gorig-cli

Install globally via npm:

```sh
npm install -g gorig-cli
```

Or run directly using npx:

```sh
npx gorig-cli@latest <command>
```

## Quick Start

### Initialize a New Project

Use the `init` command to create a new project:

```sh
gorig-cli init my-new-project
```

Or use npx:

```sh
npx gorig-cli@latest init my-new-project
```

This will create a new project in the current directory, including basic files and directories such as `_cmd/main.go`, `domain/init.go`, and `cron/cron.go`.

### Create a New Module

Create a new module using the `create` command from the project's root directory:

```sh
gorig-cli create user
```

Or use npx:

```sh
npx gorig-cli@latest create user
```

This will create a module named `user`, including directories such as `api/`, `internal/`, `model/`, and essential boilerplate code.

### Running the Project

Navigate to your project directory and run the project using:

```sh
cd my-new-project
go run _cmd/main.go
```

Or compile and run:

```sh
go build -o my-new-project _cmd/main.go && ./my-new-project
```

