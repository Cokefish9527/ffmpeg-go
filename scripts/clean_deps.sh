#!/bin/bash

# 清理模块缓存
go clean -modcache

# 验证模块
go mod tidy

# 验证依赖
go mod download
go mod verify

echo "Dependencies cleaned and verified successfully"