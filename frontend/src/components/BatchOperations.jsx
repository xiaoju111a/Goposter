import React, { useState, useCallback, useEffect } from 'react';

const BatchOperations = ({
  selectedEmails = [],
  allEmails = [],
  onSelectAll,
  onSelectNone,
  onDelete,
  onMove,
  onMarkRead,
  onMarkUnread,
  onAddTag,
  onRemoveTag,
  onExport,
  availableMailboxes = [],
  availableTags = [],
  className = '',
  style = {}
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [operation, setOperation] = useState(null);
  const [moveTarget, setMoveTarget] = useState('');
  const [tagInput, setTagInput] = useState('');
  const [exportFormat, setExportFormat] = useState('json');
  const [confirmAction, setConfirmAction] = useState(null);
  
  // 处理删除
  const handleDelete = useCallback(async () => {
    try {
      await onDelete?.(selectedEmails);
      setConfirmAction(null);
      setIsOpen(false);
    } catch (error) {
      console.error('批量删除失败:', error);
      alert('删除失败，请重试');
    }
  }, [selectedEmails, onDelete]);
  
  // 处理移动
  const handleMove = useCallback(async () => {
    if (!moveTarget) {
      alert('请选择目标邮箱');
      return;
    }
    
    try {
      await onMove?.(selectedEmails, moveTarget);
      setMoveTarget('');
      setOperation(null);
      setIsOpen(false);
    } catch (error) {
      console.error('批量移动失败:', error);
      alert('移动失败，请重试');
    }
  }, [selectedEmails, moveTarget, onMove]);
  
  // 处理标记已读
  const handleMarkRead = useCallback(async () => {
    try {
      await onMarkRead?.(selectedEmails);
      setIsOpen(false);
    } catch (error) {
      console.error('标记已读失败:', error);
      alert('操作失败，请重试');
    }
  }, [selectedEmails, onMarkRead]);
  
  // 处理标记未读
  const handleMarkUnread = useCallback(async () => {
    try {
      await onMarkUnread?.(selectedEmails);
      setIsOpen(false);
    } catch (error) {
      console.error('标记未读失败:', error);
      alert('操作失败，请重试');
    }
  }, [selectedEmails, onMarkUnread]);
  
  // 处理添加标签
  const handleAddTag = useCallback(async () => {
    if (!tagInput.trim()) {
      alert('请输入标签');
      return;
    }
    
    try {
      await onAddTag?.(selectedEmails, tagInput.trim());
      setTagInput('');
      setOperation(null);
      setIsOpen(false);
    } catch (error) {
      console.error('添加标签失败:', error);
      alert('操作失败，请重试');
    }
  }, [selectedEmails, tagInput, onAddTag]);
  
  // 处理导出
  const handleExport = useCallback(async () => {
    try {
      await onExport?.(selectedEmails, exportFormat);
      setOperation(null);
      setIsOpen(false);
    } catch (error) {
      console.error('导出失败:', error);
      alert('导出失败，请重试');
    }
  }, [selectedEmails, exportFormat, onExport]);

  // 操作配置
  const operations = [
    {
      id: 'delete',
      label: '删除邮件',
      icon: '🗑️',
      color: '#f44336',
      requiresConfirm: true,
      action: () => setConfirmAction({
        title: '确认删除',
        message: `确定要删除 ${selectedEmails.length} 封邮件吗？此操作不可撤销。`,
        action: handleDelete
      })
    },
    {
      id: 'move',
      label: '移动到',
      icon: '📂',
      color: '#2196F3',
      hasOptions: true,
      component: MoveOptions
    },
    {
      id: 'markRead',
      label: '标记为已读',
      icon: '✅',
      color: '#4CAF50',
      action: handleMarkRead
    },
    {
      id: 'markUnread',
      label: '标记为未读',
      icon: '📩',
      color: '#FF9800',
      action: handleMarkUnread
    },
    {
      id: 'addTag',
      label: '添加标签',
      icon: '🏷️',
      color: '#9C27B0',
      hasOptions: true,
      component: TagOptions
    },
    {
      id: 'export',
      label: '导出邮件',
      icon: '💾',
      color: '#607D8B',
      hasOptions: true,
      component: ExportOptions
    }
  ];
  
  // 处理全选
  const handleSelectAll = useCallback(() => {
    onSelectAll?.(allEmails);
  }, [allEmails, onSelectAll]);
  
  // 处理取消全选
  const handleSelectNone = useCallback(() => {
    onSelectNone?.();
  }, [onSelectNone]);
  
  // 选择状态
  const isAllSelected = selectedEmails.length === allEmails.length && allEmails.length > 0;
  const isPartialSelected = selectedEmails.length > 0 && selectedEmails.length < allEmails.length;
  
  // 移动选项组件
  function MoveOptions() {
    return (
      <div>
        <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '6px' }}>
          选择目标邮箱:
        </label>
        <select
          value={moveTarget}
          onChange={(e) => setMoveTarget(e.target.value)}
          style={{
            width: '100%',
            padding: '8px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            marginBottom: '12px'
          }}
        >
          <option value="">请选择邮箱</option>
          {availableMailboxes.map(mailbox => (
            <option key={mailbox} value={mailbox}>
              {mailbox}
            </option>
          ))}
        </select>
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
          <button
            onClick={() => setOperation(null)}
            style={{
              padding: '6px 12px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              backgroundColor: '#fff',
              cursor: 'pointer'
            }}
          >
            取消
          </button>
          <button
            onClick={handleMove}
            disabled={!moveTarget}
            style={{
              padding: '6px 12px',
              border: '1px solid #2196F3',
              borderRadius: '4px',
              backgroundColor: moveTarget ? '#2196F3' : '#ccc',
              color: 'white',
              cursor: moveTarget ? 'pointer' : 'not-allowed'
            }}
          >
            移动
          </button>
        </div>
      </div>
    );
  }
  
  // 标签选项组件
  function TagOptions() {
    return (
      <div>
        <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '6px' }}>
          输入标签名称:
        </label>
        <input
          type="text"
          value={tagInput}
          onChange={(e) => setTagInput(e.target.value)}
          placeholder="输入新标签或选择现有标签"
          style={{
            width: '100%',
            padding: '8px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            marginBottom: '8px'
          }}
          onKeyPress={(e) => e.key === 'Enter' && handleAddTag()}
        />
        
        {availableTags.length > 0 && (
          <div>
            <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '6px' }}>
              或选择现有标签:
            </label>
            <div style={{ 
              display: 'flex', 
              flexWrap: 'wrap', 
              gap: '4px', 
              marginBottom: '12px',
              maxHeight: '80px',
              overflowY: 'auto'
            }}>
              {availableTags.map((tag, index) => (
                <button
                  key={index}
                  onClick={() => setTagInput(tag)}
                  style={{
                    padding: '4px 8px',
                    border: '1px solid #e0e0e0',
                    borderRadius: '12px',
                    backgroundColor: tagInput === tag ? '#e3f2fd' : '#f5f5f5',
                    cursor: 'pointer',
                    fontSize: '11px',
                    color: '#666'
                  }}
                >
                  {tag}
                </button>
              ))}
            </div>
          </div>
        )}
        
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
          <button
            onClick={() => setOperation(null)}
            style={{
              padding: '6px 12px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              backgroundColor: '#fff',
              cursor: 'pointer'
            }}
          >
            取消
          </button>
          <button
            onClick={handleAddTag}
            disabled={!tagInput.trim()}
            style={{
              padding: '6px 12px',
              border: '1px solid #9C27B0',
              borderRadius: '4px',
              backgroundColor: tagInput.trim() ? '#9C27B0' : '#ccc',
              color: 'white',
              cursor: tagInput.trim() ? 'pointer' : 'not-allowed'
            }}
          >
            添加标签
          </button>
        </div>
      </div>
    );
  }
  
  // 导出选项组件
  function ExportOptions() {
    return (
      <div>
        <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '6px' }}>
          选择导出格式:
        </label>
        <select
          value={exportFormat}
          onChange={(e) => setExportFormat(e.target.value)}
          style={{
            width: '100%',
            padding: '8px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            marginBottom: '12px'
          }}
        >
          <option value="json">JSON 格式</option>
          <option value="csv">CSV 格式</option>
          <option value="mbox">MBOX 格式</option>
          <option value="eml">EML 文件</option>
        </select>
        
        <div style={{
          padding: '8px',
          backgroundColor: '#f0f8ff',
          borderRadius: '4px',
          fontSize: '11px',
          color: '#666',
          marginBottom: '12px'
        }}>
          将导出 {selectedEmails.length} 封邮件为 {exportFormat.toUpperCase()} 格式
        </div>
        
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
          <button
            onClick={() => setOperation(null)}
            style={{
              padding: '6px 12px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              backgroundColor: '#fff',
              cursor: 'pointer'
            }}
          >
            取消
          </button>
          <button
            onClick={handleExport}
            style={{
              padding: '6px 12px',
              border: '1px solid #607D8B',
              borderRadius: '4px',
              backgroundColor: '#607D8B',
              color: 'white',
              cursor: 'pointer'
            }}
          >
            导出
          </button>
        </div>
      </div>
    );
  }
  
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
  
  // 当没有选中邮件时，不显示组件
  if (selectedEmails.length === 0 && !isOpen) {
    // 如果没有邮件，完全不显示批量操作组件
    if (allEmails.length === 0) {
      return null;
    }
    
    return (
      <div className={`batch-operations ${className}`} style={style}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          padding: '12px 16px',
          backgroundColor: '#f8f9fa',
          borderRadius: '8px',
          fontSize: '14px',
          color: '#666'
        }}>
          <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={isAllSelected}
              ref={(input) => {
                if (input) input.indeterminate = isPartialSelected;
              }}
              onChange={isAllSelected ? handleSelectNone : handleSelectAll}
              style={{ marginRight: '8px' }}
            />
            {isAllSelected ? '取消全选' : isPartialSelected ? '全选' : '全选'}
          </label>
          <span>({allEmails.length} 封邮件)</span>
        </div>
      </div>
    );
  }
  
  return (
    <div className={`batch-operations ${className}`} style={style}>
      {/* 主操作栏 */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        padding: '12px 16px',
        backgroundColor: '#e3f2fd',
        borderRadius: '8px',
        border: '1px solid #2196F3'
      }}>
        {/* 选择控件 */}
        <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={isAllSelected}
            ref={(input) => {
              if (input) input.indeterminate = isPartialSelected;
            }}
            onChange={isAllSelected ? handleSelectNone : handleSelectAll}
            style={{ marginRight: '8px' }}
          />
          <span style={{ fontSize: '14px', color: '#1976d2', fontWeight: '500' }}>
            已选择 {selectedEmails.length} 封邮件
          </span>
        </label>
        
        {/* 快速操作按钮 */}
        <div style={{ display: 'flex', gap: '6px', flex: 1, justifyContent: 'flex-end' }}>
          {operations.slice(0, 4).map(op => (
            <button
              key={op.id}
              onClick={op.action || (() => setOperation(op.id))}
              style={{
                padding: '6px 12px',
                border: `1px solid ${op.color}`,
                borderRadius: '4px',
                backgroundColor: '#fff',
                color: op.color,
                cursor: 'pointer',
                fontSize: '12px',
                display: 'flex',
                alignItems: 'center',
                gap: '4px'
              }}
              title={op.label}
            >
              <span>{op.icon}</span>
              <span>{op.label}</span>
            </button>
          ))}
          
          {/* 更多操作按钮 */}
          <button
            onClick={() => setIsOpen(!isOpen)}
            style={{
              padding: '6px 12px',
              border: '1px solid #666',
              borderRadius: '4px',
              backgroundColor: isOpen ? '#f5f5f5' : '#fff',
              color: '#666',
              cursor: 'pointer',
              fontSize: '12px'
            }}
          >
            更多 {isOpen ? '▲' : '▼'}
          </button>
        </div>
      </div>
      
      {/* 展开的操作面板 */}
      {isOpen && (
        <div style={{
          marginTop: '8px',
          padding: '16px',
          backgroundColor: '#fff',
          border: '1px solid #e0e0e0',
          borderRadius: '8px',
          boxShadow: '0 2px 8px rgba(0,0,0,0.1)'
        }}>
          {operation ? (
            <div>
              <h4 style={{ margin: '0 0 12px 0', color: '#333', fontSize: '14px' }}>
                {operations.find(op => op.id === operation)?.label}
              </h4>
              {operations.find(op => op.id === operation)?.component?.()}
            </div>
          ) : (
            <div>
              <h4 style={{ margin: '0 0 12px 0', color: '#333', fontSize: '14px' }}>
                批量操作
              </h4>
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
                gap: '8px'
              }}>
                {operations.map(op => (
                  <button
                    key={op.id}
                    onClick={op.action || (() => setOperation(op.id))}
                    style={{
                      padding: '12px',
                      border: `1px solid ${op.color}`,
                      borderRadius: '6px',
                      backgroundColor: '#fff',
                      color: op.color,
                      cursor: 'pointer',
                      fontSize: '12px',
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      gap: '6px',
                      transition: 'all 0.2s ease'
                    }}
                    onMouseEnter={(e) => {
                      e.target.style.backgroundColor = op.color;
                      e.target.style.color = 'white';
                    }}
                    onMouseLeave={(e) => {
                      e.target.style.backgroundColor = '#fff';
                      e.target.style.color = op.color;
                    }}
                  >
                    <span style={{ fontSize: '16px' }}>{op.icon}</span>
                    <span>{op.label}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
      
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

export default BatchOperations;