import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import './VirtualList.css';

const VirtualList = ({ 
  items = [], 
  itemHeight = 80, 
  containerHeight = 400, 
  renderItem,
  onScroll,
  onLoadMore,
  loading = false,
  hasMore = true,
  overscan = 5,
  loadMoreThreshold = 0.8,
  className = '',
  estimatedItemSize = 80,
  dynamic = false
}) => {
  const [scrollTop, setScrollTop] = useState(0);
  const [containerSize, setContainerSize] = useState({ width: 0, height: containerHeight });
  const [itemSizes, setItemSizes] = useState(new Map());
  const [isScrolling, setIsScrolling] = useState(false);
  
  const containerRef = useRef(null);
  const scrollElementRef = useRef(null);
  const scrollingTimeoutRef = useRef(null);
  const itemRefs = useRef(new Map());
  
  // 计算可视区域
  const visibleRange = useMemo(() => {
    if (!items.length) return { start: 0, end: 0 };
    
    let start = 0;
    let end = 0;
    let currentHeight = 0;
    
    if (dynamic) {
      // 动态高度计算
      for (let i = 0; i < items.length; i++) {
        const itemSize = itemSizes.get(i) || estimatedItemSize;
        if (currentHeight + itemSize > scrollTop) {
          start = i;
          break;
        }
        currentHeight += itemSize;
      }
      
      currentHeight = 0;
      for (let i = 0; i < items.length; i++) {
        const itemSize = itemSizes.get(i) || estimatedItemSize;
        currentHeight += itemSize;
        if (currentHeight >= scrollTop + containerSize.height) {
          end = i;
          break;
        }
      }
    } else {
      // 固定高度计算
      start = Math.floor(scrollTop / itemHeight);
      end = Math.min(
        start + Math.ceil(containerSize.height / itemHeight) + overscan,
        items.length
      );
    }
    
    start = Math.max(0, start - overscan);
    end = Math.min(items.length, end + overscan);
    
    return { start, end };
  }, [items.length, scrollTop, containerSize.height, itemHeight, itemSizes, dynamic, estimatedItemSize, overscan]);
  
  // 计算总高度
  const totalHeight = useMemo(() => {
    if (dynamic) {
      return items.reduce((total, _, index) => {
        return total + (itemSizes.get(index) || estimatedItemSize);
      }, 0);
    }
    return items.length * itemHeight;
  }, [items.length, itemHeight, itemSizes, dynamic, estimatedItemSize]);
  
  // 计算偏移量
  const offsetY = useMemo(() => {
    if (dynamic) {
      let offset = 0;
      for (let i = 0; i < visibleRange.start; i++) {
        offset += itemSizes.get(i) || estimatedItemSize;
      }
      return offset;
    }
    return visibleRange.start * itemHeight;
  }, [visibleRange.start, itemHeight, itemSizes, dynamic, estimatedItemSize]);
  
  // 处理滚动事件
  const handleScroll = useCallback((e) => {
    const newScrollTop = e.currentTarget.scrollTop;
    setScrollTop(newScrollTop);
    setIsScrolling(true);
    
    // 清除之前的定时器
    if (scrollingTimeoutRef.current) {
      clearTimeout(scrollingTimeoutRef.current);
    }
    
    // 设置新的定时器
    scrollingTimeoutRef.current = setTimeout(() => {
      setIsScrolling(false);
    }, 150);
    
    // 调用外部滚动回调
    if (onScroll) {
      onScroll(e);
    }
    
    // 检查是否需要加载更多
    if (hasMore && !loading && onLoadMore) {
      const scrollHeight = e.currentTarget.scrollHeight;
      const clientHeight = e.currentTarget.clientHeight;
      const scrollPercent = (newScrollTop + clientHeight) / scrollHeight;
      
      if (scrollPercent >= loadMoreThreshold) {
        onLoadMore();
      }
    }
  }, [onScroll, onLoadMore, hasMore, loading, loadMoreThreshold]);
  
  // 观察器回调
  const resizeObserverCallback = useCallback((entries) => {
    if (dynamic) {
      const newSizes = new Map(itemSizes);
      let changed = false;
      
      entries.forEach(entry => {
        const index = parseInt(entry.target.dataset.index);
        const newSize = entry.contentRect.height;
        if (newSizes.get(index) !== newSize) {
          newSizes.set(index, newSize);
          changed = true;
        }
      });
      
      if (changed) {
        setItemSizes(newSizes);
      }
    }
  }, [itemSizes, dynamic]);
  
  // 设置观察器
  useEffect(() => {
    if (!dynamic) return;
    
    const observer = new ResizeObserver(resizeObserverCallback);
    
    itemRefs.current.forEach((ref, index) => {
      if (ref) {
        observer.observe(ref);
      }
    });
    
    return () => {
      observer.disconnect();
    };
  }, [resizeObserverCallback, dynamic, visibleRange]);
  
  // 容器大小监听
  useEffect(() => {
    const handleResize = () => {
      if (containerRef.current && typeof containerRef.current.getBoundingClientRect === 'function') {
        const { width, height } = containerRef.current.getBoundingClientRect();
        setContainerSize({ width, height });
      }
    };
    
    // 延迟执行以确保DOM已经挂载
    const timer = setTimeout(handleResize, 0);
    window.addEventListener('resize', handleResize);
    
    return () => {
      clearTimeout(timer);
      window.removeEventListener('resize', handleResize);
    };
  }, []);
  
  // 清理定时器
  useEffect(() => {
    return () => {
      if (scrollingTimeoutRef.current) {
        clearTimeout(scrollingTimeoutRef.current);
      }
    };
  }, []);
  
  // 渲染可见项
  const renderVisibleItems = () => {
    const visibleItems = [];
    
    for (let i = visibleRange.start; i < visibleRange.end; i++) {
      const item = items[i];
      if (!item) continue;
      
      const itemStyle = {
        position: 'absolute',
        top: dynamic ? 
          items.slice(0, i).reduce((acc, _, idx) => acc + (itemSizes.get(idx) || estimatedItemSize), 0) :
          i * itemHeight,
        left: 0,
        right: 0,
        height: dynamic ? (itemSizes.get(i) || estimatedItemSize) : itemHeight,
        zIndex: 1,
      };
      
      visibleItems.push(
        <div
          key={item.id || i}
          style={itemStyle}
          data-index={i}
          className="virtual-list-item"
          ref={el => {
            if (dynamic) {
              itemRefs.current.set(i, el);
            }
          }}
        >
          {renderItem(item, i, {
            isScrolling,
            isVisible: true,
            index: i
          })}
        </div>
      );
    }
    
    return visibleItems;
  };
  
  // 滚动到指定位置
  const scrollToIndex = useCallback((index, align = 'start') => {
    if (!scrollElementRef.current) return;
    
    let scrollTop = 0;
    
    if (dynamic) {
      for (let i = 0; i < index; i++) {
        scrollTop += itemSizes.get(i) || estimatedItemSize;
      }
    } else {
      scrollTop = index * itemHeight;
    }
    
    if (align === 'center') {
      scrollTop -= containerSize.height / 2;
    } else if (align === 'end') {
      scrollTop -= containerSize.height;
    }
    
    scrollElementRef.current.scrollTop = Math.max(0, scrollTop);
  }, [itemHeight, itemSizes, containerSize.height, dynamic, estimatedItemSize]);
  
  // 暴露给父组件的方法
  React.useImperativeHandle(containerRef, () => ({
    scrollToIndex,
    scrollToTop: () => scrollElementRef.current?.scrollTo({ top: 0, behavior: 'smooth' }),
    scrollToBottom: () => scrollElementRef.current?.scrollTo({ top: totalHeight, behavior: 'smooth' }),
    getVisibleRange: () => visibleRange,
    getTotalHeight: () => totalHeight,
    refresh: () => {
      setScrollTop(0);
      setItemSizes(new Map());
    }
  }));
  
  return (
    <div 
      ref={containerRef}
      className={`virtual-list ${className}`}
      style={{ height: containerHeight }}
    >
      <div
        ref={scrollElementRef}
        className="virtual-list-scrollable"
        style={{ height: '100%', overflow: 'auto' }}
        onScroll={handleScroll}
      >
        <div
          className="virtual-list-inner"
          style={{
            position: 'relative',
            height: totalHeight,
            overflow: 'hidden',
          }}
        >
          {/* 渲染可见项 */}
          {renderVisibleItems()}
          
          {/* 占位符 */}
          <div
            className="virtual-list-spacer"
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              height: offsetY,
              backgroundColor: 'transparent',
              pointerEvents: 'none',
            }}
          />
          
          {/* 加载更多指示器 */}
          {loading && (
            <div
              className="virtual-list-loading"
              style={{
                position: 'absolute',
                bottom: 0,
                left: 0,
                right: 0,
                height: 50,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                backgroundColor: 'rgba(255, 255, 255, 0.9)',
                borderTop: '1px solid #e0e0e0',
              }}
            >
              <div className="loading-spinner">
                <div className="spinner"></div>
                <span>Loading more emails...</span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default VirtualList;