package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/monitor/stats": {
            "get": {
                "description": "获取系统资源使用情况统计信息，包括CPU、内存、磁盘等",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "获取系统统计信息",
                "operationId": "get-system-stats",
                "responses": {
                    "200": {
                        "description": "系统统计信息",
                        "schema": {
                            "$ref": "#/definitions/api.SystemStats"
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks": {
            "get": {
                "description": "获取所有任务列表，支持按状态和优先级筛选",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "获取任务列表",
                "operationId": "get-tasks",
                "parameters": [
                    {
                        "type": "string",
                        "description": "任务状态筛选",
                        "name": "status",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "任务优先级筛选",
                        "name": "priority",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "任务列表",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/queue.Task"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks/stats": {
            "get": {
                "description": "获取任务统计信息，包括各种状态的任务数量",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "获取任务统计信息",
                "operationId": "get-task-stats",
                "responses": {
                    "200": {
                        "description": "任务统计信息",
                        "schema": {
                            "$ref": "#/definitions/api.TaskStats"
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks/{taskId}": {
            "get": {
                "description": "根据任务ID获取任务详细信息",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "获取任务详情",
                "operationId": "get-task-detail",
                "parameters": [
                    {
                        "type": "string",
                        "description": "任务ID",
                        "name": "taskId",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "任务详情",
                        "schema": {
                            "$ref": "#/definitions/queue.Task"
                        }
                    },
                    "404": {
                        "description": "任务未找到",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks/{taskId}/executions": {
            "get": {
                "description": "获取指定任务的所有执行历史记录",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "获取任务执行历史",
                "operationId": "get-task-executions",
                "parameters": [
                    {
                        "type": "string",
                        "description": "任务ID",
                        "name": "taskId",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "任务执行历史记录列表",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/queue.TaskExecution"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks/{taskId}/log": {
            "get": {
                "description": "获取指定任务的日志内容",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "获取任务日志",
                "operationId": "get-task-log",
                "parameters": [
                    {
                        "type": "string",
                        "description": "任务ID",
                        "name": "taskId",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "任务日志内容",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "404": {
                        "description": "任务日志未找到",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks/cancel": {
            "post": {
                "description": "取消一个待处理或处理中的任务",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "取消任务",
                "operationId": "cancel-task",
                "parameters": [
                    {
                        "description": "任务取消请求",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.TaskCancelRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "取消成功",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "请求参数错误或任务状态不正确",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "404": {
                        "description": "任务未找到",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks/discard": {
            "post": {
                "description": "丢弃一个已完成或失败的任务",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "丢弃任务",
                "operationId": "discard-task",
                "parameters": [
                    {
                        "description": "任务丢弃请求",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.TaskDiscardRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "丢弃成功",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "请求参数错误或任务状态不正确",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "404": {
                        "description": "任务未找到",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/tasks/retry": {
            "post": {
                "description": "重试一个失败的任务",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "重试失败的任务",
                "operationId": "retry-task",
                "parameters": [
                    {
                        "description": "任务重试请求",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.TaskRetryRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "重试成功",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "请求参数错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "404": {
                        "description": "任务未找到",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/monitor/workers": {
            "get": {
                "description": "获取Worker池的统计信息",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "monitor"
                ],
                "summary": "获取Worker统计信息",
                "operationId": "get-worker-stats",
                "responses": {
                    "200": {
                        "description": "Worker统计信息",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/video/edit": {
            "post": {
                "description": "提交一个新的视频编辑任务",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "video"
                ],
                "summary": "提交视频编辑任务",
                "operationId": "submit-video-edit",
                "parameters": [
                    {
                        "description": "视频编辑请求",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.VideoEditRequest"
                        }
                    }
                ],
                "responses": {
                    "202": {
                        "description": "任务提交成功",
                        "schema": {
                            "$ref": "#/definitions/api.VideoEditResponse"
                        }
                    },
                    "400": {
                        "description": "请求参数错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/video/edit/{id}": {
            "delete": {
                "description": "根据任务ID取消视频编辑任务",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "video"
                ],
                "summary": "取消视频编辑任务",
                "operationId": "cancel-video-edit",
                "parameters": [
                    {
                        "type": "string",
                        "description": "任务ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "任务取消成功",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "请求参数错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "404": {
                        "description": "任务未找到",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            },
            "get": {
                "description": "根据任务ID获取视频编辑任务的状态信息",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "video"
                ],
                "summary": "获取视频编辑任务状态",
                "operationId": "get-video-edit-status",
                "parameters": [
                    {
                        "type": "string",
                        "description": "任务ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "任务状态信息",
                        "schema": {
                            "$ref": "#/definitions/api.TaskStatusResponse"
                        }
                    },
                    "400": {
                        "description": "请求参数错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "404": {
                        "description": "任务未找到",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/video/url": {
            "post": {
                "description": "通过URL下载视频并提交处理任务",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "video"
                ],
                "summary": "处理视频URL",
                "operationId": "handle-video-url",
                "parameters": [
                    {
                        "description": "视频URL请求",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/api.VideoURLRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "处理成功",
                        "schema": {
                            "$ref": "#/definitions/api.VideoURLResponse"
                        }
                    },
                    "400": {
                        "description": "请求参数错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/workerpool/resize": {
            "post": {
                "description": "动态调整工作池中工作线程的数量",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "workerpool"
                ],
                "summary": "调整工作池大小",
                "operationId": "resize-worker-pool",
                "parameters": [
                    {
                        "description": "工作池大小调整请求",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "size": {
                                    "type": "integer"
                                }
                            }
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "工作池调整成功",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "400": {
                        "description": "请求参数错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/workerpool/status": {
            "get": {
                "description": "获取当前工作池的状态信息",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "workerpool"
                ],
                "summary": "获取工作池状态",
                "operationId": "get-worker-pool-status",
                "responses": {
                    "200": {
                        "description": "工作池状态信息",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "内部服务器错误",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "api.OSSOutput": {
            "type": "object",
            "properties": {
                "accessKey": {
                    "description": "AccessKey",
                    "type": "string"
                },
                "bucket": {
                    "description": "Bucket",
                    "type": "string"
                },
                "endpoint": {
                    "description": "Endpoint",
                    "type": "string"
                },
                "key": {
                    "description": "Key",
                    "type": "string"
                },
                "secretKey": {
                    "description": "SecretKey",
                    "type": "string"
                }
            }
        },
        "api.SystemStats": {
            "type": "object",
            "properties": {
                "activeWorkers": {
                    "description": "活跃工作线程数",
                    "type": "integer"
                },
                "cpuUsage": {
                    "description": "CPU使用率",
                    "type": "number"
                },
                "diskTotal": {
                    "description": "总磁盘空间",
                    "type": "integer"
                },
                "diskUsage": {
                    "description": "磁盘使用率",
                    "type": "number"
                },
                "diskUsed": {
                    "description": "已使用磁盘空间",
                    "type": "integer"
                },
                "goroutines": {
                    "description": "Goroutines数量",
                    "type": "integer"
                },
                "memoryTotal": {
                    "description": "总内存",
                    "type": "integer"
                },
                "memoryUsage": {
                    "description": "内存使用率",
                    "type": "number"
                },
                "memoryUsed": {
                    "description": "已使用内存",
                    "type": "integer"
                },
                "taskQueueSize": {
                    "description": "任务队列大小",
                    "type": "integer"
                },
                "timestamp": {
                    "description": "时间戳",
                    "type": "string"
                },
                "workerCount": {
                    "description": "工作线程总数",
                    "type": "integer"
                }
            }
        },
        "api.TaskCancelRequest": {
            "type": "object",
            "properties": {
                "taskId": {
                    "description": "任务ID",
                    "type": "string"
                }
            },
            "required": [
                "taskId"
            ]
        },
        "api.TaskDiscardRequest": {
            "type": "object",
            "properties": {
                "taskId": {
                    "description": "任务ID",
                    "type": "string"
                }
            },
            "required": [
                "taskId"
            ]
        },
        "api.TaskRetryRequest": {
            "type": "object",
            "properties": {
                "taskId": {
                    "description": "任务ID",
                    "type": "string"
                }
            },
            "required": [
                "taskId"
            ]
        },
        "api.TaskStats": {
            "type": "object",
            "properties": {
                "completedTasks": {
                    "description": "已完成任务数",
                    "type": "integer"
                },
                "failedTasks": {
                    "description": "失败任务数",
                    "type": "integer"
                },
                "pendingTasks": {
                    "description": "待处理任务数",
                    "type": "integer"
                },
                "processingTasks": {
                    "description": "处理中任务数",
                    "type": "integer"
                },
                "totalTasks": {
                    "description": "总任务数",
                    "type": "integer"
                }
            }
        },
        "api.TaskStatusResponse": {
            "type": "object",
            "properties": {
                "created": {
                    "description": "创建时间",
                    "type": "string"
                },
                "finished": {
                    "description": "完成时间",
                    "type": "string"
                },
                "message": {
                    "description": "消息",
                    "type": "string"
                },
                "outputUrl": {
                    "description": "输出URL",
                    "type": "string"
                },
                "priority": {
                    "description": "优先级",
                    "type": "integer"
                },
                "progress": {
                    "description": "进度",
                    "type": "number"
                },
                "started": {
                    "description": "开始时间",
                    "type": "string"
                },
                "status": {
                    "description": "状态",
                    "type": "string"
                },
                "taskId": {
                    "description": "任务ID",
                    "type": "string"
                }
            }
        },
        "api.VideoEditRequest": {
            "type": "object",
            "properties": {
                "ossOutput": {
                    "description": "OSS输出配置",
                    "$ref": "#/definitions/api.OSSOutput"
                },
                "outputPath": {
                    "description": "输出路径",
                    "type": "string"
                },
                "priority": {
                    "description": "任务优先级",
                    "type": "integer"
                },
                "spec": {
                    "description": "任务规格",
                    "type": "object"
                }
            }
        },
        "api.VideoEditResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "description": "消息",
                    "type": "string"
                },
                "outputUrl": {
                    "description": "输出URL",
                    "type": "string"
                },
                "status": {
                    "description": "状态",
                    "type": "string"
                },
                "taskId": {
                    "description": "任务ID",
                    "type": "string"
                }
            }
        },
        "api.VideoURLRequest": {
            "type": "object",
            "properties": {
                "url": {
                    "description": "视频URL",
                    "type": "string"
                }
            }
        },
        "api.VideoURLResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "description": "错误信息",
                    "type": "string"
                },
                "message": {
                    "description": "消息",
                    "type": "string"
                },
                "status": {
                    "description": "状态",
                    "type": "string"
                },
                "taskId": {
                    "description": "任务ID",
                    "type": "string"
                },
                "tsFilePath": {
                    "description": "TS文件路径",
                    "type": "string"
                }
            }
        },
        "queue.Task": {
            "type": "object",
            "properties": {
                "created": {
                    "type": "string"
                },
                "error": {
                    "type": "string"
                },
                "executionCount": {
                    "type": "integer"
                },
                "finished": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "lastExecution": {
                    "type": "string"
                },
                "priority": {
                    "$ref": "#/definitions/queue.TaskPriority"
                },
                "progress": {
                    "type": "number"
                },
                "result": {
                    "type": "string"
                },
                "spec": {
                    "type": "object"
                },
                "started": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                }
            }
        },
        "queue.TaskExecution": {
            "type": "object",
            "properties": {
                "created": {
                    "type": "string"
                },
                "error": {
                    "type": "string"
                },
                "executionNumber": {
                    "type": "integer"
                },
                "executionTime": {
                    "type": "integer"
                },
                "finished": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "priority": {
                    "$ref": "#/definitions/queue.TaskPriority"
                },
                "progress": {
                    "type": "number"
                },
                "result": {
                    "type": "string"
                },
                "spec": {
                    "type": "object"
                },
                "started": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "taskId": {
                    "type": "string"
                }
            }
        },
        "queue.TaskPriority": {
            "type": "integer",
            "enum": [
                0,
                1,
                2,
                3
            ],
            "x-enum-varnames": [
                "PriorityLow",
                "PriorityNormal",
                "PriorityHigh",
                "PriorityCritical"
            ]
        }
    }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8084",
	BasePath:         "/api/v1",
	Schemes:          []string{},
	Title:            "FFmpeg Go API",
	Description:      "This is a FFmpeg Go server.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}