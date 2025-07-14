package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type ElasticsearchClient struct {
	client    *elasticsearch.Client
	indexName string
	enabled   bool
}

type EmailDocument struct {
	ID          string    `json:"id"`
	Mailbox     string    `json:"mailbox"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	Timestamp   time.Time `json:"timestamp"`
	HasAttachment bool    `json:"has_attachment"`
	IsRead      bool      `json:"is_read"`
	Priority    string    `json:"priority"`
	Size        int64     `json:"size"`
	Tags        []string  `json:"tags"`
	Headers     map[string]string `json:"headers"`
}

type SearchResult struct {
	Total    int64           `json:"total"`
	Emails   []EmailDocument `json:"emails"`
	Took     int             `json:"took"`
	TimedOut bool            `json:"timed_out"`
}

type SearchRequest struct {
	Query      string            `json:"query"`
	Mailbox    string            `json:"mailbox,omitempty"`
	From       string            `json:"from,omitempty"`
	To         string            `json:"to,omitempty"`
	Subject    string            `json:"subject,omitempty"`
	DateStart  string            `json:"date_start,omitempty"`
	DateEnd    string            `json:"date_end,omitempty"`
	HasAttachment *bool          `json:"has_attachment,omitempty"`
	IsRead     *bool             `json:"is_read,omitempty"`
	Priority   string            `json:"priority,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Size       int               `json:"size"`
	From_      int               `json:"from"`
	Sort       string            `json:"sort,omitempty"`
	Highlight  bool              `json:"highlight"`
}

// NewElasticsearchClient 创建ElasticSearch客户端
func NewElasticsearchClient() *ElasticsearchClient {
	// 尝试连接到ElasticSearch
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
			"http://elasticsearch:9200", // Docker环境
		},
		Username: "",
		Password: "",
		CloudID:  "",
		APIKey:   "",
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Printf("ElasticSearch连接失败: %v", err)
		return &ElasticsearchClient{enabled: false}
	}

	// 测试连接
	res, err := client.Info()
	if err != nil {
		log.Printf("ElasticSearch连接测试失败: %v", err)
		return &ElasticsearchClient{enabled: false}
	}
	res.Body.Close()

	esClient := &ElasticsearchClient{
		client:    client,
		indexName: "freeagent-mail",
		enabled:   true,
	}

	// 初始化索引
	if err := esClient.initializeIndex(); err != nil {
		log.Printf("ElasticSearch索引初始化失败: %v", err)
		esClient.enabled = false
	}

	log.Printf("ElasticSearch客户端初始化成功")
	return esClient
}

// IsEnabled 检查ElasticSearch是否可用
func (es *ElasticsearchClient) IsEnabled() bool {
	return es.enabled
}

// initializeIndex 初始化ElasticSearch索引
func (es *ElasticsearchClient) initializeIndex() error {
	if !es.enabled {
		return fmt.Errorf("ElasticSearch未启用")
	}

	// 检查索引是否存在
	req := esapi.IndicesExistsRequest{
		Index: []string{es.indexName},
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return fmt.Errorf("检查索引失败: %v", err)
	}
	defer res.Body.Close()

	// 如果索引不存在，创建索引
	if res.StatusCode == 404 {
		return es.createIndex()
	}

	return nil
}

// createIndex 创建ElasticSearch索引
func (es *ElasticsearchClient) createIndex() error {
	mapping := `{
		"settings": {
			"number_of_shards": 1,
			"number_of_replicas": 0,
			"analysis": {
				"analyzer": {
					"email_analyzer": {
						"type": "custom",
						"tokenizer": "standard",
						"filter": ["lowercase", "stop", "snowball"]
					},
					"chinese_analyzer": {
						"type": "custom",
						"tokenizer": "ik_max_word",
						"filter": ["lowercase"]
					}
				}
			}
		},
		"mappings": {
			"properties": {
				"id": {"type": "keyword"},
				"mailbox": {"type": "keyword"},
				"from": {
					"type": "text",
					"analyzer": "email_analyzer",
					"fields": {
						"keyword": {"type": "keyword"}
					}
				},
				"to": {
					"type": "text",
					"analyzer": "email_analyzer",
					"fields": {
						"keyword": {"type": "keyword"}
					}
				},
				"subject": {
					"type": "text",
					"analyzer": "chinese_analyzer",
					"fields": {
						"keyword": {"type": "keyword"}
					}
				},
				"body": {
					"type": "text",
					"analyzer": "chinese_analyzer"
				},
				"timestamp": {"type": "date"},
				"has_attachment": {"type": "boolean"},
				"is_read": {"type": "boolean"},
				"priority": {"type": "keyword"},
				"size": {"type": "long"},
				"tags": {"type": "keyword"},
				"headers": {"type": "object"}
			}
		}
	}`

	req := esapi.IndicesCreateRequest{
		Index: es.indexName,
		Body:  strings.NewReader(mapping),
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return fmt.Errorf("创建索引失败: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("创建索引失败: %s", res.String())
	}

	log.Printf("ElasticSearch索引创建成功: %s", es.indexName)
	return nil
}

// IndexEmail 索引邮件到ElasticSearch
func (es *ElasticsearchClient) IndexEmail(email EmailDocument) error {
	if !es.enabled {
		return nil // 静默失败
	}

	jsonBody, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("邮件序列化失败: %v", err)
	}

	req := esapi.IndexRequest{
		Index:      es.indexName,
		DocumentID: email.ID,
		Body:       bytes.NewReader(jsonBody),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return fmt.Errorf("索引邮件失败: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("索引邮件失败: %s", res.String())
	}

	return nil
}

// DeleteEmail 从ElasticSearch删除邮件
func (es *ElasticsearchClient) DeleteEmail(emailID string) error {
	if !es.enabled {
		return nil
	}

	req := esapi.DeleteRequest{
		Index:      es.indexName,
		DocumentID: emailID,
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return fmt.Errorf("删除邮件失败: %v", err)
	}
	defer res.Body.Close()

	return nil
}

// SearchEmails 搜索邮件
func (es *ElasticsearchClient) SearchEmails(searchReq SearchRequest) (*SearchResult, error) {
	if !es.enabled {
		return &SearchResult{}, fmt.Errorf("ElasticSearch未启用")
	}

	// 构建查询
	query := es.buildQuery(searchReq)
	
	// 构建请求体
	requestBody := map[string]interface{}{
		"query": query,
		"size":  searchReq.Size,
		"from":  searchReq.From_,
		"sort": []map[string]interface{}{
			{
				"timestamp": map[string]string{"order": "desc"},
			},
		},
	}

	// 添加高亮
	if searchReq.Highlight && searchReq.Query != "" {
		requestBody["highlight"] = map[string]interface{}{
			"fields": map[string]interface{}{
				"subject": map[string]interface{}{},
				"body":    map[string]interface{}{
					"fragment_size": 150,
					"number_of_fragments": 3,
				},
				"from": map[string]interface{}{},
			},
			"pre_tags":  []string{"<mark>"},
			"post_tags": []string{"</mark>"},
		}
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("构建查询失败: %v", err)
	}

	// 执行搜索
	req := esapi.SearchRequest{
		Index: []string{es.indexName},
		Body:  bytes.NewReader(jsonBody),
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return nil, fmt.Errorf("搜索执行失败: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("搜索失败: %s", res.String())
	}

	// 解析结果
	var searchResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v", err)
	}

	return es.parseSearchResponse(searchResponse), nil
}

// buildQuery 构建ElasticSearch查询
func (es *ElasticsearchClient) buildQuery(searchReq SearchRequest) map[string]interface{} {
	var must []map[string]interface{}
	var filter []map[string]interface{}

	// 主查询
	if searchReq.Query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  searchReq.Query,
				"fields": []string{"subject^2", "body", "from", "to"},
				"type":   "best_fields",
				"fuzziness": "AUTO",
			},
		})
	}

	// 邮箱筛选
	if searchReq.Mailbox != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]string{"mailbox": searchReq.Mailbox},
		})
	}

	// 发件人筛选
	if searchReq.From != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"from": map[string]interface{}{
					"query":     searchReq.From,
					"fuzziness": "AUTO",
				},
			},
		})
	}

	// 收件人筛选
	if searchReq.To != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"to": map[string]interface{}{
					"query":     searchReq.To,
					"fuzziness": "AUTO",
				},
			},
		})
	}

	// 主题筛选
	if searchReq.Subject != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"subject": map[string]interface{}{
					"query":     searchReq.Subject,
					"fuzziness": "AUTO",
				},
			},
		})
	}

	// 日期范围筛选
	if searchReq.DateStart != "" || searchReq.DateEnd != "" {
		dateRange := map[string]interface{}{}
		if searchReq.DateStart != "" {
			dateRange["gte"] = searchReq.DateStart
		}
		if searchReq.DateEnd != "" {
			dateRange["lte"] = searchReq.DateEnd
		}
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"timestamp": dateRange,
			},
		})
	}

	// 附件筛选
	if searchReq.HasAttachment != nil {
		filter = append(filter, map[string]interface{}{
			"term": map[string]bool{"has_attachment": *searchReq.HasAttachment},
		})
	}

	// 读取状态筛选
	if searchReq.IsRead != nil {
		filter = append(filter, map[string]interface{}{
			"term": map[string]bool{"is_read": *searchReq.IsRead},
		})
	}

	// 优先级筛选
	if searchReq.Priority != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]string{"priority": searchReq.Priority},
		})
	}

	// 标签筛选
	if len(searchReq.Tags) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string][]string{"tags": searchReq.Tags},
		})
	}

	// 构建最终查询
	boolQuery := map[string]interface{}{}
	
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}

	// 如果没有任何条件，返回匹配所有
	if len(must) == 0 && len(filter) == 0 {
		return map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	return map[string]interface{}{
		"bool": boolQuery,
	}
}

// parseSearchResponse 解析ElasticSearch响应
func (es *ElasticsearchClient) parseSearchResponse(response map[string]interface{}) *SearchResult {
	result := &SearchResult{
		Emails: []EmailDocument{},
	}

	// 解析统计信息
	if hits, ok := response["hits"].(map[string]interface{}); ok {
		if total, ok := hits["total"].(map[string]interface{}); ok {
			if value, ok := total["value"].(float64); ok {
				result.Total = int64(value)
			}
		}

		// 解析邮件列表
		if hitsList, ok := hits["hits"].([]interface{}); ok {
			for _, hit := range hitsList {
				if hitMap, ok := hit.(map[string]interface{}); ok {
					if source, ok := hitMap["_source"].(map[string]interface{}); ok {
						email := EmailDocument{}
						
						// 解析邮件字段
						if id, ok := source["id"].(string); ok {
							email.ID = id
						}
						if mailbox, ok := source["mailbox"].(string); ok {
							email.Mailbox = mailbox
						}
						if from, ok := source["from"].(string); ok {
							email.From = from
						}
						if to, ok := source["to"].(string); ok {
							email.To = to
						}
						if subject, ok := source["subject"].(string); ok {
							email.Subject = subject
						}
						if body, ok := source["body"].(string); ok {
							email.Body = body
						}
						if timestamp, ok := source["timestamp"].(string); ok {
							if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
								email.Timestamp = t
							}
						}
						if hasAttachment, ok := source["has_attachment"].(bool); ok {
							email.HasAttachment = hasAttachment
						}
						if isRead, ok := source["is_read"].(bool); ok {
							email.IsRead = isRead
						}
						if priority, ok := source["priority"].(string); ok {
							email.Priority = priority
						}
						if size, ok := source["size"].(float64); ok {
							email.Size = int64(size)
						}
						
						result.Emails = append(result.Emails, email)
					}
				}
			}
		}
	}

	// 解析响应时间
	if took, ok := response["took"].(float64); ok {
		result.Took = int(took)
	}

	// 解析超时状态
	if timedOut, ok := response["timed_out"].(bool); ok {
		result.TimedOut = timedOut
	}

	return result
}

// BulkIndexEmails 批量索引邮件
func (es *ElasticsearchClient) BulkIndexEmails(emails []EmailDocument) error {
	if !es.enabled || len(emails) == 0 {
		return nil
	}

	var buf bytes.Buffer
	
	for _, email := range emails {
		// 添加索引操作
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": es.indexName,
				"_id":    email.ID,
			},
		}
		
		metaBytes, _ := json.Marshal(meta)
		buf.Write(metaBytes)
		buf.WriteByte('\n')
		
		// 添加文档数据
		docBytes, _ := json.Marshal(email)
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "true",
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return fmt.Errorf("批量索引失败: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("批量索引失败: %s", res.String())
	}

	log.Printf("批量索引完成: %d 封邮件", len(emails))
	return nil
}

// GetIndexStats 获取索引统计信息
func (es *ElasticsearchClient) GetIndexStats() (map[string]interface{}, error) {
	if !es.enabled {
		return nil, fmt.Errorf("ElasticSearch未启用")
	}

	req := esapi.IndicesStatsRequest{
		Index: []string{es.indexName},
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return nil, fmt.Errorf("获取索引统计失败: %v", err)
	}
	defer res.Body.Close()

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("解析统计信息失败: %v", err)
	}

	return stats, nil
}