// 更新系统信息
function updateSystemInfo(data) {
    // 解析活跃Worker数量
    const activeWorkers = data.activeWorkers || '0/0'; // 确保默认值
    document.getElementById('active-workers').innerText = activeWorkers;
    
    // 解析CPU使用率
    const cpuUsage = data.cpuUsage || '0%';
    document.getElementById('cpu-usage').innerText = cpuUsage;

    // 解析内存使用
    const memoryUsage = data.memoryUsage || '0 MB';
    document.getElementById('memory-usage').innerText = memoryUsage;
}