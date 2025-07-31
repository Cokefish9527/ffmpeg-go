# Git操作说明文档

## 概述

本文档记录了项目git仓库的修改操作，以及如何验证修改是否成功。

## 仓库修改记录

### 修改时间
2025-08-01

### 修改内容
将git远程仓库地址从 `https://github.com/u2takey/ffmpeg-go.git` 修改为 `https://github.com/Cokefish9527/ffmpeg-go.git`

### 修改文件
`.git/config`

### 修改前内容
```
[remote "origin"]
	url = https://github.com/u2takey/ffmpeg-go.git
	fetch = +refs/heads/*:refs/remotes/origin/*
```

### 修改后内容
```
[remote "origin"]
	url = https://github.com/Cokefish9527/ffmpeg-go.git
	fetch = +refs/heads/*:refs/remotes/origin/*
```

## 验证修改是否成功的方法

### 方法一：使用git remote命令
```bash
git remote -v
```

预期输出：
```
origin  https://github.com/Cokefish9527/ffmpeg-go.git (fetch)
origin  https://github.com/Cokefish9527/ffmpeg-go.git (push)
```

### 方法二：查看.git/config文件
直接查看 `.git/config` 文件中 `[remote "origin"]` 部分的url字段是否为 `https://github.com/Cokefish9527/ffmpeg-go.git`

## 后续操作步骤

### 1. 获取fork仓库的更新（如果需要）
```bash
git fetch origin
```

### 2. 推送本地修改到fork仓库
```bash
git push origin master
```

### 3. 创建新的分支进行开发（推荐）
```bash
git checkout -b feature-branch
```

## 注意事项

1. 确保在执行git命令前已安装Git工具
2. 确保网络连接正常，能够访问GitHub
3. 确保有fork仓库的写入权限
4. 建议在推送前先执行fetch操作，避免冲突
5. 所有本地修改记录都已保留，不会因为仓库地址修改而丢失

## 常见问题及解决方案

### 问题1：git命令未找到
**原因**：系统环境变量中未添加Git路径
**解决方案**：
1. 手动添加Git路径到环境变量，或者
2. 使用完整路径执行git命令，如：`"C:\Program Files\Git\bin\git.exe" remote -v`

### 问题2：推送时权限被拒绝
**原因**：没有fork仓库的写入权限或者认证信息不正确
**解决方案**：
1. 确认已登录GitHub账号
2. 检查SSH密钥配置或使用HTTPS认证方式
3. 确认fork仓库地址正确

### 问题3：推送时出现冲突
**原因**：fork仓库与upstream仓库存在差异
**解决方案**：
1. 先执行 `git fetch origin` 获取最新代码
2. 如有必要，执行 `git merge origin/master` 合并远程分支
3. 解决可能的冲突后再推送