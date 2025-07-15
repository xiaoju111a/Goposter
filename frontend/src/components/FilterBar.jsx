import React, { useCallback } from 'react';

const FilterBar = ({
  searchQuery,
  onSearchChange,
  filterConfig,
  onFilterChange,
  totalCount,
  filteredCount,
  className = '',
  style = {}
}) => {
  // 处理搜索
  const handleSearch = useCallback(() => {
    // 搜索逻辑已经通过onChange实时处理
  }, []);
  
  // 清除搜索
  const clearSearch = useCallback(() => {
    onSearchChange('');
    onFilterChange({});
  }, [onSearchChange, onFilterChange]);
  
  // 检查是否有搜索条件
  const hasSearch = searchQuery.trim() || (filterConfig && Object.keys(filterConfig).length > 0);
  
  return (
    <div className={`filter-bar ${className}`} style={style}>
      {/* 主搜索栏 */}
      <div className="search-section" style={{
        display: 'flex',
        gap: '12px',
        alignItems: 'center',
        marginBottom: '16px',
        padding: '16px',
        backgroundColor: '#f8f9fa',
        borderRadius: '8px'
      }}>
        <div style={{ position: 'relative', flex: 1 }}>
          <input
            type="text"
            placeholder="搜索邮箱名称..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            style={{
              width: '100%',
              padding: '12px 40px 12px 16px',
              border: '1px solid #ddd',
              borderRadius: '8px',
              fontSize: '14px',
              outline: 'none',
              backgroundColor: '#fff'
            }}
          />
          <button
            onClick={handleSearch}
            style={{
              position: 'absolute',
              right: '12px',
              top: '50%',
              transform: 'translateY(-50%)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: '16px',
              color: '#666'
            }}
          >
            🔍
          </button>
        </div>
        
        {hasSearch && (
          <button
            onClick={clearSearch}
            style={{
              padding: '12px 16px',
              border: '1px solid #f44336',
              borderRadius: '6px',
              backgroundColor: '#fff',
              color: '#f44336',
              cursor: 'pointer',
              fontSize: '14px',
              whiteSpace: 'nowrap'
            }}
          >
            清除搜索
          </button>
        )}
      </div>
      
      {/* 搜索结果统计 */}
      <div className="search-stats" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '0 16px',
        marginBottom: '16px',
        fontSize: '14px',
        color: '#666'
      }}>
        <div>
          {searchQuery ? 
            `找到 ${filteredCount} 个匹配的邮箱` : 
            `共 ${totalCount} 个邮箱`
          }
        </div>
        {searchQuery && (
          <div>
            搜索: "<strong>{searchQuery}</strong>"
          </div>
        )}
      </div>
      
    </div>
  );
};

export default FilterBar;