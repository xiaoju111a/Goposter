import React, { useState, useRef, useEffect, useCallback } from 'react';

const SwipeActions = ({ 
  children, 
  leftActions = [], 
  rightActions = [], 
  threshold = 80,
  className = '',
  disabled = false,
  onSwipeStart,
  onSwipeEnd,
  resetOnPropsChange = true,
  ...props 
}) => {
  const [offset, setOffset] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isOpen, setIsOpen] = useState(false);
  const [openDirection, setOpenDirection] = useState(null);
  
  const containerRef = useRef(null);
  const startXRef = useRef(0);
  const startYRef = useRef(0);
  const lastXRef = useRef(0);
  const isSwipeValidRef = useRef(false);
  const animationRef = useRef(null);
  
  // 重置状态
  const resetSwipe = useCallback(() => {
    if (animationRef.current) {
      cancelAnimationFrame(animationRef.current);
    }
    
    setOffset(0);
    setIsDragging(false);
    setIsOpen(false);
    setOpenDirection(null);
    lastXRef.current = 0;
  }, []);
  
  // 属性变化时重置
  useEffect(() => {
    if (resetOnPropsChange) {
      resetSwipe();
    }
  }, [children, leftActions, rightActions, resetOnPropsChange, resetSwipe]);
  
  // 手势开始处理
  const handleStart = useCallback((clientX, clientY) => {
    if (disabled) return;
    
    startXRef.current = clientX;
    startYRef.current = clientY;
    lastXRef.current = clientX;
    isSwipeValidRef.current = false;
    setIsDragging(true);
    
    onSwipeStart?.();
  }, [disabled, onSwipeStart]);
  
  // 手势移动处理
  const handleMove = useCallback((clientX, clientY) => {
    if (!isDragging || disabled) return;
    
    const deltaX = clientX - startXRef.current;
    const deltaY = Math.abs(clientY - startYRef.current);
    const currentDeltaX = clientX - lastXRef.current;
    
    // 判断是否为有效的横向滑动
    if (!isSwipeValidRef.current) {
      if (Math.abs(deltaX) > 10 && Math.abs(deltaX) > deltaY * 2) {
        isSwipeValidRef.current = true;
        
        // 阻止默认滚动行为
        if (containerRef.current) {
          containerRef.current.style.touchAction = 'none';
        }
      } else if (deltaY > 10) {
        // 垂直滑动，取消手势
        setIsDragging(false);
        return;
      }
    }
    
    if (!isSwipeValidRef.current) return;
    
    let newOffset = offset + currentDeltaX;
    
    // 限制滑动范围
    const maxLeftOffset = leftActions.length > 0 ? threshold * leftActions.length : 0;
    const maxRightOffset = rightActions.length > 0 ? threshold * rightActions.length : 0;
    
    newOffset = Math.max(-maxRightOffset, Math.min(maxLeftOffset, newOffset));
    
    // 添加阻尼效果
    if (newOffset > maxLeftOffset) {
      newOffset = maxLeftOffset + (newOffset - maxLeftOffset) * 0.3;
    } else if (newOffset < -maxRightOffset) {
      newOffset = -maxRightOffset + (newOffset + maxRightOffset) * 0.3;
    }
    
    setOffset(newOffset);
    lastXRef.current = clientX;
  }, [isDragging, disabled, offset, threshold, leftActions.length, rightActions.length]);
  
  // 手势结束处理
  const handleEnd = useCallback(() => {
    if (!isDragging || !isSwipeValidRef.current) {
      setIsDragging(false);
      return;
    }
    
    setIsDragging(false);
    
    // 恢复触摸行为
    if (containerRef.current) {
      containerRef.current.style.touchAction = '';
    }
    
    // 判断是否打开操作区域
    const leftThreshold = threshold * 0.6;
    const rightThreshold = -threshold * 0.6;
    
    if (offset > leftThreshold && leftActions.length > 0) {
      // 打开左侧操作
      const targetOffset = threshold * Math.min(leftActions.length, Math.ceil(offset / threshold));
      animateToOffset(targetOffset);
      setIsOpen(true);
      setOpenDirection('left');
    } else if (offset < rightThreshold && rightActions.length > 0) {
      // 打开右侧操作
      const targetOffset = -threshold * Math.min(rightActions.length, Math.ceil(Math.abs(offset) / threshold));
      animateToOffset(targetOffset);
      setIsOpen(true);
      setOpenDirection('right');
    } else {
      // 关闭操作区域
      animateToOffset(0);
      setIsOpen(false);
      setOpenDirection(null);
    }
    
    onSwipeEnd?.({ isOpen: offset !== 0, direction: offset > 0 ? 'left' : 'right' });
  }, [isDragging, offset, threshold, leftActions.length, rightActions.length, onSwipeEnd]);
  
  // 动画到指定位置
  const animateToOffset = useCallback((targetOffset) => {
    const duration = 300;
    const startOffset = offset;
    const startTime = Date.now();
    
    const animate = () => {
      const elapsed = Date.now() - startTime;
      const progress = Math.min(elapsed / duration, 1);
      
      // 使用缓动函数
      const easeOutQuart = 1 - Math.pow(1 - progress, 4);
      const currentOffset = startOffset + (targetOffset - startOffset) * easeOutQuart;
      
      setOffset(currentOffset);
      
      if (progress < 1) {
        animationRef.current = requestAnimationFrame(animate);
      }
    };
    
    animationRef.current = requestAnimationFrame(animate);
  }, [offset]);
  
  // 触摸事件处理
  const handleTouchStart = useCallback((e) => {
    const touch = e.touches[0];
    handleStart(touch.clientX, touch.clientY);
  }, [handleStart]);
  
  const handleTouchMove = useCallback((e) => {
    const touch = e.touches[0];
    handleMove(touch.clientX, touch.clientY);
    
    if (isSwipeValidRef.current) {
      e.preventDefault();
    }
  }, [handleMove]);
  
  const handleTouchEnd = useCallback(() => {
    handleEnd();
  }, [handleEnd]);
  
  // 鼠标事件处理（开发调试用）
  const handleMouseDown = useCallback((e) => {
    handleStart(e.clientX, e.clientY);
  }, [handleStart]);
  
  const handleMouseMove = useCallback((e) => {
    handleMove(e.clientX, e.clientY);
  }, [handleMove]);
  
  const handleMouseUp = useCallback(() => {
    handleEnd();
  }, [handleEnd]);
  
  // 全局鼠标事件监听
  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      
      return () => {
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [isDragging, handleMouseMove, handleMouseUp]);
  
  // 操作按钮点击处理
  const handleActionClick = useCallback((action, index, direction) => {
    action.handler?.();
    
    // 点击后关闭
    if (action.closeOnClick !== false) {
      animateToOffset(0);
      setIsOpen(false);
      setOpenDirection(null);
    }
  }, [animateToOffset]);
  
  // 渲染操作按钮
  const renderActions = (actions, direction) => {
    return actions.map((action, index) => (
      <div
        key={index}
        className={`swipe-action ${action.className || ''}`}
        style={{
          backgroundColor: action.backgroundColor || '#007AFF',
          color: action.color || 'white',
          width: threshold,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexDirection: 'column',
          cursor: 'pointer',
          userSelect: 'none',
          transition: isDragging ? 'none' : 'all 0.2s ease',
          ...action.style
        }}
        onClick={() => handleActionClick(action, index, direction)}
      >
        {action.icon && (
          <div className="action-icon" style={{ fontSize: '1.2rem', marginBottom: '4px' }}>
            {action.icon}
          </div>
        )}
        {action.text && (
          <div className="action-text" style={{ fontSize: '0.8rem', textAlign: 'center' }}>
            {action.text}
          </div>
        )}
      </div>
    ));
  };
  
  const containerStyle = {
    position: 'relative',
    overflow: 'hidden',
    touchAction: isDragging ? 'none' : 'auto',
    ...props.style
  };
  
  const contentStyle = {
    transform: `translateX(${offset}px)`,
    transition: isDragging ? 'none' : 'transform 0.3s ease',
    position: 'relative',
    zIndex: 1,
    backgroundColor: 'inherit'
  };
  
  const leftActionsStyle = {
    position: 'absolute',
    left: 0,
    top: 0,
    bottom: 0,
    display: 'flex',
    transform: `translateX(${-threshold * leftActions.length}px)`,
    zIndex: 0
  };
  
  const rightActionsStyle = {
    position: 'absolute',
    right: 0,
    top: 0,
    bottom: 0,
    display: 'flex',
    transform: `translateX(${threshold * rightActions.length}px)`,
    zIndex: 0
  };
  
  return (
    <div
      ref={containerRef}
      className={`swipe-container ${className} ${isDragging ? 'dragging' : ''} ${isOpen ? 'open' : ''}`}
      style={containerStyle}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onMouseDown={handleMouseDown}
    >
      {/* 左侧操作 */}
      {leftActions.length > 0 && (
        <div className="left-actions" style={leftActionsStyle}>
          {renderActions(leftActions, 'left')}
        </div>
      )}
      
      {/* 右侧操作 */}
      {rightActions.length > 0 && (
        <div className="right-actions" style={rightActionsStyle}>
          {renderActions(rightActions, 'right')}
        </div>
      )}
      
      {/* 主内容 */}
      <div className="swipe-content" style={contentStyle}>
        {children}
      </div>
    </div>
  );
};

export default SwipeActions;