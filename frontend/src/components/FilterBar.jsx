import React, { useState, useCallback, useEffect, useMemo } from 'react';

const FilterBar = ({
  onFilterChange,
  initialFilters = {},
  mailboxes = [],
  showAdvanced = false,
  className = '',
  style = {}
}) => {
  const [filters, setFilters] = useState({
    search: '',
    mailbox: '',
    dateRange: { start: '', end: '' },
    sender: '',
    subject: '',
    hasAttachment: '',
    isRead: '',
    priority: '',
    size: { min: '', max: '' },
    tags: [],
    ...initialFilters
  });
  
  const [isAdvancedOpen, setIsAdvancedOpen] = useState(showAdvanced);
  const [searchMode, setSearchMode] = useState('simple'); // 'simple' | 'advanced'
  const [recentSearches, setRecentSearches] = useState([]);
  const [savedFilters, setSavedFilters] = useState([]);
  
  // 从localStorage加载保存的搜索和筛选
  useEffect(() => {
    try {
      const saved = localStorage.getItem('freeagent-mail-recent-searches');
      if (saved) {
        setRecentSearches(JSON.parse(saved));
      }
      
      const savedFiltersData = localStorage.getItem('freeagent-mail-saved-filters');
      if (savedFiltersData) {
        setSavedFilters(JSON.parse(savedFiltersData));
      }
    } catch (error) {
      console.error('加载保存的筛选条件失败:', error);
    }
  }, []);
  
  // 保存最近搜索
  const saveRecentSearch = useCallback((searchTerm) => {
    if (!searchTerm.trim()) return;
    
    const newRecentSearches = [
      searchTerm,
      ...recentSearches.filter(s => s !== searchTerm)
    ].slice(0, 10);
    
    setRecentSearches(newRecentSearches);
    
    try {
      localStorage.setItem('freeagent-mail-recent-searches', JSON.stringify(newRecentSearches));
    } catch (error) {
      console.error('保存最近搜索失败:', error);
    }
  }, [recentSearches]);
  
  // 更新筛选条件
  const updateFilter = useCallback((key, value) => {
    const newFilters = { ...filters, [key]: value };
    setFilters(newFilters);
    onFilterChange?.(newFilters);
  }, [filters, onFilterChange]);
  
  // 批量更新筛选条件
  const updateFilters = useCallback((newFilters) => {
    const updatedFilters = { ...filters, ...newFilters };
    setFilters(updatedFilters);
    onFilterChange?.(updatedFilters);
  }, [filters, onFilterChange]);
  
  // 清除所有筛选
  const clearAllFilters = useCallback(() => {
    const clearedFilters = {
      search: '',
      mailbox: '',
      dateRange: { start: '', end: '' },
      sender: '',
      subject: '',
      hasAttachment: '',
      isRead: '',
      priority: '',
      size: { min: '', max: '' },
      tags: []
    };
    setFilters(clearedFilters);
    onFilterChange?.(clearedFilters);
  }, [onFilterChange]);
  
  // 搜索处理
  const handleSearch = useCallback(() => {
    if (filters.search.trim()) {
      saveRecentSearch(filters.search.trim());
    }
    onFilterChange?.(filters);
  }, [filters, onFilterChange, saveRecentSearch]);
  
  // 快速搜索
  const handleQuickSearch = useCallback((searchTerm) => {
    updateFilter('search', searchTerm);
    saveRecentSearch(searchTerm);
  }, [updateFilter, saveRecentSearch]);
  
  // 保存当前筛选条件
  const saveCurrentFilters = useCallback(() => {
    const name = prompt('请输入筛选条件名称:');
    if (!name) return;
    
    const newSavedFilter = {
      id: Date.now(),
      name,
      filters: { ...filters },
      createdAt: new Date().toISOString()
    };
    
    const newSavedFilters = [...savedFilters, newSavedFilter];
    setSavedFilters(newSavedFilters);
    
    try {
      localStorage.setItem('freeagent-mail-saved-filters', JSON.stringify(newSavedFilters));
    } catch (error) {
      console.error('保存筛选条件失败:', error);
    }
  }, [filters, savedFilters]);
  
  // 应用保存的筛选条件
  const applySavedFilter = useCallback((savedFilter) => {
    setFilters(savedFilter.filters);
    onFilterChange?.(savedFilter.filters);
  }, [onFilterChange]);
  
  // 删除保存的筛选条件
  const deleteSavedFilter = useCallback((filterId) => {
    const newSavedFilters = savedFilters.filter(f => f.id !== filterId);
    setSavedFilters(newSavedFilters);
    
    try {
      localStorage.setItem('freeagent-mail-saved-filters', JSON.stringify(newSavedFilters));
    } catch (error) {
      console.error('删除保存的筛选条件失败:', error);
    }
  }, [savedFilters]);
  
  // 检查是否有活动筛选
  const hasActiveFilters = useMemo(() => {
    return filters.search || 
           filters.mailbox || 
           filters.dateRange.start || 
           filters.dateRange.end ||
           filters.sender || 
           filters.subject || 
           filters.hasAttachment || 
           filters.isRead || 
           filters.priority ||
           filters.size.min || 
           filters.size.max ||
           filters.tags.length > 0;
  }, [filters]);
  
  // 预设的快速筛选选项
  const quickFilters = [
    { label: '今天', value: 'today', action: () => {
      const today = new Date().toISOString().split('T')[0];
      updateFilter('dateRange', { start: today, end: today });
    }},
    { label: '本周', value: 'week', action: () => {
      const today = new Date();
      const weekStart = new Date(today.setDate(today.getDate() - today.getDay()));
      const weekEnd = new Date(today.setDate(today.getDate() - today.getDay() + 6));
      updateFilter('dateRange', { 
        start: weekStart.toISOString().split('T')[0],
        end: weekEnd.toISOString().split('T')[0]
      });
    }},
    { label: '有附件', value: 'hasAttachment', action: () => updateFilter('hasAttachment', 'true') },
    { label: '未读', value: 'unread', action: () => updateFilter('isRead', 'false') },
    { label: '重要', value: 'important', action: () => updateFilter('priority', 'high') }
  ];
  
  return (
    <div className={`filter-bar ${className}`} style={style}>
      {/* 主搜索栏 */}
      <div className="search-section" style={{
        display: 'flex',
        gap: '8px',
        alignItems: 'center',
        marginBottom: '12px',
        padding: '12px',
        backgroundColor: '#f8f9fa',
        borderRadius: '8px'
      }}>
        <div style={{ position: 'relative', flex: 1 }}>
          <input
            type="text"
            placeholder="搜索邮件内容、发件人、主题..."
            value={filters.search}
            onChange={(e) => updateFilter('search', e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            style={{
              width: '100%',
              padding: '10px 40px 10px 12px',
              border: '1px solid #ddd',
              borderRadius: '6px',
              fontSize: '14px',
              outline: 'none'
            }}
          />
          <button
            onClick={handleSearch}
            style={{
              position: 'absolute',
              right: '8px',
              top: '50%',
              transform: 'translateY(-50%)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: '16px'
            }}
          >
            🔍
          </button>
        </div>
        
        <button
          onClick={() => setIsAdvancedOpen(!isAdvancedOpen)}
          style={{
            padding: '10px 16px',
            border: '1px solid #ddd',
            borderRadius: '6px',
            backgroundColor: isAdvancedOpen ? '#e3f2fd' : '#fff',
            cursor: 'pointer',
            fontSize: '14px'
          }}
        >
          高级筛选 {isAdvancedOpen ? '▲' : '▼'}
        </button>
        
        {hasActiveFilters && (
          <button
            onClick={clearAllFilters}
            style={{
              padding: '10px 16px',
              border: '1px solid #f44336',
              borderRadius: '6px',
              backgroundColor: '#fff',
              color: '#f44336',
              cursor: 'pointer',
              fontSize: '14px'
            }}
          >
            清除筛选
          </button>
        )}
      </div>
      
      {/* 快速筛选按钮 */}
      <div className="quick-filters" style={{
        display: 'flex',
        gap: '6px',
        flexWrap: 'wrap',
        marginBottom: '12px',
        padding: '0 12px'
      }}>
        {quickFilters.map(filter => (
          <button
            key={filter.value}
            onClick={filter.action}
            style={{
              padding: '6px 12px',
              border: '1px solid #ddd',
              borderRadius: '12px',
              backgroundColor: '#fff',
              cursor: 'pointer',
              fontSize: '12px',
              color: '#666'
            }}
          >
            {filter.label}
          </button>
        ))}
      </div>
      
      {/* 最近搜索 */}
      {recentSearches.length > 0 && (
        <div className="recent-searches" style={{
          padding: '0 12px',
          marginBottom: '12px'
        }}>
          <div style={{ fontSize: '12px', color: '#666', marginBottom: '4px' }}>
            最近搜索:
          </div>
          <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
            {recentSearches.slice(0, 5).map((search, index) => (
              <button
                key={index}
                onClick={() => handleQuickSearch(search)}
                style={{
                  padding: '4px 8px',
                  border: '1px solid #e0e0e0',
                  borderRadius: '8px',
                  backgroundColor: '#f5f5f5',
                  cursor: 'pointer',
                  fontSize: '11px',
                  color: '#555'
                }}
              >
                {search}
              </button>
            ))}
          </div>
        </div>
      )}
      
      {/* 高级筛选面板 */}
      {isAdvancedOpen && (
        <div className="advanced-filters" style={{
          padding: '16px',
          border: '1px solid #e0e0e0',
          borderRadius: '8px',
          backgroundColor: '#fff',
          marginBottom: '12px'
        }}>
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
            gap: '16px'
          }}>
            {/* 邮箱筛选 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                邮箱
              </label>
              <select
                value={filters.mailbox}
                onChange={(e) => updateFilter('mailbox', e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
              >
                <option value="">所有邮箱</option>
                {mailboxes.map(mailbox => (
                  <option key={mailbox} value={mailbox}>
                    {mailbox}
                  </option>
                ))}
              </select>
            </div>
            
            {/* 发件人筛选 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                发件人
              </label>
              <input
                type="text"
                placeholder="发件人邮箱或姓名"
                value={filters.sender}
                onChange={(e) => updateFilter('sender', e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
              />
            </div>
            
            {/* 主题筛选 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                主题
              </label>
              <input
                type="text"
                placeholder="邮件主题"
                value={filters.subject}
                onChange={(e) => updateFilter('subject', e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
              />
            </div>
            
            {/* 日期范围 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                日期范围
              </label>
              <div style={{ display: 'flex', gap: '4px' }}>
                <input
                  type="date"
                  value={filters.dateRange.start}
                  onChange={(e) => updateFilter('dateRange', { ...filters.dateRange, start: e.target.value })}
                  style={{
                    flex: 1,
                    padding: '8px',
                    border: '1px solid #ddd',
                    borderRadius: '4px'
                  }}
                />
                <input
                  type="date"
                  value={filters.dateRange.end}
                  onChange={(e) => updateFilter('dateRange', { ...filters.dateRange, end: e.target.value })}
                  style={{
                    flex: 1,
                    padding: '8px',
                    border: '1px solid #ddd',
                    borderRadius: '4px'
                  }}
                />
              </div>
            </div>
            
            {/* 附件筛选 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                附件
              </label>
              <select
                value={filters.hasAttachment}
                onChange={(e) => updateFilter('hasAttachment', e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
              >
                <option value="">全部</option>
                <option value="true">有附件</option>
                <option value="false">无附件</option>
              </select>
            </div>
            
            {/* 读取状态 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                读取状态
              </label>
              <select
                value={filters.isRead}
                onChange={(e) => updateFilter('isRead', e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
              >
                <option value="">全部</option>
                <option value="true">已读</option>
                <option value="false">未读</option>
              </select>
            </div>
            
            {/* 优先级 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                优先级
              </label>
              <select
                value={filters.priority}
                onChange={(e) => updateFilter('priority', e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
              >
                <option value="">全部</option>
                <option value="high">高</option>
                <option value="normal">普通</option>
                <option value="low">低</option>
              </select>
            </div>
            
            {/* 邮件大小 */}
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                邮件大小 (KB)
              </label>
              <div style={{ display: 'flex', gap: '4px', alignItems: 'center' }}>
                <input
                  type="number"
                  placeholder="最小"
                  value={filters.size.min}
                  onChange={(e) => updateFilter('size', { ...filters.size, min: e.target.value })}
                  style={{
                    flex: 1,
                    padding: '8px',
                    border: '1px solid #ddd',
                    borderRadius: '4px'
                  }}
                />
                <span style={{ color: '#666' }}>-</span>
                <input
                  type="number"
                  placeholder="最大"
                  value={filters.size.max}
                  onChange={(e) => updateFilter('size', { ...filters.size, max: e.target.value })}
                  style={{
                    flex: 1,
                    padding: '8px',
                    border: '1px solid #ddd',
                    borderRadius: '4px'
                  }}
                />
              </div>
            </div>
          </div>
          
          {/* 高级筛选操作按钮 */}
          <div style={{
            display: 'flex',
            gap: '8px',
            justifyContent: 'flex-end',
            marginTop: '16px',
            paddingTop: '16px',
            borderTop: '1px solid #e0e0e0'
          }}>
            <button
              onClick={saveCurrentFilters}
              style={{
                padding: '8px 16px',
                border: '1px solid #4CAF50',
                borderRadius: '4px',
                backgroundColor: '#fff',
                color: '#4CAF50',
                cursor: 'pointer',
                fontSize: '14px'
              }}
            >
              保存筛选
            </button>
            <button
              onClick={clearAllFilters}
              style={{
                padding: '8px 16px',
                border: '1px solid #f44336',
                borderRadius: '4px',
                backgroundColor: '#fff',
                color: '#f44336',
                cursor: 'pointer',
                fontSize: '14px'
              }}
            >
              重置
            </button>
          </div>
        </div>
      )}
      
      {/* 保存的筛选条件 */}
      {savedFilters.length > 0 && (
        <div className="saved-filters" style={{
          padding: '12px',
          backgroundColor: '#f0f8ff',
          borderRadius: '8px'
        }}>
          <div style={{ fontSize: '12px', color: '#666', marginBottom: '8px' }}>
            保存的筛选条件:
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            {savedFilters.map(savedFilter => (
              <div
                key={savedFilter.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  padding: '6px 12px',
                  border: '1px solid #2196F3',
                  borderRadius: '12px',
                  backgroundColor: '#fff',
                  fontSize: '12px'
                }}
              >
                <button
                  onClick={() => applySavedFilter(savedFilter)}
                  style={{
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: '#2196F3'
                  }}
                >
                  {savedFilter.name}
                </button>
                <button
                  onClick={() => deleteSavedFilter(savedFilter.id)}
                  style={{
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: '#999',
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
    </div>
  );
};

export default FilterBar;