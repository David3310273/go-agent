---
name: file-writer
description: Write content to a specified file path. Creates the file if it does not exist, or overwrites it if it does.
priority: 10
---

# File Writer Tool

## Parameters
- `path` (string, required): The absolute or relative file path to write to
- `content` (string, required): The text content to write into the file

## Behavior
- Creates the file if it does not exist
- Overwrites the file if it already exists
- File permissions are set to 0644

## Example
Write "Hello World" to `/tmp/hello.txt`:
```
WriteToFile("/tmp/hello.txt", "Hello World")
```
