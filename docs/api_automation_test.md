# HTTP API 自动化测试指南

本文档介绍了如何运行和使用项目的HTTP API自动化测试，这些测试既可以用于当前阶段里程碑验收，也是未来进行回归测试的基础。

## 测试结构

项目包含两种类型的测试：

1. **单元测试** - 位于 [api/http_api_test.go](file:///d:/Work/hsch/ffmpeg-go/api/http_api_test.go)，针对HTTP接口进行测试
2. **集成测试** - 位于 [integration/api_integration_test.go](file:///d:/Work/hsch/ffmpeg-go/integration/api_integration_test.go)，进行端到端的完整测试

## 运行测试

### 运行单元测试

```bash
go test -v ./api
```

这个命令将运行所有HTTP API相关的单元测试，包括：
- 健康检查接口测试
- 提交视频编辑任务接口测试
- 获取任务状态接口测试
- 取消任务接口测试
- 完整工作流程测试

### 运行集成测试

```bash
go test -v ./integration
```

这个命令将运行端到端的集成测试，包括：
- 完整的API测试套件
- 实际任务提交和处理流程
- 任务状态跟踪和验证

### 运行所有测试

```bash
go test -v ./...
```

这个命令将运行项目中的所有测试。

## 测试内容

### 单元测试 (api/http_api_test.go)

1. **TestHealthCheck** - 测试 `/health` 端点
2. **TestSubmitVideoEdit** - 测试 POST `/api/v1/video/edit` 端点
3. **TestSubmitVideoEditInvalidJSON** - 测试提交无效JSON的错误处理
4. **TestGetVideoEditStatus** - 测试 GET `/api/v1/video/edit/{taskId}` 端点
5. **TestGetVideoEditStatusNotFound** - 测试获取不存在任务的错误处理
6. **TestCancelVideoEdit** - 测试 DELETE `/api/v1/video/edit/{taskId}` 端点
7. **TestCancelVideoEditNotFound** - 测试取消不存在任务的错误处理
8. **TestFullWorkflow** - 测试完整的任务提交、处理和状态查询流程

### 集成测试 (integration/api_integration_test.go)

1. **TestAPISuite** - 完整的API测试套件，包含以下子测试：
   - **HealthCheck** - 健康检查
   - **SubmitVideoEdit** - 提交视频编辑任务
   - **GetTaskStatus** - 获取任务状态
   - **WaitForTaskCompletion** - 等待任务完成
   - **GetFinalTaskStatus** - 获取任务最终状态
   - **SubmitInvalidTask** - 提交无效任务
   - **GetNonExistentTask** - 获取不存在的任务

## 测试特点

### 1. 独立性
所有测试都是独立的，不依赖外部服务，使用 httptest 创建模拟服务器。

### 2. 完整性
测试覆盖了所有HTTP接口和各种边界情况，包括：
- 正常流程
- 错误处理
- 无效输入
- 不存在的资源

### 3. 自动化
测试完全自动化，不需要手动干预，适合CI/CD流程集成。

### 4. 可重复性
测试每次都从相同的状态开始，确保结果的一致性。

## 用于里程碑验收

这些测试可以作为里程碑验收的工具：

1. **功能验证** - 确保所有HTTP接口按预期工作
2. **回归保证** - 确保新功能不会破坏现有功能
3. **质量度量** - 提供可量化的质量指标

## 用于回归测试

测试可以轻松集成到CI/CD流程中：

1. **提交时运行** - 每次代码提交时自动运行
2. **定时运行** - 定期运行以确保系统稳定性
3. **发布前运行** - 在发布新版本前运行以确保质量

## 扩展测试

可以根据需要添加更多测试用例：

1. 添加更多边界情况测试
2. 添加性能测试
3. 添加安全测试
4. 添加不同视频格式的测试

## 注意事项

1. 集成测试需要示例数据文件，如果文件不存在，测试会自动跳过
2. 测试使用内存队列和工作池，不会影响实际系统
3. 测试完成后会自动清理资源