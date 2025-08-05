# 素材上传与预处理功能说明

## 功能概述

本功能提供了一个新的API端点，用于上传视频素材并将其预处理为TS格式。该功能通过任务队列系统进行异步处理，确保系统能够高效处理大量上传请求。

## API端点

### 上传素材

**URL**: `POST /api/v1/material/upload`

**参数**:
- `file` (multipart/form-data): 要上传的视频文件

**响应**:
- `taskId` (string): 任务ID，用于查询处理状态
- `status` (string): 任务状态，上传成功后为"accepted"
- `message` (string): 响应消息

**示例请求**:
```bash
curl -X POST \
  http://localhost:8080/api/v1/material/upload \
  -H 'content-type: multipart/form-data' \
  -F 'file=@/path/to/video.mp4'
```

**示例响应**:
```json
{
  "taskId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "accepted",
  "message": "Material upload accepted for processing"
}
```

### 查询任务状态

**URL**: `GET /api/v1/video/edit/{taskId}`

**参数**:
- `taskId` (path): 上传时返回的任务ID

**响应**:
- `taskId` (string): 任务ID
- `status` (string): 任务当前状态 (pending, processing, completed, failed)
- `progress` (number): 处理进度 (0-1)
- `message` (string, optional): 错误信息（如果任务失败）
- `created` (string): 任务创建时间
- `started` (string, optional): 任务开始处理时间
- `finished` (string, optional): 任务完成时间
- `outputUrl` (string, optional): 输出文件路径（任务完成后）

## 处理流程

1. 用户通过 `/api/v1/material/upload` 端点上传视频文件
2. 服务器接收文件并保存到临时目录
3. 创建预处理任务并添加到任务队列
4. 工作池中的工作者获取任务并开始处理
5. 使用FFmpeg将视频文件转换为TS格式
6. 处理完成后更新任务状态并保存结果路径

## 技术实现

### 素材预处理服务

素材预处理由 [MaterialPreprocessorService](file:///D:/Work/hsch/ffmpeg-go/service/material_preprocessor.go#L16-L16) 处理，主要功能包括：

1. 验证源文件是否存在
2. 使用FFmpeg将视频转换为TS格式
3. 更新任务状态和结果

转换命令示例：
```bash
ffmpeg -i input.mp4 -c copy -bsf:v h264_mp4toannexb -f mpegts output.ts
```

该命令使用以下参数：
- `-c copy`: 直接复制音频和视频流，不重新编码
- `-bsf:v h264_mp4toannexb`: 应用H.264比特流过滤器，将MP4格式转换为适合TS的格式
- `-f mpegts`: 指定输出格式为MPEG-TS

### 任务处理

任务处理流程如下：
1. 任务被添加到队列中，初始状态为"pending"
2. 工作者获取任务并将其状态更新为"processing"
3. 调用素材预处理服务执行转换
4. 转换成功后，任务状态更新为"completed"，进度设置为1.0
5. 转换失败时，任务状态更新为"failed"，并记录错误信息

## 错误处理

可能出现的错误包括：
1. 文件上传失败
2. 源文件不存在
3. FFmpeg转换失败
4. 磁盘空间不足

所有错误都会被记录并更新到任务状态中，用户可以通过查询任务状态接口获取错误详情。

## 性能优化

1. 使用流复制（`-c copy`）避免重新编码，提高处理速度
2. 通过任务队列实现异步处理，避免阻塞API响应
3. 使用工作池管理并发处理，控制资源使用