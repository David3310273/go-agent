# Changelog

## v1.1.0: (coming soon...)

feature:

- add command line for simple agent, such as:
  - add skill
  - add knowledge base
  - add user prompt
  - add mcp server
  - ask question and show thoughts

- move related doc to app folder to make agent more like bare cores

## v1.0.6

final fixes before new feature:

- tide knowledgebase to make it easier to add new document
- support customize sandbox for one tool

## v1.0.5: 2026-10-04

fix:

- log tide and error code translation

feature:

- multiple model support
- support sandbox

## v1.0.4

fix: 

- fix the model param to support the llm call with model params
- fix the prompt to support parallel execution and sub session execution
- architecture improvement in harness and tools

feature:

- support tool parallel execution
- support sub session execution

## v1.0.3: 2026-09-22

fix:

- allow user to make sure the operation
- allow stop the process from user
- support store tools in a session

feature:

- support calling MCP server

## V1.0.2: 2026-09-16

fixs:

- schema injection from app, adding more CRUD methods, publish CRUD api of kb
- add harness interface/component to organize the context
- move searching kb to the tool

feature:

- publish tools as MCP service
- support dynamic skills and tools


## V1.0.1

fixs:

- path management
- support loading history in session
- support for streaming mode

feature:

- support local kb
- support dynamic loading context from local kb.

## V1.0.0: 2026-09-07

- initial release: core framework, with simple agent and restful http server app
  - happy path for the simple agent implementation