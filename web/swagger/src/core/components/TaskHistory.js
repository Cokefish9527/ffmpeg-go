import React from 'react';
import { Table, Button } from 'antd';

const TaskHistory = ({ data }) => {
    const columns = [
        {
            title: '执行序号',
            dataIndex: 'ExecutionNumber',
            key: 'ExecutionNumber',
        },
        {
            title: '状态',
            dataIndex: 'Status',
            key: 'Status',
        },
        {
            title: '开始时间',
            dataIndex: 'Started',
            key: 'Started',
            render: text => text ? text.toLocaleString() : '-'
        },
        {
            title: '结束时间',
            dataIndex: 'Finished',
            key: 'Finished',
            render: text => text ? text.toLocaleString() : '-'
        },
        {
            title: '执行时间',
            dataIndex: 'ExecutionTime',
            key: 'ExecutionTime',
            render: text => `${text}ms`
        },
        {
            title: '错误信息',
            dataIndex: 'Error',
            key: 'Error',
        },
    ];

    return (
        <div>
            <h3>任务执行历史</h3>
            <Button type="primary" style={{ float: 'right' }}>×</Button>
            <Table dataSource={data} columns={columns} />
        </div>
    );
};

export default TaskHistory;