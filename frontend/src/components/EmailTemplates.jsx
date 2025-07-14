import React, { useState, useEffect, useCallback } from 'react';
import EmailEditor from './EmailEditor.jsx';

const EmailTemplates = ({
  onTemplateSelect,
  onTemplateCreate,
  onTemplateUpdate,
  onTemplateDelete,
  className = '',
  style = {}
}) => {
  const [templates, setTemplates] = useState([]);
  const [selectedTemplate, setSelectedTemplate] = useState(null);
  const [isEditing, setIsEditing] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [loading, setLoading] = useState(false);
  
  // 预设模板分类
  const categories = [
    { value: 'all', label: '全部模板' },
    { value: 'business', label: '商务邮件' },
    { value: 'personal', label: '个人邮件' },
    { value: 'marketing', label: '营销邮件' },
    { value: 'notification', label: '通知邮件' },
    { value: 'support', label: '客服邮件' },
    { value: 'custom', label: '自定义' }
  ];
  
  // 预设模板
  const defaultTemplates = [
    {
      id: 'welcome',
      name: '欢迎邮件',
      category: 'business',
      description: '新用户欢迎邮件模板',
      subject: '欢迎加入 FreeAgent！',
      content: `
        <h2>欢迎来到 FreeAgent！</h2>
        <p>亲爱的用户，</p>
        <p>感谢您注册 FreeAgent 邮件服务！我们很高兴您能加入我们的社区。</p>
        
        <h3>您可以开始：</h3>
        <ul>
          <li>📧 发送和接收邮件</li>
          <li>📎 管理附件</li>
          <li>🔍 使用强大的搜索功能</li>
          <li>📱 在任何设备上访问您的邮箱</li>
        </ul>
        
        <p>如果您有任何问题，请随时联系我们的客服团队。</p>
        
        <p>祝您使用愉快！<br>
        FreeAgent 团队</p>
      `,
      variables: ['用户名', '注册日期'],
      tags: ['欢迎', '新用户', '介绍'],
      isBuiltIn: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    },
    {
      id: 'meeting',
      name: '会议邀请',
      category: 'business',
      description: '会议邀请邮件模板',
      subject: '会议邀请：{{会议主题}}',
      content: `
        <h2>会议邀请</h2>
        <p>您好，</p>
        <p>我想邀请您参加以下会议：</p>
        
        <div style="background: #f5f5f5; padding: 15px; border-left: 4px solid #4CAF50; margin: 20px 0;">
          <h3>{{会议主题}}</h3>
          <p><strong>时间：</strong> {{会议时间}}</p>
          <p><strong>地点：</strong> {{会议地点}}</p>
          <p><strong>会议链接：</strong> <a href="{{会议链接}}">点击加入</a></p>
        </div>
        
        <h3>议程：</h3>
        <ol>
          <li>{{议程项目1}}</li>
          <li>{{议程项目2}}</li>
          <li>{{议程项目3}}</li>
        </ol>
        
        <p>请确认您的参会情况，如有冲突请及时告知。</p>
        
        <p>期待您的参与！</p>
      `,
      variables: ['会议主题', '会议时间', '会议地点', '会议链接', '议程项目1', '议程项目2', '议程项目3'],
      tags: ['会议', '邀请', '商务'],
      isBuiltIn: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    },
    {
      id: 'followup',
      name: '跟进邮件',
      category: 'business',
      description: '客户跟进邮件模板',
      subject: '跟进：{{项目名称}}',
      content: `
        <h2>项目跟进</h2>
        <p>亲爱的 {{客户名称}}，</p>
        <p>希望您一切都好！我想跟进一下我们之前讨论的 {{项目名称}} 项目。</p>
        
        <h3>当前进展：</h3>
        <ul>
          <li>✅ {{已完成事项1}}</li>
          <li>✅ {{已完成事项2}}</li>
          <li>🔄 {{进行中事项}}</li>
          <li>📅 {{待完成事项}}</li>
        </ul>
        
        <h3>下一步计划：</h3>
        <p>{{下一步计划描述}}</p>
        
        <p>预计完成时间：<strong>{{预计完成时间}}</strong></p>
        
        <p>如果您有任何问题或建议，请随时联系我。</p>
        
        <p>谢谢！<br>
        {{您的姓名}}</p>
      `,
      variables: ['客户名称', '项目名称', '已完成事项1', '已完成事项2', '进行中事项', '待完成事项', '下一步计划描述', '预计完成时间', '您的姓名'],
      tags: ['跟进', '项目', '客户'],
      isBuiltIn: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    },
    {
      id: 'newsletter',
      name: '新闻简报',
      category: 'marketing',
      description: '新闻简报邮件模板',
      subject: '{{公司名称}} 月度简报 - {{月份}}',
      content: `
        <div style="max-width: 600px; margin: 0 auto; font-family: Arial, sans-serif;">
          <header style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center;">
            <h1>{{公司名称}}</h1>
            <p>{{月份}} 月度简报</p>
          </header>
          
          <div style="padding: 30px;">
            <h2>本月亮点</h2>
            <div style="background: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
              <h3>🎉 {{亮点标题1}}</h3>
              <p>{{亮点内容1}}</p>
            </div>
            
            <div style="background: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
              <h3>📈 {{亮点标题2}}</h3>
              <p>{{亮点内容2}}</p>
            </div>
            
            <h2>产品更新</h2>
            <ul>
              <li>{{更新项目1}}</li>
              <li>{{更新项目2}}</li>
              <li>{{更新项目3}}</li>
            </ul>
            
            <h2>即将到来</h2>
            <p>{{即将到来的内容}}</p>
            
            <div style="background: #e3f2fd; padding: 20px; border-radius: 8px; text-align: center; margin: 30px 0;">
              <h3>联系我们</h3>
              <p>有问题或建议？我们很乐意听到您的声音！</p>
              <a href="mailto:{{联系邮箱}}" style="background: #2196F3; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">联系我们</a>
            </div>
          </div>
          
          <footer style="background: #f5f5f5; padding: 20px; text-align: center; border-top: 1px solid #e0e0e0;">
            <p style="color: #666; font-size: 12px;">
              © {{年份}} {{公司名称}}. 保留所有权利.<br>
              如不想再收到此邮件，请<a href="{{取消订阅链接}}">取消订阅</a>
            </p>
          </footer>
        </div>
      `,
      variables: ['公司名称', '月份', '亮点标题1', '亮点内容1', '亮点标题2', '亮点内容2', '更新项目1', '更新项目2', '更新项目3', '即将到来的内容', '联系邮箱', '年份', '取消订阅链接'],
      tags: ['简报', '营销', '新闻'],
      isBuiltIn: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    },
    {
      id: 'support',
      name: '客服回复',
      category: 'support',
      description: '客服支持邮件模板',
      subject: '回复：{{工单编号}} - {{问题描述}}',
      content: `
        <h2>客服支持</h2>
        <p>亲爱的 {{客户姓名}}，</p>
        <p>感谢您联系我们的客服团队。关于您提到的问题，我们已经进行了详细的调查。</p>
        
        <div style="background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 5px; margin: 20px 0;">
          <h3>📋 工单信息</h3>
          <p><strong>工单编号：</strong> {{工单编号}}</p>
          <p><strong>问题类型：</strong> {{问题类型}}</p>
          <p><strong>创建时间：</strong> {{创建时间}}</p>
        </div>
        
        <h3>解决方案</h3>
        <p>{{解决方案描述}}</p>
        
        <h3>操作步骤</h3>
        <ol>
          <li>{{步骤1}}</li>
          <li>{{步骤2}}</li>
          <li>{{步骤3}}</li>
        </ol>
        
        <div style="background: #d4edda; border: 1px solid #c3e6cb; padding: 15px; border-radius: 5px; margin: 20px 0;">
          <h3>💡 温馨提示</h3>
          <p>{{额外提示}}</p>
        </div>
        
        <p>如果这个解决方案没有解决您的问题，或者您有其他疑问，请随时回复此邮件。</p>
        
        <p>我们致力于为您提供最好的服务体验！</p>
        
        <p>最佳祝愿，<br>
        {{客服代表姓名}}<br>
        {{公司名称}} 客服团队</p>
      `,
      variables: ['客户姓名', '工单编号', '问题描述', '问题类型', '创建时间', '解决方案描述', '步骤1', '步骤2', '步骤3', '额外提示', '客服代表姓名', '公司名称'],
      tags: ['客服', '支持', '解决方案'],
      isBuiltIn: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    }
  ];
  
  // 加载模板
  const loadTemplates = useCallback(async () => {
    setLoading(true);
    try {
      // 从localStorage加载自定义模板
      const savedTemplates = localStorage.getItem('email-templates');
      const customTemplates = savedTemplates ? JSON.parse(savedTemplates) : [];
      
      // 合并预设模板和自定义模板
      setTemplates([...defaultTemplates, ...customTemplates]);
    } catch (error) {
      console.error('加载模板失败:', error);
      setTemplates(defaultTemplates);
    } finally {
      setLoading(false);
    }
  }, []);
  
  // 保存模板到localStorage
  const saveTemplateToStorage = useCallback((template) => {
    try {
      const savedTemplates = localStorage.getItem('email-templates');
      const customTemplates = savedTemplates ? JSON.parse(savedTemplates) : [];
      
      const existingIndex = customTemplates.findIndex(t => t.id === template.id);
      if (existingIndex >= 0) {
        customTemplates[existingIndex] = template;
      } else {
        customTemplates.push(template);
      }
      
      localStorage.setItem('email-templates', JSON.stringify(customTemplates));
      return true;
    } catch (error) {
      console.error('保存模板失败:', error);
      return false;
    }
  }, []);
  
  // 从localStorage删除模板
  const deleteTemplateFromStorage = useCallback((templateId) => {
    try {
      const savedTemplates = localStorage.getItem('email-templates');
      const customTemplates = savedTemplates ? JSON.parse(savedTemplates) : [];
      
      const filteredTemplates = customTemplates.filter(t => t.id !== templateId);
      localStorage.setItem('email-templates', JSON.stringify(filteredTemplates));
      return true;
    } catch (error) {
      console.error('删除模板失败:', error);
      return false;
    }
  }, []);
  
  // 创建新模板
  const handleCreateTemplate = useCallback((templateData) => {
    const newTemplate = {
      id: Date.now().toString(),
      name: templateData.name || '新模板',
      category: templateData.category || 'custom',
      description: templateData.description || '',
      subject: templateData.subject || '',
      content: templateData.content || '',
      variables: templateData.variables || [],
      tags: templateData.tags || [],
      isBuiltIn: false,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
    
    if (saveTemplateToStorage(newTemplate)) {
      setTemplates(prev => [...prev, newTemplate]);
      onTemplateCreate?.(newTemplate);
      setIsCreating(false);
      setSelectedTemplate(null);
    }
  }, [saveTemplateToStorage, onTemplateCreate]);
  
  // 更新模板
  const handleUpdateTemplate = useCallback((templateData) => {
    const updatedTemplate = {
      ...selectedTemplate,
      ...templateData,
      updatedAt: new Date().toISOString()
    };
    
    if (updatedTemplate.isBuiltIn) {
      // 内置模板不能修改，创建为新模板
      const newTemplate = {
        ...updatedTemplate,
        id: Date.now().toString(),
        name: updatedTemplate.name + ' (自定义)',
        isBuiltIn: false,
        createdAt: new Date().toISOString()
      };
      
      if (saveTemplateToStorage(newTemplate)) {
        setTemplates(prev => [...prev, newTemplate]);
        onTemplateCreate?.(newTemplate);
      }
    } else {
      if (saveTemplateToStorage(updatedTemplate)) {
        setTemplates(prev => prev.map(t => t.id === updatedTemplate.id ? updatedTemplate : t));
        onTemplateUpdate?.(updatedTemplate);
      }
    }
    
    setIsEditing(false);
    setSelectedTemplate(null);
  }, [selectedTemplate, saveTemplateToStorage, onTemplateCreate, onTemplateUpdate]);
  
  // 删除模板
  const handleDeleteTemplate = useCallback((template) => {
    if (template.isBuiltIn) {
      alert('内置模板不能删除');
      return;
    }
    
    if (window.confirm(`确定要删除模板"${template.name}"吗？`)) {
      if (deleteTemplateFromStorage(template.id)) {
        setTemplates(prev => prev.filter(t => t.id !== template.id));
        onTemplateDelete?.(template);
        if (selectedTemplate?.id === template.id) {
          setSelectedTemplate(null);
          setIsEditing(false);
        }
      }
    }
  }, [deleteTemplateFromStorage, onTemplateDelete, selectedTemplate]);
  
  // 筛选模板
  const filteredTemplates = templates.filter(template => {
    const matchesSearch = template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         template.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         template.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()));
    
    const matchesCategory = categoryFilter === 'all' || template.category === categoryFilter;
    
    return matchesSearch && matchesCategory;
  });
  
  // 使用模板
  const handleUseTemplate = useCallback((template) => {
    onTemplateSelect?.(template);
  }, [onTemplateSelect]);
  
  // 组件挂载时加载模板
  useEffect(() => {
    loadTemplates();
  }, [loadTemplates]);
  
  return (
    <div className={`email-templates ${className}`} style={style}>
      {/* 头部工具栏 */}
      <div className="templates-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '16px',
        borderBottom: '1px solid #e0e0e0',
        backgroundColor: '#f8f9fa'
      }}>
        <h2 style={{ margin: 0, color: '#333' }}>邮件模板库</h2>
        <button
          onClick={() => setIsCreating(true)}
          style={{
            padding: '8px 16px',
            border: '1px solid #4CAF50',
            borderRadius: '6px',
            backgroundColor: '#4CAF50',
            color: 'white',
            cursor: 'pointer',
            fontSize: '14px'
          }}
        >
          ➕ 创建模板
        </button>
      </div>
      
      {/* 搜索和筛选 */}
      <div className="templates-filters" style={{
        display: 'flex',
        gap: '12px',
        padding: '16px',
        borderBottom: '1px solid #e0e0e0'
      }}>
        <input
          type="text"
          placeholder="搜索模板..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          style={{
            flex: 1,
            padding: '8px 12px',
            border: '1px solid #ddd',
            borderRadius: '6px',
            fontSize: '14px'
          }}
        />
        <select
          value={categoryFilter}
          onChange={(e) => setCategoryFilter(e.target.value)}
          style={{
            padding: '8px 12px',
            border: '1px solid #ddd',
            borderRadius: '6px',
            fontSize: '14px',
            minWidth: '150px'
          }}
        >
          {categories.map(category => (
            <option key={category.value} value={category.value}>
              {category.label}
            </option>
          ))}
        </select>
      </div>
      
      <div className="templates-container" style={{ display: 'flex', height: 'calc(100vh - 200px)' }}>
        {/* 模板列表 */}
        <div className="templates-list" style={{
          width: '300px',
          borderRight: '1px solid #e0e0e0',
          overflow: 'auto'
        }}>
          {loading ? (
            <div style={{ padding: '20px', textAlign: 'center' }}>
              正在加载模板...
            </div>
          ) : filteredTemplates.length === 0 ? (
            <div style={{ padding: '20px', textAlign: 'center', color: '#999' }}>
              没有找到匹配的模板
            </div>
          ) : (
            filteredTemplates.map(template => (
              <div
                key={template.id}
                className={`template-item ${selectedTemplate?.id === template.id ? 'selected' : ''}`}
                style={{
                  padding: '12px',
                  borderBottom: '1px solid #f0f0f0',
                  cursor: 'pointer',
                  backgroundColor: selectedTemplate?.id === template.id ? '#e3f2fd' : 'white',
                  transition: 'background-color 0.2s ease'
                }}
                onClick={() => setSelectedTemplate(template)}
                onMouseEnter={(e) => {
                  if (selectedTemplate?.id !== template.id) {
                    e.target.style.backgroundColor = '#f5f5f5';
                  }
                }}
                onMouseLeave={(e) => {
                  if (selectedTemplate?.id !== template.id) {
                    e.target.style.backgroundColor = 'white';
                  }
                }}
              >
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'flex-start',
                  marginBottom: '8px'
                }}>
                  <h4 style={{ margin: 0, fontSize: '14px', color: '#333' }}>
                    {template.name}
                    {template.isBuiltIn && (
                      <span style={{
                        marginLeft: '6px',
                        fontSize: '10px',
                        backgroundColor: '#2196F3',
                        color: 'white',
                        padding: '2px 6px',
                        borderRadius: '10px'
                      }}>
                        内置
                      </span>
                    )}
                  </h4>
                  <span style={{
                    fontSize: '10px',
                    backgroundColor: '#f0f0f0',
                    padding: '2px 6px',
                    borderRadius: '8px',
                    color: '#666'
                  }}>
                    {categories.find(c => c.value === template.category)?.label}
                  </span>
                </div>
                <p style={{
                  margin: 0,
                  fontSize: '12px',
                  color: '#666',
                  lineHeight: '1.4',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  display: '-webkit-box',
                  WebkitLineClamp: 2,
                  WebkitBoxOrient: 'vertical'
                }}>
                  {template.description}
                </p>
                {template.tags.length > 0 && (
                  <div style={{ marginTop: '6px' }}>
                    {template.tags.slice(0, 3).map((tag, index) => (
                      <span
                        key={index}
                        style={{
                          fontSize: '10px',
                          backgroundColor: '#e8f5e8',
                          color: '#4CAF50',
                          padding: '2px 4px',
                          borderRadius: '6px',
                          marginRight: '4px'
                        }}
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))
          )}
        </div>
        
        {/* 模板预览和编辑 */}
        <div className="template-preview" style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          {isCreating || isEditing ? (
            <TemplateEditor
              template={isEditing ? selectedTemplate : null}
              onSave={isCreating ? handleCreateTemplate : handleUpdateTemplate}
              onCancel={() => {
                setIsCreating(false);
                setIsEditing(false);
                setSelectedTemplate(null);
              }}
              categories={categories}
            />
          ) : selectedTemplate ? (
            <TemplatePreview
              template={selectedTemplate}
              onEdit={() => setIsEditing(true)}
              onDelete={() => handleDeleteTemplate(selectedTemplate)}
              onUse={() => handleUseTemplate(selectedTemplate)}
            />
          ) : (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: '#999',
              fontSize: '16px'
            }}>
              选择一个模板来预览
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// 模板预览组件
const TemplatePreview = ({ template, onEdit, onDelete, onUse }) => {
  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* 预览头部 */}
      <div style={{
        padding: '16px',
        borderBottom: '1px solid #e0e0e0',
        backgroundColor: '#f8f9fa'
      }}>
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: '12px'
        }}>
          <h3 style={{ margin: 0, color: '#333' }}>{template.name}</h3>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              onClick={onUse}
              style={{
                padding: '6px 12px',
                border: '1px solid #4CAF50',
                borderRadius: '4px',
                backgroundColor: '#4CAF50',
                color: 'white',
                cursor: 'pointer',
                fontSize: '12px'
              }}
            >
              使用模板
            </button>
            <button
              onClick={onEdit}
              style={{
                padding: '6px 12px',
                border: '1px solid #2196F3',
                borderRadius: '4px',
                backgroundColor: '#fff',
                color: '#2196F3',
                cursor: 'pointer',
                fontSize: '12px'
              }}
            >
              编辑
            </button>
            {!template.isBuiltIn && (
              <button
                onClick={onDelete}
                style={{
                  padding: '6px 12px',
                  border: '1px solid #f44336',
                  borderRadius: '4px',
                  backgroundColor: '#fff',
                  color: '#f44336',
                  cursor: 'pointer',
                  fontSize: '12px'
                }}
              >
                删除
              </button>
            )}
          </div>
        </div>
        
        <p style={{ margin: 0, color: '#666', fontSize: '14px' }}>
          {template.description}
        </p>
        
        {template.tags.length > 0 && (
          <div style={{ marginTop: '8px' }}>
            {template.tags.map((tag, index) => (
              <span
                key={index}
                style={{
                  fontSize: '11px',
                  backgroundColor: '#e8f5e8',
                  color: '#4CAF50',
                  padding: '3px 6px',
                  borderRadius: '8px',
                  marginRight: '6px'
                }}
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>
      
      {/* 主题预览 */}
      <div style={{ padding: '16px', borderBottom: '1px solid #e0e0e0' }}>
        <label style={{ fontSize: '12px', color: '#666', display: 'block', marginBottom: '4px' }}>
          主题：
        </label>
        <div style={{
          padding: '8px',
          backgroundColor: '#f5f5f5',
          borderRadius: '4px',
          fontSize: '14px',
          fontFamily: 'monospace'
        }}>
          {template.subject || '无主题'}
        </div>
      </div>
      
      {/* 内容预览 */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        <div style={{ padding: '16px' }}>
          <label style={{ fontSize: '12px', color: '#666', display: 'block', marginBottom: '8px' }}>
            内容预览：
          </label>
          <div
            style={{
              border: '1px solid #e0e0e0',
              borderRadius: '4px',
              padding: '16px',
              backgroundColor: '#fff',
              minHeight: '200px'
            }}
            dangerouslySetInnerHTML={{ __html: template.content }}
          />
        </div>
        
        {/* 变量列表 */}
        {template.variables && template.variables.length > 0 && (
          <div style={{ padding: '16px', borderTop: '1px solid #e0e0e0' }}>
            <label style={{ fontSize: '12px', color: '#666', display: 'block', marginBottom: '8px' }}>
              可用变量：
            </label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
              {template.variables.map((variable, index) => (
                <span
                  key={index}
                  style={{
                    fontSize: '11px',
                    backgroundColor: '#fff3cd',
                    color: '#856404',
                    padding: '4px 8px',
                    borderRadius: '8px',
                    border: '1px solid #ffeaa7'
                  }}
                >
                  {`{{${variable}}}`}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

// 模板编辑器组件
const TemplateEditor = ({ template, onSave, onCancel, categories }) => {
  const [formData, setFormData] = useState({
    name: template?.name || '',
    category: template?.category || 'custom',
    description: template?.description || '',
    subject: template?.subject || '',
    content: template?.content || '',
    variables: template?.variables || [],
    tags: template?.tags?.join(', ') || ''
  });
  
  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (!formData.name.trim()) {
      alert('请输入模板名称');
      return;
    }
    
    const templateData = {
      ...formData,
      variables: formData.variables,
      tags: formData.tags.split(',').map(tag => tag.trim()).filter(tag => tag)
    };
    
    onSave(templateData);
  };
  
  const extractVariables = () => {
    const content = formData.subject + ' ' + formData.content;
    const matches = content.match(/\{\{([^}]+)\}\}/g);
    if (matches) {
      const variables = [...new Set(matches.map(match => match.slice(2, -2)))];
      setFormData(prev => ({ ...prev, variables }));
    }
  };
  
  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{
        padding: '16px',
        borderBottom: '1px solid #e0e0e0',
        backgroundColor: '#f8f9fa'
      }}>
        <h3 style={{ margin: 0, color: '#333' }}>
          {template ? '编辑模板' : '创建新模板'}
        </h3>
      </div>
      
      <form onSubmit={handleSubmit} style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div style={{ flex: 1, overflow: 'auto', padding: '16px' }}>
          {/* 基本信息 */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '16px',
            marginBottom: '16px'
          }}>
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                模板名称 *
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
                required
              />
            </div>
            
            <div>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                分类
              </label>
              <select
                value={formData.category}
                onChange={(e) => setFormData(prev => ({ ...prev, category: e.target.value }))}
                style={{
                  width: '100%',
                  padding: '8px',
                  border: '1px solid #ddd',
                  borderRadius: '4px'
                }}
              >
                {categories.filter(c => c.value !== 'all').map(category => (
                  <option key={category.value} value={category.value}>
                    {category.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          
          {/* 描述 */}
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
              描述
            </label>
            <input
              type="text"
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px'
              }}
              placeholder="简短描述这个模板的用途"
            />
          </div>
          
          {/* 主题 */}
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
              邮件主题
            </label>
            <input
              type="text"
              value={formData.subject}
              onChange={(e) => setFormData(prev => ({ ...prev, subject: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px'
              }}
              placeholder="使用 {{变量名}} 来插入变量"
            />
          </div>
          
          {/* 内容编辑器 */}
          <div style={{ marginBottom: '16px' }}>
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '4px'
            }}>
              <label style={{ fontSize: '12px', color: '#666' }}>
                邮件内容
              </label>
              <button
                type="button"
                onClick={extractVariables}
                style={{
                  padding: '4px 8px',
                  border: '1px solid #4CAF50',
                  borderRadius: '4px',
                  backgroundColor: '#fff',
                  color: '#4CAF50',
                  cursor: 'pointer',
                  fontSize: '11px'
                }}
              >
                提取变量
              </button>
            </div>
            <EmailEditor
              value={formData.content}
              onChange={(content) => setFormData(prev => ({ ...prev, content }))}
              placeholder="输入邮件内容，使用 {{变量名}} 来插入变量"
              style={{ minHeight: '300px' }}
            />
          </div>
          
          {/* 标签 */}
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
              标签 (用逗号分隔)
            </label>
            <input
              type="text"
              value={formData.tags}
              onChange={(e) => setFormData(prev => ({ ...prev, tags: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px'
              }}
              placeholder="例如: 欢迎, 新用户, 介绍"
            />
          </div>
          
          {/* 变量列表 */}
          {formData.variables.length > 0 && (
            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '12px', color: '#666', marginBottom: '4px' }}>
                发现的变量:
              </label>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                {formData.variables.map((variable, index) => (
                  <span
                    key={index}
                    style={{
                      fontSize: '11px',
                      backgroundColor: '#fff3cd',
                      color: '#856404',
                      padding: '4px 8px',
                      borderRadius: '8px',
                      border: '1px solid #ffeaa7'
                    }}
                  >
                    {`{{${variable}}}`}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
        
        {/* 操作按钮 */}
        <div style={{
          padding: '16px',
          borderTop: '1px solid #e0e0e0',
          backgroundColor: '#f8f9fa',
          display: 'flex',
          justifyContent: 'flex-end',
          gap: '8px'
        }}>
          <button
            type="button"
            onClick={onCancel}
            style={{
              padding: '8px 16px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              backgroundColor: '#fff',
              color: '#333',
              cursor: 'pointer'
            }}
          >
            取消
          </button>
          <button
            type="submit"
            style={{
              padding: '8px 16px',
              border: '1px solid #4CAF50',
              borderRadius: '4px',
              backgroundColor: '#4CAF50',
              color: 'white',
              cursor: 'pointer'
            }}
          >
            {template ? '更新模板' : '创建模板'}
          </button>
        </div>
      </form>
    </div>
  );
};

export default EmailTemplates;