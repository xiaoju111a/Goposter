import React, { useState, useCallback, useEffect } from 'react';

const AttachmentViewer = ({ 
  attachments = [], 
  onDownload, 
  onPreview, 
  onDelete,
  showPreview = true,
  showDownload = true,
  showDelete = false,
  maxPreviewSize = 10 * 1024 * 1024, // 10MB
  className = '',
  style = {},
  layout = 'grid' // 'grid' | 'list'
}) => {
  const [previewFile, setPreviewFile] = useState(null);
  const [previewUrl, setPreviewUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  // 文件类型判断
  const getFileType = (file) => {
    const fileName = file.name || file.filename || '';
    const mimeType = file.type || file.mimeType || '';
    const extension = fileName.split('.').pop()?.toLowerCase() || '';
    
    if (mimeType.startsWith('image/') || ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'].includes(extension)) {
      return 'image';
    }
    if (mimeType.startsWith('video/') || ['mp4', 'avi', 'mov', 'wmv', 'flv', 'webm'].includes(extension)) {
      return 'video';
    }
    if (mimeType.startsWith('audio/') || ['mp3', 'wav', 'ogg', 'aac', 'flac'].includes(extension)) {
      return 'audio';
    }
    if (mimeType === 'application/pdf' || extension === 'pdf') {
      return 'pdf';
    }
    if (['txt', 'md', 'json', 'xml', 'csv'].includes(extension) || mimeType.startsWith('text/')) {
      return 'text';
    }
    if (['doc', 'docx'].includes(extension) || mimeType.includes('word')) {
      return 'document';
    }
    if (['xls', 'xlsx'].includes(extension) || mimeType.includes('spreadsheet')) {
      return 'spreadsheet';
    }
    if (['ppt', 'pptx'].includes(extension) || mimeType.includes('presentation')) {
      return 'presentation';
    }
    if (['zip', 'rar', '7z', 'tar', 'gz'].includes(extension)) {
      return 'archive';
    }
    
    return 'unknown';
  };
  
  // 获取文件图标
  const getFileIcon = (fileType) => {
    const icons = {
      image: '🖼️',
      video: '🎥',
      audio: '🎵',
      pdf: '📄',
      text: '📝',
      document: '📄',
      spreadsheet: '📊',
      presentation: '📈',
      archive: '🗜️',
      unknown: '📁'
    };
    
    return icons[fileType] || icons.unknown;
  };
  
  // 格式化文件大小
  const formatFileSize = (bytes) => {
    if (!bytes) return '未知大小';
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };
  
  // 判断是否可以预览
  const canPreview = useCallback((file) => {
    if (!showPreview) return false;
    
    const fileType = getFileType(file);
    const fileSize = file.size || 0;
    
    // 检查文件大小限制
    if (fileSize > maxPreviewSize) return false;
    
    // 支持预览的文件类型
    return ['image', 'video', 'audio', 'pdf', 'text'].includes(fileType);
  }, [showPreview, maxPreviewSize]);
  
  // 处理文件预览
  const handlePreview = useCallback(async (file) => {
    if (!canPreview(file)) return;
    
    setLoading(true);
    setError('');
    
    try {
      let url = '';
      
      if (file.url) {
        // 已有URL
        url = file.url;
      } else if (file.content) {
        // 有content内容
        const blob = new Blob([file.content], { type: file.type });
        url = URL.createObjectURL(blob);
      } else if (onPreview) {
        // 通过回调获取
        url = await onPreview(file);
      }
      
      if (url) {
        setPreviewFile(file);
        setPreviewUrl(url);
      } else {
        setError('无法加载文件预览');
      }
    } catch (err) {
      console.error('预览失败:', err);
      setError('预览加载失败');
    } finally {
      setLoading(false);
    }
  }, [canPreview, onPreview]);
  
  // 关闭预览
  const closePreview = useCallback(() => {
    setPreviewFile(null);
    if (previewUrl && previewUrl.startsWith('blob:')) {
      URL.revokeObjectURL(previewUrl);
    }
    setPreviewUrl('');
    setError('');
  }, [previewUrl]);
  
  // 处理下载
  const handleDownload = useCallback((file) => {
    if (onDownload) {
      onDownload(file);
    } else if (file.url) {
      // 直接下载
      const link = document.createElement('a');
      link.href = file.url;
      link.download = file.name || 'download';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
  }, [onDownload]);
  
  // 处理删除
  const handleDelete = useCallback((file) => {
    if (onDelete) {
      onDelete(file);
    }
  }, [onDelete]);
  
  // 渲染预览内容
  const renderPreviewContent = () => {
    if (!previewFile || !previewUrl) return null;
    
    const fileType = getFileType(previewFile);
    
    switch (fileType) {
      case 'image':
        return (
          <img
            src={previewUrl}
            alt={previewFile.name}
            style={{
              maxWidth: '100%',
              maxHeight: '80vh',
              objectFit: 'contain'
            }}
          />
        );
        
      case 'video':
        return (
          <video
            src={previewUrl}
            controls
            style={{
              maxWidth: '100%',
              maxHeight: '80vh'
            }}
          />
        );
        
      case 'audio':
        return (
          <audio
            src={previewUrl}
            controls
            style={{ width: '100%' }}
          />
        );
        
      case 'pdf':
        return (
          <iframe
            src={previewUrl}
            style={{
              width: '100%',
              height: '80vh',
              border: 'none'
            }}
          />
        );
        
      case 'text':
        return (
          <div
            style={{
              width: '100%',
              height: '80vh',
              overflow: 'auto',
              padding: '20px',
              backgroundColor: '#f5f5f5',
              border: '1px solid #ddd',
              borderRadius: '4px'
            }}
          >
            <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
              {/* 这里需要加载文本内容 */}
              正在加载文本内容...
            </pre>
          </div>
        );
        
      default:
        return (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <div style={{ fontSize: '48px', marginBottom: '20px' }}>
              {getFileIcon(fileType)}
            </div>
            <div style={{ fontSize: '16px', color: '#666' }}>
              暂不支持预览此文件类型
            </div>
          </div>
        );
    }
  };
  
  // 清理预览URL
  useEffect(() => {
    return () => {
      if (previewUrl && previewUrl.startsWith('blob:')) {
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [previewUrl]);
  
  if (!attachments.length) {
    return (
      <div className={`attachment-viewer empty ${className}`} style={style}>
        <div style={{
          textAlign: 'center',
          padding: '40px',
          color: '#999',
          fontSize: '14px'
        }}>
          暂无附件
        </div>
      </div>
    );
  }
  
  return (
    <div className={`attachment-viewer ${className}`} style={style}>
      {/* 附件列表 */}
      <div className={`attachments-${layout}`} style={{
        display: layout === 'grid' ? 'grid' : 'flex',
        gridTemplateColumns: layout === 'grid' ? 'repeat(auto-fill, minmax(200px, 1fr))' : undefined,
        flexDirection: layout === 'list' ? 'column' : undefined,
        gap: '12px',
        padding: '12px'
      }}>
        {attachments.map((file, index) => {
          const fileType = getFileType(file);
          const canPreviewFile = canPreview(file);
          
          return (
            <div
              key={file.id || index}
              className="attachment-item"
              style={{
                border: '1px solid #e0e0e0',
                borderRadius: '8px',
                padding: '12px',
                backgroundColor: '#fff',
                transition: 'box-shadow 0.2s ease',
                cursor: canPreviewFile ? 'pointer' : 'default'
              }}
              onClick={() => canPreviewFile && handlePreview(file)}
              onMouseEnter={(e) => {
                e.target.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)';
              }}
              onMouseLeave={(e) => {
                e.target.style.boxShadow = 'none';
              }}
            >
              {/* 文件图标和信息 */}
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                marginBottom: '8px'
              }}>
                <span style={{ fontSize: '24px' }}>
                  {getFileIcon(fileType)}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{
                    fontSize: '14px',
                    fontWeight: '500',
                    color: '#333',
                    wordBreak: 'break-all',
                    lineHeight: '1.2'
                  }}>
                    {file.name || file.filename || '未知文件'}
                  </div>
                  <div style={{
                    fontSize: '12px',
                    color: '#999',
                    marginTop: '2px'
                  }}>
                    {formatFileSize(file.size)}
                  </div>
                </div>
              </div>
              
              {/* 操作按钮 */}
              <div style={{
                display: 'flex',
                gap: '4px',
                justifyContent: 'flex-end'
              }}>
                {canPreviewFile && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handlePreview(file);
                    }}
                    style={{
                      padding: '4px 8px',
                      border: '1px solid #4CAF50',
                      borderRadius: '4px',
                      backgroundColor: '#fff',
                      color: '#4CAF50',
                      cursor: 'pointer',
                      fontSize: '12px'
                    }}
                    title="预览"
                  >
                    👁️ 预览
                  </button>
                )}
                
                {showDownload && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDownload(file);
                    }}
                    style={{
                      padding: '4px 8px',
                      border: '1px solid #2196F3',
                      borderRadius: '4px',
                      backgroundColor: '#fff',
                      color: '#2196F3',
                      cursor: 'pointer',
                      fontSize: '12px'
                    }}
                    title="下载"
                  >
                    ⬇️ 下载
                  </button>
                )}
                
                {showDelete && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      if (confirm('确定要删除这个附件吗？')) {
                        handleDelete(file);
                      }
                    }}
                    style={{
                      padding: '4px 8px',
                      border: '1px solid #f44336',
                      borderRadius: '4px',
                      backgroundColor: '#fff',
                      color: '#f44336',
                      cursor: 'pointer',
                      fontSize: '12px'
                    }}
                    title="删除"
                  >
                    🗑️ 删除
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>
      
      {/* 预览模态框 */}
      {previewFile && (
        <div
          className="preview-modal"
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.8)',
            zIndex: 1000,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '20px'
          }}
          onClick={closePreview}
        >
          <div
            className="preview-content"
            style={{
              position: 'relative',
              backgroundColor: '#fff',
              borderRadius: '8px',
              padding: '20px',
              maxWidth: '90vw',
              maxHeight: '90vh',
              overflow: 'auto'
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* 预览头部 */}
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '16px',
              paddingBottom: '8px',
              borderBottom: '1px solid #e0e0e0'
            }}>
              <div>
                <h3 style={{ margin: 0, fontSize: '16px', color: '#333' }}>
                  {previewFile.name || '文件预览'}
                </h3>
                <div style={{ fontSize: '12px', color: '#999', marginTop: '2px' }}>
                  {formatFileSize(previewFile.size)}
                </div>
              </div>
              <button
                onClick={closePreview}
                style={{
                  background: 'none',
                  border: 'none',
                  fontSize: '24px',
                  cursor: 'pointer',
                  color: '#999',
                  padding: '0',
                  lineHeight: 1
                }}
              >
                ×
              </button>
            </div>
            
            {/* 预览内容 */}
            {loading ? (
              <div style={{ textAlign: 'center', padding: '40px' }}>
                <div>正在加载...</div>
              </div>
            ) : error ? (
              <div style={{ textAlign: 'center', padding: '40px', color: '#f44336' }}>
                <div>❌ {error}</div>
              </div>
            ) : (
              renderPreviewContent()
            )}
            
            {/* 预览底部操作 */}
            <div style={{
              display: 'flex',
              justifyContent: 'center',
              gap: '12px',
              marginTop: '16px',
              paddingTop: '8px',
              borderTop: '1px solid #e0e0e0'
            }}>
              {showDownload && (
                <button
                  onClick={() => handleDownload(previewFile)}
                  style={{
                    padding: '8px 16px',
                    border: '1px solid #2196F3',
                    borderRadius: '4px',
                    backgroundColor: '#2196F3',
                    color: '#fff',
                    cursor: 'pointer'
                  }}
                >
                  下载文件
                </button>
              )}
              <button
                onClick={closePreview}
                style={{
                  padding: '8px 16px',
                  border: '1px solid #ccc',
                  borderRadius: '4px',
                  backgroundColor: '#fff',
                  color: '#333',
                  cursor: 'pointer'
                }}
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AttachmentViewer;