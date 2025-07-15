import React, { useState, useCallback } from 'react';

const MailboxBatchOperations = ({
  selectedMailboxes = [],
  onClearSelection,
  onDelete,
  className = '',
  style = {}
}) => {
  const [confirmAction, setConfirmAction] = useState(null);
  
  // 处理删除
  const handleDelete = useCallback(async () => {
    try {
      await onDelete?.(selectedMailboxes);
      setConfirmAction(null);
    } catch (error) {
      console.error('批量删除失败:', error);
      alert('删除失败，请重试');
    }
  }, [selectedMailboxes, onDelete]);
  
  // 确认对话框组件
  const ConfirmDialog = ({ action, onConfirm, onCancel }) => (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000
      }}
      onClick={onCancel}
    >
      <div
        style={{
          backgroundColor: 'white',
          padding: '24px',
          borderRadius: '8px',
          maxWidth: '400px',
          margin: '20px'
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h3 style={{ margin: '0 0 16px 0', color: '#333' }}>
          {action.title}
        </h3>
        <p style={{ margin: '0 0 20px 0', color: '#666', lineHeight: '1.5' }}>
          {action.message}
        </p>
        <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
          <button
            onClick={onCancel}
            style={{
              padding: '8px 16px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              backgroundColor: '#fff',
              cursor: 'pointer'
            }}
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            style={{
              padding: '8px 16px',
              border: '1px solid #f44336',
              borderRadius: '4px',
              backgroundColor: '#f44336',
              color: 'white',
              cursor: 'pointer'
            }}
          >
            确认删除
          </button>
        </div>
      </div>
    </div>
  );
  
  if (selectedMailboxes.length === 0) {
    return null;
  }
  
  return (
    <div className={`mailbox-batch-operations ${className}`} style={style}>
      {/* 主操作栏 */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        padding: '12px 16px',
        backgroundColor: '#e3f2fd',
        borderRadius: '8px',
        border: '1px solid #2196F3',
        marginBottom: '16px'
      }}>
        {/* 选择信息 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flex: 1 }}>
          <span style={{ fontSize: '14px', color: '#1976d2', fontWeight: '500' }}>
            已选择 {selectedMailboxes.length} 个邮箱
          </span>
          <button
            onClick={onClearSelection}
            style={{
              background: 'none',
              border: 'none',
              color: '#1976d2',
              cursor: 'pointer',
              fontSize: '12px',
              textDecoration: 'underline'
            }}
          >
            取消选择
          </button>
        </div>
        
        {/* 操作按钮 */}
        <div style={{ display: 'flex', gap: '8px' }}>
          <button
            onClick={() => setConfirmAction({
              title: '确认删除邮箱',
              message: `确定要删除 ${selectedMailboxes.length} 个邮箱吗？这将删除这些邮箱的所有邮件，此操作不可撤销。`,
              action: handleDelete
            })}
            style={{
              padding: '8px 16px',
              border: '1px solid #f44336',
              borderRadius: '4px',
              backgroundColor: '#fff',
              color: '#f44336',
              cursor: 'pointer',
              fontSize: '14px',
              display: 'flex',
              alignItems: 'center',
              gap: '4px'
            }}
          >
            <span>🗑️</span>
            <span>删除邮箱</span>
          </button>
        </div>
      </div>
      
      {/* 确认对话框 */}
      {confirmAction && (
        <ConfirmDialog
          action={confirmAction}
          onConfirm={confirmAction.action}
          onCancel={() => setConfirmAction(null)}
        />
      )}
    </div>
  );
};

export default MailboxBatchOperations;