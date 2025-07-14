import React, { useState, useRef, useCallback, useEffect } from 'react';

const EmailEditor = ({
  value = '',
  onChange,
  placeholder = '请输入邮件内容...',
  disabled = false,
  className = '',
  style = {},
  autoFocus = false,
  maxLength,
  enableFormatting = true,
  enableAttachments = true,
  templates = [],
  onAttachmentAdd,
  onTemplateSelect
}) => {
  const [content, setContent] = useState(value);
  const [selectedRange, setSelectedRange] = useState(null);
  const [isFormatting, setIsFormatting] = useState(false);
  const [attachments, setAttachments] = useState([]);
  const [showTemplates, setShowTemplates] = useState(false);
  const [textStats, setTextStats] = useState({ chars: 0, words: 0 });
  
  const editorRef = useRef(null);
  const fileInputRef = useRef(null);
  
  // 计算文本统计
  const calculateStats = useCallback((text) => {
    const chars = text.length;
    const words = text.trim() ? text.trim().split(/\s+/).length : 0;
    setTextStats({ chars, words });
  }, []);
  
  // 内容变化处理
  const handleContentChange = useCallback((newContent) => {
    setContent(newContent);
    calculateStats(newContent);
    onChange?.(newContent);
  }, [onChange, calculateStats]);
  
  // 保存选择范围
  const saveSelection = useCallback(() => {
    if (editorRef.current) {
      const selection = window.getSelection();
      if (selection.rangeCount > 0) {
        setSelectedRange(selection.getRangeAt(0).cloneRange());
      }
    }
  }, []);
  
  // 恢复选择范围
  const restoreSelection = useCallback(() => {
    if (selectedRange && editorRef.current) {
      const selection = window.getSelection();
      selection.removeAllRanges();
      selection.addRange(selectedRange);
    }
  }, [selectedRange]);
  
  // 执行格式化命令
  const execCommand = useCallback((command, value = null) => {
    if (!enableFormatting || disabled) return;
    
    setIsFormatting(true);
    restoreSelection();
    
    try {
      document.execCommand(command, false, value);
      const newContent = editorRef.current.innerHTML;
      handleContentChange(newContent);
    } catch (error) {
      console.error('格式化命令执行失败:', error);
    } finally {
      setIsFormatting(false);
    }
  }, [enableFormatting, disabled, restoreSelection, handleContentChange]);
  
  // 格式化按钮
  const formatButtons = [
    { command: 'bold', icon: '𝐁', title: '粗体' },
    { command: 'italic', icon: '𝐼', title: '斜体' },
    { command: 'underline', icon: '𝐔', title: '下划线' },
    { command: 'strikeThrough', icon: '𝐒', title: '删除线' },
    { type: 'separator' },
    { command: 'justifyLeft', icon: '≡', title: '左对齐' },
    { command: 'justifyCenter', icon: '≣', title: '居中对齐' },
    { command: 'justifyRight', icon: '≡', title: '右对齐' },
    { type: 'separator' },
    { command: 'insertUnorderedList', icon: '•', title: '无序列表' },
    { command: 'insertOrderedList', icon: '1.', title: '有序列表' },
    { type: 'separator' },
    { command: 'createLink', icon: '🔗', title: '插入链接' },
    { command: 'unlink', icon: '🔗', title: '移除链接' }
  ];
  
  // 字体大小选项
  const fontSizes = [
    { value: '1', label: '极小' },
    { value: '2', label: '小' },
    { value: '3', label: '正常' },
    { value: '4', label: '大' },
    { value: '5', label: '较大' },
    { value: '6', label: '很大' },
    { value: '7', label: '极大' }
  ];
  
  // 字体颜色选项
  const colors = [
    '#000000', '#333333', '#666666', '#999999',
    '#FF0000', '#FF6600', '#FFCC00', '#00FF00',
    '#0066CC', '#0000FF', '#6600CC', '#CC0066'
  ];
  
  // 处理输入事件
  const handleInput = useCallback((e) => {
    if (isFormatting) return;
    
    const newContent = e.target.innerHTML;
    handleContentChange(newContent);
  }, [isFormatting, handleContentChange]);
  
  // 处理粘贴事件
  const handlePaste = useCallback((e) => {
    if (disabled) return;
    
    e.preventDefault();
    const text = e.clipboardData.getData('text/plain');
    
    // 清理粘贴的内容
    const cleanText = text.replace(/[\r\n]/g, '<br>');
    document.execCommand('insertHTML', false, cleanText);
  }, [disabled]);
  
  // 处理键盘事件
  const handleKeyDown = useCallback((e) => {
    if (disabled) return;
    
    // 快捷键处理
    if (e.ctrlKey || e.metaKey) {
      switch (e.key) {
        case 'b':
          e.preventDefault();
          execCommand('bold');
          break;
        case 'i':
          e.preventDefault();
          execCommand('italic');
          break;
        case 'u':
          e.preventDefault();
          execCommand('underline');
          break;
        case 'z':
          e.preventDefault();
          execCommand('undo');
          break;
        case 'y':
          e.preventDefault();
          execCommand('redo');
          break;
      }
    }
  }, [disabled, execCommand]);
  
  // 插入链接
  const insertLink = useCallback(() => {
    const url = prompt('请输入链接地址:');
    if (url) {
      execCommand('createLink', url);
    }
  }, [execCommand]);
  
  // 插入图片
  const insertImage = useCallback(() => {
    const url = prompt('请输入图片地址:');
    if (url) {
      execCommand('insertImage', url);
    }
  }, [execCommand]);
  
  // 处理附件上传
  const handleFileSelect = useCallback((e) => {
    const files = Array.from(e.target.files);
    
    files.forEach(file => {
      const attachment = {
        id: Date.now() + Math.random(),
        file,
        name: file.name,
        size: file.size,
        type: file.type,
        url: URL.createObjectURL(file)
      };
      
      setAttachments(prev => [...prev, attachment]);
      onAttachmentAdd?.(attachment);
    });
    
    // 清空文件输入
    e.target.value = '';
  }, [onAttachmentAdd]);
  
  // 移除附件
  const removeAttachment = useCallback((id) => {
    setAttachments(prev => {
      const attachment = prev.find(att => att.id === id);
      if (attachment?.url) {
        URL.revokeObjectURL(attachment.url);
      }
      return prev.filter(att => att.id !== id);
    });
  }, []);
  
  // 选择模板
  const handleTemplateSelect = useCallback((template) => {
    if (template.content) {
      setContent(template.content);
      handleContentChange(template.content);
      editorRef.current.innerHTML = template.content;
    }
    setShowTemplates(false);
    onTemplateSelect?.(template);
  }, [handleContentChange, onTemplateSelect]);
  
  // 格式化文件大小
  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };
  
  // 初始化内容
  useEffect(() => {
    if (editorRef.current && value !== content) {
      editorRef.current.innerHTML = value;
      setContent(value);
      calculateStats(value);
    }
  }, [value, content, calculateStats]);
  
  // 自动聚焦
  useEffect(() => {
    if (autoFocus && editorRef.current) {
      editorRef.current.focus();
    }
  }, [autoFocus]);
  
  const editorStyle = {
    minHeight: '200px',
    maxHeight: '400px',
    padding: '12px',
    border: '1px solid #ddd',
    borderRadius: '4px',
    overflow: 'auto',
    outline: 'none',
    lineHeight: '1.5',
    backgroundColor: disabled ? '#f5f5f5' : '#fff',
    color: disabled ? '#999' : '#333',
    cursor: disabled ? 'not-allowed' : 'text',
    ...style
  };
  
  return (
    <div className={`email-editor ${className}`}>
      {/* 工具栏 */}
      {enableFormatting && !disabled && (
        <div className="editor-toolbar" style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '4px',
          padding: '8px',
          borderBottom: '1px solid #ddd',
          backgroundColor: '#f9f9f9'
        }}>
          {/* 格式化按钮 */}
          {formatButtons.map((button, index) => {
            if (button.type === 'separator') {
              return (
                <div
                  key={index}
                  style={{
                    width: '1px',
                    height: '24px',
                    backgroundColor: '#ddd',
                    margin: '0 4px'
                  }}
                />
              );
            }
            
            return (
              <button
                key={button.command}
                type="button"
                title={button.title}
                onClick={() => {
                  if (button.command === 'createLink') {
                    insertLink();
                  } else {
                    execCommand(button.command);
                  }
                }}
                style={{
                  padding: '4px 8px',
                  border: '1px solid #ddd',
                  borderRadius: '3px',
                  backgroundColor: '#fff',
                  cursor: 'pointer',
                  fontSize: '14px',
                  minWidth: '28px',
                  height: '28px'
                }}
                onMouseDown={(e) => e.preventDefault()}
              >
                {button.icon}
              </button>
            );
          })}
          
          {/* 字体大小 */}
          <select
            onChange={(e) => execCommand('fontSize', e.target.value)}
            style={{
              padding: '4px',
              border: '1px solid #ddd',
              borderRadius: '3px',
              backgroundColor: '#fff'
            }}
            onMouseDown={(e) => e.preventDefault()}
          >
            <option value="">字体大小</option>
            {fontSizes.map(size => (
              <option key={size.value} value={size.value}>
                {size.label}
              </option>
            ))}
          </select>
          
          {/* 字体颜色 */}
          <div style={{ display: 'flex', gap: '2px', alignItems: 'center' }}>
            <span style={{ fontSize: '12px', color: '#666' }}>颜色:</span>
            {colors.slice(0, 4).map(color => (
              <button
                key={color}
                type="button"
                onClick={() => execCommand('foreColor', color)}
                style={{
                  width: '20px',
                  height: '20px',
                  backgroundColor: color,
                  border: '1px solid #ddd',
                  borderRadius: '2px',
                  cursor: 'pointer'
                }}
                onMouseDown={(e) => e.preventDefault()}
              />
            ))}
          </div>
          
          {/* 其他工具 */}
          <button
            type="button"
            title="插入图片"
            onClick={insertImage}
            style={{
              padding: '4px 8px',
              border: '1px solid #ddd',
              borderRadius: '3px',
              backgroundColor: '#fff',
              cursor: 'pointer'
            }}
            onMouseDown={(e) => e.preventDefault()}
          >
            🖼️
          </button>
          
          {/* 模板按钮 */}
          {templates.length > 0 && (
            <button
              type="button"
              title="选择模板"
              onClick={() => setShowTemplates(!showTemplates)}
              style={{
                padding: '4px 8px',
                border: '1px solid #ddd',
                borderRadius: '3px',
                backgroundColor: showTemplates ? '#e3f2fd' : '#fff',
                cursor: 'pointer'
              }}
              onMouseDown={(e) => e.preventDefault()}
            >
              📄
            </button>
          )}
          
          {/* 附件按钮 */}
          {enableAttachments && (
            <button
              type="button"
              title="添加附件"
              onClick={() => fileInputRef.current?.click()}
              style={{
                padding: '4px 8px',
                border: '1px solid #ddd',
                borderRadius: '3px',
                backgroundColor: '#fff',
                cursor: 'pointer'
              }}
              onMouseDown={(e) => e.preventDefault()}
            >
              📎
            </button>
          )}
        </div>
      )}
      
      {/* 模板选择器 */}
      {showTemplates && templates.length > 0 && (
        <div className="template-selector" style={{
          padding: '8px',
          borderBottom: '1px solid #ddd',
          backgroundColor: '#f0f8ff',
          maxHeight: '150px',
          overflowY: 'auto'
        }}>
          <div style={{ fontSize: '12px', color: '#666', marginBottom: '4px' }}>
            选择模板:
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
            {templates.map((template, index) => (
              <button
                key={index}
                onClick={() => handleTemplateSelect(template)}
                style={{
                  padding: '6px 12px',
                  border: '1px solid #4CAF50',
                  borderRadius: '12px',
                  backgroundColor: '#fff',
                  color: '#4CAF50',
                  cursor: 'pointer',
                  fontSize: '12px'
                }}
              >
                {template.name}
              </button>
            ))}
          </div>
        </div>
      )}
      
      {/* 编辑器主体 */}
      <div
        ref={editorRef}
        contentEditable={!disabled}
        suppressContentEditableWarning={true}
        onInput={handleInput}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        onBlur={saveSelection}
        onMouseUp={saveSelection}
        style={editorStyle}
        data-placeholder={placeholder}
      />
      
      {/* 附件列表 */}
      {attachments.length > 0 && (
        <div className="attachments-list" style={{
          padding: '8px',
          borderTop: '1px solid #ddd',
          backgroundColor: '#f9f9f9'
        }}>
          <div style={{ fontSize: '12px', color: '#666', marginBottom: '4px' }}>
            附件 ({attachments.length}):
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
            {attachments.map(attachment => (
              <div
                key={attachment.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  padding: '4px 8px',
                  backgroundColor: '#fff',
                  border: '1px solid #ddd',
                  borderRadius: '12px',
                  fontSize: '12px'
                }}
              >
                <span>📎</span>
                <span>{attachment.name}</span>
                <span style={{ color: '#999' }}>
                  ({formatFileSize(attachment.size)})
                </span>
                <button
                  onClick={() => removeAttachment(attachment.id)}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: '#999',
                    cursor: 'pointer',
                    padding: '0 2px'
                  }}
                >
                  ×
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
      
      {/* 状态栏 */}
      <div className="editor-status" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '4px 8px',
        borderTop: '1px solid #ddd',
        backgroundColor: '#f9f9f9',
        fontSize: '12px',
        color: '#666'
      }}>
        <div>
          字符: {textStats.chars}
          {maxLength && ` / ${maxLength}`}
          {' | '}
          单词: {textStats.words}
        </div>
        <div>
          {attachments.length > 0 && `附件: ${attachments.length}`}
        </div>
      </div>
      
      {/* 隐藏的文件输入 */}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        style={{ display: 'none' }}
        onChange={handleFileSelect}
        accept="*/*"
      />
    </div>
  );
};

export default EmailEditor;