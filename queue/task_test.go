package queue

import (
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
)

func TestInMemoryTaskQueue_AddAndGet(t *testing.T) {
	// 创建任务队列实例
	queue := NewInMemoryTaskQueue()
	
	// 创建测试任务
	task := &Task{
		ID:      "test-task-1",
		Spec:    map[string]interface{}{"test": "data"},
		Status:  "pending",
		Created: time.Now(),
	}
	
	// 添加任务到队列
	err := queue.Add(task)
	assert.NoError(t, err)
	
	// 从队列获取任务
	retrievedTask, err := queue.Get("test-task-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	
	// 验证任务信息
	assert.Equal(t, task.ID, retrievedTask.ID)
	assert.Equal(t, task.Status, retrievedTask.Status)
	assert.Equal(t, task.Spec, retrievedTask.Spec)
}

func TestInMemoryTaskQueue_GetNonExistentTask(t *testing.T) {
	// 创建任务队列实例
	queue := NewInMemoryTaskQueue()
	
	// 尝试获取不存在的任务
	task, err := queue.Get("non-existent-task")
	assert.NoError(t, err)
	assert.Nil(t, task)
}

func TestInMemoryTaskQueue_List(t *testing.T) {
	// 创建任务队列实例
	queue := NewInMemoryTaskQueue()
	
	// 添加多个测试任务
	tasks := []*Task{
		{
			ID:      "test-task-1",
			Spec:    map[string]interface{}{"test": "data1"},
			Status:  "pending",
			Created: time.Now(),
		},
		{
			ID:      "test-task-2",
			Spec:    map[string]interface{}{"test": "data2"},
			Status:  "processing",
			Created: time.Now(),
		},
	}
	
	for _, task := range tasks {
		err := queue.Add(task)
		assert.NoError(t, err)
	}
	
	// 获取任务列表
	retrievedTasks, err := queue.List()
	assert.NoError(t, err)
	assert.Len(t, retrievedTasks, 2)
	
	// 验证任务信息
	taskMap := make(map[string]*Task)
	for _, task := range retrievedTasks {
		taskMap[task.ID] = task
	}
	
	for _, task := range tasks {
		retrievedTask, exists := taskMap[task.ID]
		assert.True(t, exists)
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, task.Status, retrievedTask.Status)
		assert.Equal(t, task.Spec, retrievedTask.Spec)
	}
}

func TestInMemoryTaskQueue_Remove(t *testing.T) {
	// 创建任务队列实例
	queue := NewInMemoryTaskQueue()
	
	// 添加测试任务
	task := &Task{
		ID:      "test-task-1",
		Spec:    map[string]interface{}{"test": "data"},
		Status:  "pending",
		Created: time.Now(),
	}
	
	err := queue.Add(task)
	assert.NoError(t, err)
	
	// 删除任务
	err = queue.Remove("test-task-1")
	assert.NoError(t, err)
	
	// 验证任务已被删除
	deletedTask, err := queue.Get("test-task-1")
	assert.NoError(t, err)
	assert.Nil(t, deletedTask)
}

func TestInMemoryTaskQueue_Update(t *testing.T) {
	// 创建任务队列实例
	queue := NewInMemoryTaskQueue()
	
	// 添加测试任务
	originalTask := &Task{
		ID:      "test-task-1",
		Spec:    map[string]interface{}{"test": "data"},
		Status:  "pending",
		Created: time.Now(),
	}
	
	err := queue.Add(originalTask)
	assert.NoError(t, err)
	
	// 更新任务
	updatedTask := &Task{
		ID:      "test-task-1",
		Spec:    map[string]interface{}{"test": "updated-data"},
		Status:  "processing",
		Created: originalTask.Created,
		Started: time.Now(),
	}
	
	err = queue.Update(updatedTask)
	assert.NoError(t, err)
	
	// 验证任务已更新
	retrievedTask, err := queue.Get("test-task-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	
	assert.Equal(t, updatedTask.ID, retrievedTask.ID)
	assert.Equal(t, updatedTask.Status, retrievedTask.Status)
	assert.Equal(t, updatedTask.Spec, retrievedTask.Spec)
	assert.Equal(t, updatedTask.Started, retrievedTask.Started)
}