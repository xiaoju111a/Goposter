import React, { useState, useRef, useCallback, useEffect } from 'react';

const PullToRefresh = ({ 
  children, 
  onRefresh,
  refreshThreshold = 80,
  maxPullDistance = 120,
  disabled = false,
  className = '',
  loadingText = "释放以刷新",
  pullingText = "下拉刷新",
  refreshingText = "刷新中...",
  completeText = "刷新完成"
}) => {
  const [pullDistance, setPullDistance] = useState(0);
  const [isPulling, setIsPulling] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [refreshState, setRefreshState] = useState('idle'); // idle, pulling, loading, refreshing, complete
  
  const containerRef = useRef(null);
  const startYRef = useRef(0);
  const lastYRef = useRef(0);
  const isPullValidRef = useRef(false);
  const animationRef = useRef(null);
  
  // 检查是否在顶部
  const isAtTop = useCallback(() => {
    if (!containerRef.current) return false;
    return containerRef.current.scrollTop <= 0;
  }, []);
  
  // 动画到指定位置
  const animateTo = useCallback((targetDistance, callback) => {
    const duration = 300;
    const startDistance = pullDistance;
    const startTime = Date.now();
    
    const animate = () => {
      const elapsed = Date.now() - startTime;
      const progress = Math.min(elapsed / duration, 1);
      
      // 缓动函数
      const easeOutBack = 1 - Math.pow(1 - progress, 3);
      const currentDistance = startDistance + (targetDistance - startDistance) * easeOutBack;
      
      setPullDistance(currentDistance);
      
      if (progress < 1) {
        animationRef.current = requestAnimationFrame(animate);
      } else {
        callback?.();
      }
    };
    
    animationRef.current = requestAnimationFrame(animate);
  }, [pullDistance]);
  
  // 重置状态
  const resetPull = useCallback(() => {
    if (animationRef.current) {
      cancelAnimationFrame(animationRef.current);
    }
    
    animateTo(0, () => {
      setIsPulling(false);
      setIsRefreshing(false);
      setRefreshState('idle');
    });
  }, [animateTo]);
  
  // 触发刷新
  const triggerRefresh = useCallback(async () => {
    if (isRefreshing || disabled) return;
    
    setIsRefreshing(true);
    setRefreshState('refreshing');
    
    // 保持在刷新位置
    animateTo(refreshThreshold);
    
    try {
      if (onRefresh) {
        await onRefresh();
      }
      
      setRefreshState('complete');
      
      // 显示完成状态一段时间
      setTimeout(() => {
        resetPull();
      }, 500);
      
    } catch (error) {
      console.error('刷新失败:', error);
      resetPull();
    }
  }, [isRefreshing, disabled, refreshThreshold, onRefresh, animateTo, resetPull]);
  
  // 手势开始
  const handleStart = useCallback((clientY) => {
    if (disabled || isRefreshing) return;
    
    if (isAtTop()) {
      startYRef.current = clientY;
      lastYRef.current = clientY;
      isPullValidRef.current = false;
      setIsPulling(true);
      setRefreshState('pulling');
    }
  }, [disabled, isRefreshing, isAtTop]);
  
  // 手势移动
  const handleMove = useCallback((clientY) => {
    if (!isPulling || disabled || isRefreshing) return;
    
    const deltaY = clientY - startYRef.current;
    const currentDeltaY = clientY - lastYRef.current;
    
    // 只处理向下拉动
    if (deltaY > 0) {
      if (!isPullValidRef.current && deltaY > 10) {
        isPullValidRef.current = true;
        
        // 阻止默认滚动
        if (containerRef.current) {
          containerRef.current.style.overflowY = 'hidden';
        }
      }
      
      if (isPullValidRef.current) {
        let newDistance = pullDistance + currentDeltaY;
        
        // 添加阻尼效果
        if (newDistance > maxPullDistance) {
          newDistance = maxPullDistance + (newDistance - maxPullDistance) * 0.2;
        }
        
        newDistance = Math.max(0, newDistance);
        setPullDistance(newDistance);
        
        // 更新状态
        if (newDistance >= refreshThreshold) {
          setRefreshState('loading');
        } else {
          setRefreshState('pulling');
        }
      }
    } else if (deltaY < -10) {
      // 向上滑动，取消拉动
      resetPull();
      return;
    }
    
    lastYRef.current = clientY;
  }, [isPulling, disabled, isRefreshing, pullDistance, maxPullDistance, refreshThreshold, resetPull]);
  
  // 手势结束
  const handleEnd = useCallback(() => {
    if (!isPulling || !isPullValidRef.current) {
      setIsPulling(false);
      return;
    }
    
    // 恢复滚动
    if (containerRef.current) {
      containerRef.current.style.overflowY = '';
    }
    
    if (pullDistance >= refreshThreshold) {
      triggerRefresh();
    } else {
      resetPull();
    }
  }, [isPulling, pullDistance, refreshThreshold, triggerRefresh, resetPull]);
  
  // 触摸事件
  const handleTouchStart = useCallback((e) => {
    const touch = e.touches[0];
    handleStart(touch.clientY);
  }, [handleStart]);
  
  const handleTouchMove = useCallback((e) => {
    const touch = e.touches[0];
    handleMove(touch.clientY);
    
    if (isPullValidRef.current && pullDistance > 0) {
      e.preventDefault();
    }
  }, [handleMove, pullDistance]);
  
  const handleTouchEnd = useCallback(() => {
    handleEnd();
  }, [handleEnd]);
  
  // 鼠标事件（开发调试用）
  const handleMouseDown = useCallback((e) => {
    handleStart(e.clientY);
  }, [handleStart]);
  
  const handleMouseMove = useCallback((e) => {
    handleMove(e.clientY);
  }, [handleMove]);
  
  const handleMouseUp = useCallback(() => {
    handleEnd();
  }, [handleEnd]);
  
  // 全局鼠标事件
  useEffect(() => {
    if (isPulling) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      
      return () => {
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [isPulling, handleMouseMove, handleMouseUp]);
  
  // 获取状态文本
  const getStateText = () => {
    switch (refreshState) {
      case 'pulling':
        return pullingText;
      case 'loading':
        return loadingText;
      case 'refreshing':
        return refreshingText;
      case 'complete':
        return completeText;
      default:
        return pullingText;
    }
  };
  
  // 获取进度百分比
  const getProgress = () => {
    return Math.min((pullDistance / refreshThreshold) * 100, 100);
  };
  
  // 获取图标旋转角度
  const getIconRotation = () => {
    if (refreshState === 'refreshing') {
      return 'rotate(360deg)';
    } else if (refreshState === 'loading') {
      return 'rotate(180deg)';
    } else {
      return `rotate(${Math.min((pullDistance / refreshThreshold) * 180, 180)}deg)`;
    }
  };
  
  const containerStyle = {
    position: 'relative',
    height: '100%',
    overflow: 'auto',
    transform: `translateY(${pullDistance}px)`,
    transition: isPulling ? 'none' : 'transform 0.3s ease',
    touchAction: 'auto'
  };
  
  const indicatorStyle = {
    position: 'absolute',
    top: -refreshThreshold,
    left: 0,
    right: 0,
    height: refreshThreshold,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(to bottom, rgba(255,255,255,0) 0%, rgba(255,255,255,0.9) 50%, rgba(255,255,255,1) 100%)',
    zIndex: 10,
    transition: 'opacity 0.2s ease',
    opacity: pullDistance > 0 ? 1 : 0
  };
  
  const iconStyle = {
    width: 24,
    height: 24,
    marginRight: 8,
    transform: getIconRotation(),
    transition: refreshState === 'refreshing' ? 'transform 1s linear infinite' : 'transform 0.3s ease',
    color: refreshState === 'loading' ? '#4CAF50' : '#666'
  };
  
  const progressStyle = {
    position: 'absolute',
    bottom: 0,
    left: 0,
    height: 2,
    backgroundColor: '#4CAF50',
    width: `${getProgress()}%`,
    transition: 'width 0.1s ease'
  };
  
  return (
    <div 
      ref={containerRef}
      className={`pull-to-refresh ${className} ${refreshState}`}
      style={containerStyle}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onMouseDown={handleMouseDown}
    >
      {/* 刷新指示器 */}
      <div className="refresh-indicator" style={indicatorStyle}>
        <div className="indicator-content" style={{ display: 'flex', alignItems: 'center', flexDirection: 'column' }}>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
            <div className="refresh-icon" style={iconStyle}>
              {refreshState === 'refreshing' ? (
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
                </svg>
              ) : refreshState === 'complete' ? (
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                </svg>
              ) : (
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M7.41 15.41L12 10.83l4.59 4.58L18 14l-6-6-6 6z"/>
                </svg>
              )}
            </div>
            <span className="refresh-text" style={{ fontSize: 14, color: '#666' }}>
              {getStateText()}
            </span>
          </div>
          <div className="progress-bar" style={progressStyle} />
        </div>
      </div>
      
      {/* 主内容 */}
      <div className="refresh-content">
        {children}
      </div>
    </div>
  );
};

export default PullToRefresh;