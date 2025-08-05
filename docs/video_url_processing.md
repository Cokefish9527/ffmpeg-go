# 视频URL处理功能说明

## 功能概述

本功能提供了一个新的API端点，用于接收视频文件的URL，自动下载文件到本地，将其转换为TS格式，并返回转换后文件的本地路径。该功能整合了文件下载、格式转换和异步任务处理。

## API端点

### 处理视频URL

**URL**: `POST /api/v1/video/url`

**参数**:
- `url` (string, required): 视频文件的URL地址

**响应**:
- `status` (string): 处理状态 ("success" 或 "error")
- `message` (string): 响应消息
- `tsFilePath` (string, optional): 转换后的TS文件本地路径（仅在成功时返回）
- `error` (string, optional): 错误信息（仅在失败时返回）

**示例请求**:
```bash
curl -X POST \
  http://localhost:8080/api/v1/video/url \
  -H 'content-type: application/json' \
  -d '{
    "url": "http://example.com/video.mp4"
  }'
```

**成功响应示例**:
```json
{
  "status": "success",
  "message": "Video converted successfully",
  "tsFilePath": "./temp/550e8400-e29b-41d4-a716-446655440000_temp.ts"
}
```

**错误响应示例**:
```json
{
  "status": "error",
  "message": "Failed to download file",
  "error": "Get http://example.com/video.mp4: dial tcp: lookup example.com: no such host"
}
```

## 处理流程

1. 用户通过 `/api/v1/video/url` 端点提交视频URL
2. 服务器验证URL参数的有效性
3. 生成唯一的任务ID并创建临时目录用于存储文件
4. 从指定URL下载视频文件到本地临时位置
5. 创建预处理任务并添加到任务队列
6. 等待任务完成（同步等待，最多30秒）
7. 返回转换结果给用户

## 技术实现

### 文件下载

使用Go标准库的HTTP客户端下载文件：

```go
resp, err := http.Get(url)
// ... 错误处理 ...
defer resp.Body.Close()

out, err := os.Create(filepath)
// ... 错误处理 ...

_, err = io.Copy(out, resp.Body)
```

### 格式转换

格式转换由 [MaterialPreprocessorService](file:///D:/Work/hsch/ffmpeg-go/service/material_preprocessor.go#L16-L16) 处理，使用FFmpeg命令：

```bash
ffmpeg -i input.mp4 -c copy -bsf:v h264_mp4toannexb -f mpegts output.ts
```

该命令使用以下参数：
- `-i input.mp4`: 指定输入文件
- `-c copy`: 直接复制音频和视频流，不重新编码
- `-bsf:v h264_mp4toannexb`: 应用H.264比特流过滤器，将MP4格式转换为适合TS的格式
- `-f mpegts`: 指定输出格式为MPEG-TS
- `output.ts`: 输出文件路径

### 异步任务处理

虽然用户接口是同步等待的，但后台使用任务队列系统进行异步处理：

1. 任务被添加到 [InMemoryTaskQueue](file:///D:/Work/hsch/ffmpeg-go/queue/task_queue.go#L62-L66)
2. [WorkerPool](file:///D:/Work/hsch/ffmpeg-go/service/worker_pool.go#L29-L38) 中的工作线程获取任务并处理
3. 主线程轮询任务状态直到完成或超时

## 错误处理

可能出现的错误包括：

1. **参数验证错误**：
   - URL为空
   - 请求格式无效

2. **文件下载错误**：
   - 网络连接问题
   - HTTP状态码非200
   - 磁盘空间不足
   - 文件权限问题

3. **格式转换错误**：
   - 源文件损坏
   - 不支持的视频格式
   - FFmpeg执行失败

4. **超时错误**：
   - 转换过程超过30秒

所有错误都会被捕获并返回给用户清晰的错误信息。

## 性能考虑

1. **同步等待**：当前实现采用同步等待模式，适用于快速转换的场景。对于大文件或高并发场景，建议改为异步模式。

2. **流复制**：使用`-c copy`参数避免重新编码，显著提高转换速度。

3. **超时控制**：设置30秒超时防止长时间阻塞。

4. **资源清理**：临时文件在转换完成后保留在系统中，实际项目中应考虑定期清理策略。

## 安全考虑

1. **URL验证**：应增加对URL格式和协议的验证，防止访问内部网络资源。

2. **文件类型检查**：应验证下载文件的实际类型，防止恶意文件上传。

3. **路径遍历防护**：确保生成的文件路径在预期目录内，防止路径遍历攻击。

4. **资源限制**：应限制下载文件的大小，防止磁盘空间耗尽。

## 使用示例

### 成功案例

用户提交一个有效的视频URL：
```json
{
  "url": "http://example.com/sample.mp4"
}
```

系统将：
1. 下载文件到 `./temp/[task-id]_temp.mp4`
2. 转换为TS格式 `./temp/[task-id]_temp.ts`
3. 返回成功响应和TS文件路径

### 失败案例

如果用户提交的URL无效：
```json
{
  "url": "http://nonexistent-domain.com/video.mp4"
}
```

系统将返回错误响应：
```json
{
  "status": "error",
  "message": "Failed to download file",
  "error": "Get http://nonexistent-domain.com/video.mp4: dial tcp: lookup nonexistent-domain.com: no such host"
}
```