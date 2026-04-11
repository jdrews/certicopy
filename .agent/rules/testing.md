---
description: Run tests after code changes
---

After making any changes to the Go source code, you MUST run `go test ./...` from the project root to ensure that no regressions were introduced. If tests fail, you must analyze the failure and attempt to fix it or report it to the user.
