# Comand line app

This app is a command line interface for the simple agent core. In order to provide more functions based on the go-agent framework.

**This app is not a professional command line interface which can be used directly in production**. It's more like an experimental tool based on the agent core. Please extend your own functions if necessary.

## Provided functions

Please view the help page in the app for more details. Here are some core functions:

- /ask: Ask a question to the agent.
- /skill ls: List all skills in the agent.
- /kb add: Add a knowledge doc to the agent's knowledge.
- /kb ls: List all knowledge doc names in the agent's knowledge.
- /mcp ls: List all registered MCP servers.

## Not provided functions

- no user management implementions. Such as register, auth, login. Only interface provided. It's not important compared with the framework infrastructure.
- cannot change context in runtime. If you want to add a skill or knowledge doc, well prepared in advance.

## Start the app

```bash
cd <cmd root directory>
go run main.go
```

