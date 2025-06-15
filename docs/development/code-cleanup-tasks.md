# Code Cleanup Tasks

## Dead Code to Remove

### 1. Unused Variables in `runner.go`
**Location**: Lines 21-23
```go
var (
    cachedFormatterPath string
    formatterPathOnce   sync.Once
    formatterPathErr    error
)
```
**Action**: Remove these variables and the `sync` import

### 2. Unused Function in `version.go`
**Location**: Line 16
```go
func GetVersion() string {
    return version
}
```
**Action**: Remove this function (use `GetVersionInfo()` instead)

### 3. Unused Logger Functions in `logger.go`
**Functions to remove**:
- `LogError()` (line 124)
- `LogWarn()` (line 130)
- `WithContext()` (line 135)
- `WithWorker()` (line 140)
- `WithFile()` (line 145)

**Action**: Remove these wrapper functions as the codebase uses `Logger` instance directly

### 4. Incomplete Kong CLI Implementation
**Location**: `kong.go`, line 21
```go
// TODO: Call the actual runWatch logic here
```
**Action**: Either complete the Kong CLI implementation or remove it entirely

## Summary
- Remove 5 unused logger wrapper functions
- Remove 3 unused caching variables and associated import
- Remove 1 unused version function
- Decide on Kong CLI implementation (complete or remove)

All other code appears to be actively used and properly integrated.