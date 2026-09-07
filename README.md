# Go-agent

An interface-implemented, multi-layered agent framework. Simple but not easy.

## Architecture

Using param injection from outside app to start eventual tool call

![architecture](https://github.com/David3310273/go-agent/blob/main/images/layer.png?raw=true)

### Main entities

- App: user oriented service or app, such as server, command or web application.
- AgentCore: entrypoint to app, mainly for status management and session management, with customized system prompt, history, knowledge base and tools
- Session: a conversation between user and agent, including questions, context(injected from AgentCore) and own history
- Provider: model adapter between framework and specific llms
- Context: agent/session runtime context. including prompt, history, knowledge base, available providers and tools
- Tool: local tools for agent.
- Sandbox: decorator of tool calls
- Benchmark: single agent metrics watcher, should not belong to any agents.

### Design Principles

- Param injection rather than variables in struct
- Only composition on final interface, interface unit should have single responsibility.

### Directory Structure

- core: core framework logics and interfaces(such as entities mentioned above). Should not include:
  - any specific agents, sessions, logs, etc.
  - import other specific implementation packages. Can only include other core interfaces.
- agent: agent implementations. Create your own agent logic here. Already have an example called **simple agent**. Should not include:
  - app related configs(such as language, timeZone...), logs, etc.
- providers: only focus on specific model insertion(such as claude...), if you want to use agent data, using param injection rather than define the variable in provider struct.
- app: agent app for user. such as server, web app, command line...