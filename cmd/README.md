# Comand line app

This app is a command line interface for the simple agent core. In order to provide more functions based on the go-agent framework.

**This app is not a professional command line interface which can be used directly in production**. It's more like an experimental tool based on the agent core. Please extend your own functions if necessary.

## UI display

![ui display](https://github.com/David3310273/go-agent/blob/main/images/cmdui.png?raw=true)

## Provided functions

Please view the help page in the app for more details. Here are some core functions:

- /ask: Ask a question to the agent.
- /skill ls: List all skills in the agent.
- /kb add: Add a knowledge doc to the agent's knowledge in the runtime.
- /kb ls: List all knowledge doc names in the agent's knowledge in the runtime.
- /kb rm: Remove a knowledge doc from the agent's knowledge in the runtime.
- /mcp ls: List all registered MCP servers in the runtime.
- /mcp add: Add a MCP server to the agent in the runtime.
- /mcp rm: Remove a MCP server from the agent in the runtime.      

## Not provided features

- no user management implementions. Such as register, auth, login. Only interface provided. It's not important compared with the framework infrastructure.
- cannot change skill/tool in runtime. If you want to add a skill with new tools, make sure them well prepared in advance using MCP server. (Of course advanced features such as tool plugin can solve this but now it's not necessary.)
- cannot choose the dialog history list. If you open a new page, the pervious dialog history will be cleared. (Just focus on the core features.)
- don't have tool confirm feature because it's no dangerous write operation.

## Start the app

```bash
cd <cmd root directory>
go run main.go
```

