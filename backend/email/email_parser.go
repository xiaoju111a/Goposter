package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// EmailContent 邮件内容结构
type EmailContent struct {
	Subject     string
	Body        string
	HTMLBody    string
	Date        string
	From        string
	To          []string
	CC          []string
	BCC         []string
	Attachments []AttachmentInfo
	Signature   string
	IsAutoReply bool
	Charset     string
	Headers     map[string]string
}

// AttachmentInfo 附件信息
type AttachmentInfo struct {
	Filename    string
	ContentType string
	Size        int64
	Content     []byte
	CID         string // Content-ID for inline attachments
	Disposition string // attachment or inline
}

// ParseEmailContent 解析原始邮件内容，提取主题、正文等
func ParseEmailContent(rawContent string) (subject, body, date string) {
	// 使用增强的邮件解析器
	emailContent := ParseEmailContentEnhanced(rawContent)
	return emailContent.Subject, emailContent.Body, emailContent.Date
}

// ParseEmailContentEnhanced 增强版邮件解析器
func ParseEmailContentEnhanced(rawContent string) *EmailContent {
	// 使用Go标准库解析邮件
	msg, err := mail.ReadMessage(strings.NewReader(rawContent))
	if err != nil {
		// 如果标准库解析失败，使用备用解析器
		return parseEmailFallbackEnhanced(rawContent)
	}
	
	emailContent := &EmailContent{
		Headers: make(map[string]string),
	}
	
	// 解析邮件头部
	parseEmailHeaders(msg.Header, emailContent)
	
	// 解析邮件正文和附件
	parseEmailBodyEnhanced(msg, emailContent)
	
	// 检测签名和自动回复
	detectSignatureAndAutoReply(emailContent)
	
	return emailContent
}

// parseEmailHeaders 解析邮件头部
func parseEmailHeaders(header mail.Header, emailContent *EmailContent) {
	// 解析主题
	if subjectHeader := header.Get("Subject"); subjectHeader != "" {
		emailContent.Subject = decodeHeader(subjectHeader)
	}
	
	// 解析日期
	if dateHeader := header.Get("Date"); dateHeader != "" {
		emailContent.Date = dateHeader
	} else {
		emailContent.Date = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	}
	
	// 解析发件人
	if fromHeader := header.Get("From"); fromHeader != "" {
		emailContent.From = decodeHeader(fromHeader)
	}
	
	// 解析收件人
	if toHeader := header.Get("To"); toHeader != "" {
		emailContent.To = parseAddressList(toHeader)
	}
	
	// 解析抄送
	if ccHeader := header.Get("Cc"); ccHeader != "" {
		emailContent.CC = parseAddressList(ccHeader)
	}
	
	// 解析密送
	if bccHeader := header.Get("Bcc"); bccHeader != "" {
		emailContent.BCC = parseAddressList(bccHeader)
	}
	
	// 解析字符编码
	if contentType := header.Get("Content-Type"); contentType != "" {
		emailContent.Charset = extractCharset(contentType)
	}
	
	// 存储所有头部
	for key, values := range header {
		if len(values) > 0 {
			emailContent.Headers[key] = values[0]
		}
	}
}

// parseAddressList 解析地址列表
func parseAddressList(addressList string) []string {
	addresses := strings.Split(addressList, ",")
	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addr = strings.TrimSpace(decodeHeader(addr))
		if addr != "" {
			result = append(result, addr)
		}
	}
	return result
}

// extractCharset 从 Content-Type 中提取字符编码
func extractCharset(contentType string) string {
	parts := strings.Split(contentType, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "charset=") {
			charset := strings.TrimPrefix(strings.ToLower(part), "charset=")
			charset = strings.Trim(charset, `"'`)
			return charset
		}
	}
	return "utf-8"
}

// decodeHeader 解码邮件头部（支持RFC 2047编码）
func decodeHeader(header string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		// 如果解码失败，尝试其他方法
		return decodeHeaderFallback(header)
	}
	return decoded
}

// decodeHeaderFallback 备用头部解码方法
func decodeHeaderFallback(header string) string {
	// 处理=?charset?encoding?encoded-text?=格式
	if !strings.Contains(header, "=?") {
		return header
	}
	
	// 简单的base64解码处理
	parts := strings.Split(header, "=?")
	if len(parts) < 2 {
		return header
	}
	
	for i := 1; i < len(parts); i++ {
		if strings.HasSuffix(parts[i], "?=") {
			encodedPart := strings.TrimSuffix(parts[i], "?=")
			sections := strings.Split(encodedPart, "?")
			if len(sections) >= 3 {
				encoding := strings.ToLower(sections[1])
				text := sections[2]
				
				if encoding == "b" {
					// Base64编码
					if decoded, err := base64.StdEncoding.DecodeString(text); err == nil {
						if utf8.Valid(decoded) {
							parts[i] = string(decoded)
						}
					}
				} else if encoding == "q" {
					// Quoted-printable编码
					parts[i] = decodeQuotedPrintable(text)
				}
			}
		}
	}
	
	return strings.Join(parts, "")
}

// decodeQuotedPrintable 解码Quoted-Printable编码
func decodeQuotedPrintable(s string) string {
	// 简单的quoted-printable解码
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "=20", " ")
	s = strings.ReplaceAll(s, "=0D=0A", "\n")
	s = strings.ReplaceAll(s, "=0A", "\n")
	
	// 处理=XX十六进制编码
	for i := 0; i < len(s)-2; i++ {
		if s[i] == '=' && len(s) > i+2 {
			hex := s[i+1 : i+3]
			if len(hex) == 2 {
				var b byte
				if _, err := fmt.Sscanf(hex, "%02X", &b); err == nil {
					s = s[:i] + string(b) + s[i+3:]
				}
			}
		}
	}
	
	return s
}

// parseEmailBodyEnhanced 增强版邮件正文解析
func parseEmailBodyEnhanced(msg *mail.Message, emailContent *EmailContent) {
	// 读取原始正文
	bodyBytes := make([]byte, 0, 1024*1024) // 1MB缓冲
	buffer := make([]byte, 1024)
	
	for {
		n, err := msg.Body.Read(buffer)
		if n > 0 {
			bodyBytes = append(bodyBytes, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}
	
	bodyContent := string(bodyBytes)
	
	// 检查Content-Type头部
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		emailContent.Body = strings.TrimSpace(bodyContent)
		return
	}
	
	// 解析multipart邮件
	if strings.Contains(strings.ToLower(contentType), "multipart") {
		parseMultipartContent(bodyContent, contentType, emailContent)
		return
	}
	
	// 处理单一内容类型
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		emailContent.HTMLBody = decodeContentWithCharset(bodyContent, msg.Header, emailContent.Charset)
		emailContent.Body = extractTextFromHTML(emailContent.HTMLBody)
	} else {
		emailContent.Body = decodeContentWithCharset(bodyContent, msg.Header, emailContent.Charset)
	}
}

// parseMultipartContent 解析多部分MIME内容
func parseMultipartContent(bodyContent, contentType string, emailContent *EmailContent) {
	// 提取boundary
	boundary := extractBoundary(contentType)
	if boundary == "" {
		// 如果没有boundary，使用原有的解析方法
		emailContent.Body = extractTextFromMultipart(bodyContent)
		return
	}
	
	// 创建multipart reader
	multipartReader := multipart.NewReader(strings.NewReader(bodyContent), boundary)
	
	for {
		part, err := multipartReader.NextPart()
		if err != nil {
			break
		}
		
		parseMultipartPart(part, emailContent)
		part.Close()
	}
	
	// 如果没有解析到文本内容，使用备用方法
	if emailContent.Body == "" && emailContent.HTMLBody == "" {
		emailContent.Body = extractTextFromMultipart(bodyContent)
	}
}

// parseMultipartPart 解析多部分中的单个部分
func parseMultipartPart(part *multipart.Part, emailContent *EmailContent) {
	header := part.Header
	contentType := header.Get("Content-Type")
	contentDisposition := header.Get("Content-Disposition")
	contentTransferEncoding := header.Get("Content-Transfer-Encoding")
	
	// 读取部分内容
	content, err := io.ReadAll(part)
	if err != nil {
		return
	}
	
	// 解码内容
	decodedContent := decodeTransferEncoding(string(content), contentTransferEncoding)
	
	// 处理不同的内容类型
	if strings.Contains(strings.ToLower(contentType), "text/plain") {
		if emailContent.Body == "" {
			emailContent.Body = decodeWithCharset(decodedContent, extractCharset(contentType))
		}
	} else if strings.Contains(strings.ToLower(contentType), "text/html") {
		if emailContent.HTMLBody == "" {
			emailContent.HTMLBody = decodeWithCharset(decodedContent, extractCharset(contentType))
		}
	} else if isAttachment(contentDisposition) || hasFilename(contentType, contentDisposition) {
		// 处理附件
		attachment := AttachmentInfo{
			Filename:    extractFilename(contentType, contentDisposition),
			ContentType: contentType,
			Size:        int64(len(content)),
			Content:     content,
			CID:         extractContentID(header.Get("Content-ID")),
			Disposition: extractDisposition(contentDisposition),
		}
		emailContent.Attachments = append(emailContent.Attachments, attachment)
	}
}

// extractBoundary 提取MIME边界
func extractBoundary(contentType string) string {
	parts := strings.Split(contentType, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "boundary=") {
			boundary := strings.TrimPrefix(part, "boundary=")
			boundary = strings.Trim(boundary, `"'`)
			return boundary
		}
	}
	return ""
}

// parseEmailBody 解析邮件正文（保持向后兼容）
func parseEmailBody(msg *mail.Message) string {
	emailContent := &EmailContent{
		Charset: "utf-8",
	}
	parseEmailBodyEnhanced(msg, emailContent)
	return emailContent.Body
}

// decodeContentWithCharset 根据字符集解码内容
func decodeContentWithCharset(content string, header mail.Header, charset string) string {
	// 先解码传输编码
	encoding := header.Get("Content-Transfer-Encoding")
	if encoding != "" {
		content = decodeTransferEncoding(content, encoding)
	}
	
	// 再按字符集解码
	return decodeWithCharset(content, charset)
}

// decodeWithCharset 根据字符集解码内容
func decodeWithCharset(content, charset string) string {
	if charset == "" || strings.ToLower(charset) == "utf-8" {
		return content
	}
	
	var decoder *transform.Transformer
	
	switch strings.ToLower(charset) {
	case "gb2312", "gbk", "gb18030":
		decoder = simplifiedchinese.GBK.NewDecoder()
	case "big5":
		decoder = traditionalchinese.Big5.NewDecoder()
	case "shift_jis", "shift-jis", "sjis":
		decoder = japanese.ShiftJIS.NewDecoder()
	case "euc-jp":
		decoder = japanese.EUCJP.NewDecoder()
	case "iso-2022-jp":
		decoder = japanese.ISO2022JP.NewDecoder()
	case "euc-kr":
		decoder = korean.EUCKR.NewDecoder()
	case "iso-8859-1":
		decoder = charmap.ISO8859_1.NewDecoder()
	case "iso-8859-2":
		decoder = charmap.ISO8859_2.NewDecoder()
	case "iso-8859-15":
		decoder = charmap.ISO8859_15.NewDecoder()
	case "windows-1252":
		decoder = charmap.Windows1252.NewDecoder()
	default:
		return content
	}
	
	if decoder != nil {
		if decoded, _, err := transform.String(*decoder, content); err == nil {
			return decoded
		}
	}
	
	return content
}

// decodeTransferEncoding 解码传输编码
func decodeTransferEncoding(content, encoding string) string {
	content = strings.TrimSpace(content)
	
	switch strings.ToLower(encoding) {
	case "base64":
		// 清理base64内容，移除换行符和空白字符
		cleanContent := strings.ReplaceAll(content, "\n", "")
		cleanContent = strings.ReplaceAll(cleanContent, "\r", "")
		cleanContent = strings.ReplaceAll(cleanContent, " ", "")
		cleanContent = strings.ReplaceAll(cleanContent, "\t", "")
		
		// 尝试标准base64解码
		if decoded, err := base64.StdEncoding.DecodeString(cleanContent); err == nil {
			return string(decoded)
		}
		
		// 如果标准解码失败，尝试添加填充
		if len(cleanContent)%4 != 0 {
			padding := 4 - (len(cleanContent) % 4)
			paddedContent := cleanContent + strings.Repeat("=", padding)
			if decoded, err := base64.StdEncoding.DecodeString(paddedContent); err == nil {
				return string(decoded)
			}
		}
		
		// 如果仍然失败，尝试URL安全的base64解码
		if decoded, err := base64.URLEncoding.DecodeString(cleanContent); err == nil {
			return string(decoded)
		}
	case "quoted-printable":
		return decodeQuotedPrintable(content)
	}
	return content
}

// parseEmailFallbackEnhanced 增强版备用邮件解析器
func parseEmailFallbackEnhanced(rawContent string) *EmailContent {
	emailContent := &EmailContent{
		Headers: make(map[string]string),
		Charset: "utf-8",
	}
	
	lines := strings.Split(rawContent, "\n")
	headerSection := true
	var bodyLines []string
	
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		
		if headerSection {
			// 空行表示头部结束，正文开始
			if strings.TrimSpace(line) == "" {
				headerSection = false
				continue
			}
			
			// 解析邮件头
			if strings.HasPrefix(strings.ToLower(line), "subject:") {
				emailContent.Subject = decodeHeader(strings.TrimSpace(line[8:]))
			} else if strings.HasPrefix(strings.ToLower(line), "date:") {
				emailContent.Date = strings.TrimSpace(line[5:])
			} else if strings.HasPrefix(strings.ToLower(line), "from:") {
				emailContent.From = decodeHeader(strings.TrimSpace(line[5:]))
			} else if strings.HasPrefix(strings.ToLower(line), "to:") {
				emailContent.To = parseAddressList(strings.TrimSpace(line[3:]))
			} else if strings.HasPrefix(strings.ToLower(line), "cc:") {
				emailContent.CC = parseAddressList(strings.TrimSpace(line[3:]))
			} else if strings.HasPrefix(strings.ToLower(line), "content-type:") {
				emailContent.Charset = extractCharset(strings.TrimSpace(line[13:]))
			}
			
			// 存储所有头部
			if colonIndex := strings.Index(line, ":"); colonIndex > 0 {
				key := strings.TrimSpace(line[:colonIndex])
				value := strings.TrimSpace(line[colonIndex+1:])
				emailContent.Headers[key] = value
			}
		} else {
			// 处理邮件正文
			bodyLines = append(bodyLines, line)
		}
	}
	
	// 处理multipart邮件
	body := strings.Join(bodyLines, "\n")
	emailContent.Body = extractTextFromMultipart(body)
	
	// 如果没有解析到日期，使用当前时间
	if emailContent.Date == "" {
		emailContent.Date = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	}
	
	// 检测签名和自动回复
	detectSignatureAndAutoReply(emailContent)
	
	return emailContent
}

// parseEmailFallback 备用邮件解析器（保持向后兼容）
func parseEmailFallback(rawContent string) (subject, body, date string) {
	emailContent := parseEmailFallbackEnhanced(rawContent)
	return emailContent.Subject, emailContent.Body, emailContent.Date
}

// extractTextFromMultipart 从multipart邮件中提取纯文本内容
func extractTextFromMultipart(content string) string {
	// 检查是否是multipart邮件
	if !strings.Contains(content, "Content-Type:") {
		return content
	}
	
	lines := strings.Split(content, "\n")
	var textContent []string
	inTextSection := false
	currentEncoding := ""
	skipNextEmpty := false
	
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		
		// 检查是否进入text/plain区域
		if strings.Contains(strings.ToLower(line), "content-type: text/plain") {
			inTextSection = true
			skipNextEmpty = true // 跳过content-type后的空行
			continue
		}
		
		// 检查编码方式
		if inTextSection && strings.HasPrefix(strings.ToLower(line), "content-transfer-encoding:") {
			currentEncoding = strings.TrimSpace(strings.ToLower(line[26:]))
			continue
		}
		
		// 检查是否离开text/plain区域
		if inTextSection && (strings.HasPrefix(line, "--") || 
			strings.Contains(strings.ToLower(line), "content-type:")) {
			if strings.HasPrefix(line, "--") || 
				!strings.Contains(strings.ToLower(line), "text/plain") {
				inTextSection = false
				currentEncoding = ""
				continue
			}
		}
		
		// 跳过空的内容类型行
		if inTextSection && strings.TrimSpace(line) == "" {
			if skipNextEmpty {
				skipNextEmpty = false
				continue
			}
		}
		
		// 收集文本内容
		if inTextSection && !strings.HasPrefix(strings.ToLower(line), "content-") {
			skipNextEmpty = false
			textContent = append(textContent, line)
		}
	}
	
	// 如果没有找到text/plain，尝试提取第一个文本区域
	if len(textContent) == 0 {
		return extractFirstTextContent(content)
	}
	
	result := strings.Join(textContent, "\n")
	
	// 根据编码方式解码内容
	if currentEncoding != "" {
		result = decodeTransferEncoding(result, currentEncoding)
	}
	
	result = strings.TrimSpace(result)
	
	// 移除首尾的空行
	lines = strings.Split(result, "\n")
	start, end := 0, len(lines)
	
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	
	if start < end {
		return strings.Join(lines[start:end], "\n")
	}
	
	return result
}

// extractFirstTextContent 提取第一个可能的文本内容
func extractFirstTextContent(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	foundContent := false
	
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		
		// 跳过邮件头和技术行
		if strings.HasPrefix(line, "Content-") ||
			strings.HasPrefix(line, "MIME-") ||
			strings.HasPrefix(line, "X-") ||
			strings.HasPrefix(line, "--") ||
			strings.Contains(line, "boundary=") {
			continue
		}
		
		// 如果是空行且还没找到内容，继续跳过
		if !foundContent && strings.TrimSpace(line) == "" {
			continue
		}
		
		// 找到第一行有内容的文本
		if !foundContent && strings.TrimSpace(line) != "" {
			foundContent = true
		}
		
		if foundContent {
			result = append(result, line)
		}
	}
	
	resultText := strings.TrimSpace(strings.Join(result, "\n"))
	
	// 检查是否为Base64编码内容
	if isLikelyBase64(resultText) {
		if decoded, err := base64.StdEncoding.DecodeString(resultText); err == nil {
			if utf8.Valid(decoded) {
				return string(decoded)
			}
		}
	}
	
	// 如果结果很短且看起来像base64，也尝试解码
	if len(resultText) > 0 && len(resultText) <= 1024 {
		// 尝试直接解码，即使不完全符合base64格式
		if decoded, err := base64.StdEncoding.DecodeString(resultText); err == nil {
			if utf8.Valid(decoded) && len(decoded) > 0 {
				decodedStr := string(decoded)
				// 如果解码后的内容包含可读字符，返回解码结果
				if strings.TrimSpace(decodedStr) != "" {
					return decodedStr
				}
			}
		}
	}
	
	// 强制尝试Base64解码，适用于所有可能的Base64字符串
	if strings.TrimSpace(resultText) != "" && isLikelyBase64(resultText) {
		if decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(resultText)); err == nil {
			if utf8.Valid(decoded) {
				decodedStr := string(decoded)
				if len(strings.TrimSpace(decodedStr)) > 0 {
					return decodedStr
				}
			}
		}
	}
	
	return resultText
}

// isLikelyBase64 检查字符串是否可能是Base64编码
func isLikelyBase64(s string) bool {
	s = strings.TrimSpace(s)
	// 基本长度检查，至少要4个字符
	if len(s) < 4 {
		return false
	}
	
	// 检查是否只包含Base64字符
	validChars := 0
	for _, char := range s {
		if (char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '+' || char == '/' || char == '=' {
			validChars++
		} else {
			return false
		}
	}
	
	// 检查结尾的等号数量（最多2个）
	equalCount := 0
	for i := len(s) - 1; i >= 0 && s[i] == '='; i-- {
		equalCount++
		if equalCount > 2 {
			return false
		}
	}
	
	// 如果超过75%的字符是有效Base64字符，并且长度合理，认为是Base64
	// 放宽长度要求，但增加字符比例检查
	if validChars >= len(s)*3/4 && len(s) >= 8 {
		return true
	}
	
	// 严格的Base64检查：长度是4的倍数
	return len(s)%4 == 0
}

// detectSignatureAndAutoReply 检测邮件签名和自动回复
func detectSignatureAndAutoReply(emailContent *EmailContent) {
	// 检测自动回复
	emailContent.IsAutoReply = isAutoReply(emailContent)
	
	// 检测并提取签名
	emailContent.Signature = extractSignature(emailContent.Body)
}

// isAutoReply 检查是否为自动回复邮件
func isAutoReply(emailContent *EmailContent) bool {
	// 检查主题中的自动回复关键词
	subject := strings.ToLower(emailContent.Subject)
	autoReplyKeywords := []string{
		"automatic reply", "auto reply", "auto-reply", "out of office",
		"vacation", "holiday", "absent", "away", "unavailable",
		"自动回复", "外出", "休假", "不在", "离开", "无法接收",
		"vacation message", "out-of-office", "autoreply",
	}
	
	for _, keyword := range autoReplyKeywords {
		if strings.Contains(subject, keyword) {
			return true
		}
	}
	
	// 检查邮件头部
	for key, value := range emailContent.Headers {
		key = strings.ToLower(key)
		value = strings.ToLower(value)
		
		if key == "auto-submitted" && value != "no" {
			return true
		}
		if key == "x-auto-response-suppress" {
			return true
		}
		if key == "precedence" && (value == "bulk" || value == "auto_reply") {
			return true
		}
	}
	
	// 检查正文内容
	body := strings.ToLower(emailContent.Body)
	bodyKeywords := []string{
		"this is an automated", "automatic response", "out of office",
		"vacation message", "i am currently away", "i will be away",
		"这是一封自动回复", "自动回复邮件", "外出通知", "我现在不在",
	}
	
	for _, keyword := range bodyKeywords {
		if strings.Contains(body, keyword) {
			return true
		}
	}
	
	return false
}

// extractSignature 提取邮件签名
func extractSignature(body string) string {
	lines := strings.Split(body, "\n")
	
	// 寻找签名分隔符
	signatureStart := -1
	signaturePatterns := []string{
		"-- ",         // 标准签名分隔符
		"--",          // 简化分隔符
		"___",         // 下划线分隔符
		"-----",       // 横线分隔符
		"Best regards,",
		"Sincerely,",
		"Thanks,",
		"谢谢",
		"此致",
		"敬礼",
		"祝好",
	}
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 检查标准签名分隔符
		if trimmed == "-- " || trimmed == "--" {
			signatureStart = i
			break
		}
		
		// 检查其他签名模式
		for _, pattern := range signaturePatterns {
			if strings.Contains(trimmed, pattern) {
				signatureStart = i
				break
			}
		}
		
		if signatureStart != -1 {
			break
		}
	}
	
	// 如果没有找到明确的签名分隔符，尝试从末尾开始查找
	if signatureStart == -1 {
		// 从后往前查找，寻找可能的签名开始位置
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			
			// 检查是否包含联系信息
			if containsContactInfo(line) {
				signatureStart = i
				break
			}
		}
	}
	
	if signatureStart != -1 && signatureStart < len(lines) {
		signature := strings.Join(lines[signatureStart:], "\n")
		return strings.TrimSpace(signature)
	}
	
	return ""
}

// containsContactInfo 检查是否包含联系信息
func containsContactInfo(line string) bool {
	contactPatterns := []string{
		"@", "tel:", "phone:", "mobile:", "fax:", "email:",
		"www.", "http://", "https://", "+86", "+1", "+44",
		"电话", "手机", "邮箱", "传真", "地址", "公司",
		"Tel:", "Phone:", "Email:", "Mobile:", "Fax:",
	}
	
	for _, pattern := range contactPatterns {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			return true
		}
	}
	
	return false
}

// extractTextFromHTML 从HTML中提取纯文本
func extractTextFromHTML(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}
	
	// 移除脚本和样式标签
	htmlContent = removeHTMLTags(htmlContent, []string{"script", "style", "head"})
	
	// 移除HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(htmlContent, "")
	
	// 解码HTML实体
	text = decodeHTMLEntities(text)
	
	// 清理多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	return text
}

// removeHTMLTags 移除指定的HTML标签及其内容
func removeHTMLTags(html string, tags []string) string {
	for _, tag := range tags {
		re := regexp.MustCompile(`(?i)<` + tag + `[^>]*>.*?</` + tag + `>`)
		html = re.ReplaceAllString(html, "")
	}
	return html
}

// decodeHTMLEntities 解码HTML实体
func decodeHTMLEntities(text string) string {
	entities := map[string]string{
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&apos;":  "'",
		"&nbsp;":  " ",
		"&copy;":  "©",
		"&reg;":   "®",
		"&trade;": "™",
		"&hellip;": "…",
		"&mdash;": "—",
		"&ndash;": "–",
		"&laquo;": "«",
		"&raquo;": "»",
		"&bull;":  "•",
	}
	
	for entity, replacement := range entities {
		text = strings.ReplaceAll(text, entity, replacement)
	}
	
	// 处理数字实体 &#xxx; 和 &#xXXX;
	re := regexp.MustCompile(`&#(\d+);`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		numStr := match[2 : len(match)-1]
		if num, err := strconv.Atoi(numStr); err == nil {
			return string(rune(num))
		}
		return match
	})
	
	re = regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		hexStr := match[3 : len(match)-1]
		if num, err := strconv.ParseInt(hexStr, 16, 32); err == nil {
			return string(rune(num))
		}
		return match
	})
	
	return text
}

// isAttachment 检查是否为附件
func isAttachment(contentDisposition string) bool {
	return strings.Contains(strings.ToLower(contentDisposition), "attachment")
}

// hasFilename 检查是否有文件名
func hasFilename(contentType, contentDisposition string) bool {
	return extractFilename(contentType, contentDisposition) != ""
}

// extractFilename 提取文件名
func extractFilename(contentType, contentDisposition string) string {
	// 首先从Content-Disposition中提取
	if contentDisposition != "" {
		if filename := extractFilenameFromHeader(contentDisposition); filename != "" {
			return filename
		}
	}
	
	// 然后从Content-Type中提取
	if contentType != "" {
		if filename := extractFilenameFromHeader(contentType); filename != "" {
			return filename
		}
	}
	
	return ""
}

// extractFilenameFromHeader 从邮件头中提取文件名
func extractFilenameFromHeader(header string) string {
	// 处理 filename= 或 filename*= 格式
	patterns := []string{
		`filename\*?=\s*"([^"]+)"`,
		`filename\*?=\s*'([^']+)'`,
		`filename\*?=\s*([^;\s]+)`,
		`name\s*=\s*"([^"]+)"`,
		`name\s*=\s*'([^']+)'`,
		`name\s*=\s*([^;\s]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if matches := re.FindStringSubmatch(header); len(matches) > 1 {
			filename := matches[1]
			// 解码文件名
			if decoded := decodeFilename(filename); decoded != "" {
				return decoded
			}
			return filename
		}
	}
	
	return ""
}

// decodeFilename 解码文件名
func decodeFilename(filename string) string {
	// 处理RFC 2231编码 (如: UTF-8''filename)
	if strings.Contains(filename, "''") {
		parts := strings.Split(filename, "''")
		if len(parts) >= 2 {
			charset := parts[0]
			encodedName := parts[1]
			
			// URL解码
			if decoded, err := textproto.CanonicalMIMEHeaderKey(encodedName); err == nil {
				return decoded
			}
			
			// 尝试字符集解码
			if charset != "" {
				return decodeWithCharset(encodedName, charset)
			}
			
			return encodedName
		}
	}
	
	// 处理MIME编码
	if strings.Contains(filename, "=?") {
		return decodeHeader(filename)
	}
	
	return filename
}

// extractContentID 提取Content-ID
func extractContentID(contentID string) string {
	// 移除< >包围符号
	contentID = strings.Trim(contentID, "<>")
	return contentID
}

// extractDisposition 提取内容处理方式
func extractDisposition(contentDisposition string) string {
	if contentDisposition == "" {
		return "attachment"
	}
	
	parts := strings.Split(contentDisposition, ";")
	if len(parts) > 0 {
		disposition := strings.TrimSpace(strings.ToLower(parts[0]))
		if disposition == "inline" || disposition == "attachment" {
			return disposition
		}
	}
	
	return "attachment"
}

// ExtractEmbeddedContent 提取嵌入式内容（图片、链接等）
func ExtractEmbeddedContent(emailContent *EmailContent) map[string][]string {
	embeddedContent := make(map[string][]string)
	
	// 从HTML正文中提取
	if emailContent.HTMLBody != "" {
		embeddedContent["images"] = extractImages(emailContent.HTMLBody)
		embeddedContent["links"] = extractLinks(emailContent.HTMLBody)
	}
	
	// 从纯文本正文中提取链接
	if emailContent.Body != "" {
		textLinks := extractLinksFromText(emailContent.Body)
		embeddedContent["links"] = append(embeddedContent["links"], textLinks...)
	}
	
	// 从附件中提取内联图片
	inlineImages := make([]string, 0)
	for _, attachment := range emailContent.Attachments {
		if attachment.Disposition == "inline" && strings.HasPrefix(attachment.ContentType, "image/") {
			if attachment.CID != "" {
				inlineImages = append(inlineImages, "cid:"+attachment.CID)
			}
		}
	}
	embeddedContent["inline_images"] = inlineImages
	
	return embeddedContent
}

// extractImages 从HTML中提取图片
func extractImages(htmlContent string) []string {
	var images []string
	
	// 提取img标签的src属性
	re := regexp.MustCompile(`(?i)<img[^>]+src\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			images = append(images, match[1])
		}
	}
	
	return images
}

// extractLinks 从HTML中提取链接
func extractLinks(htmlContent string) []string {
	var links []string
	
	// 提取a标签的href属性
	re := regexp.MustCompile(`(?i)<a[^>]+href\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}
	
	return links
}

// extractLinksFromText 从纯文本中提取链接
func extractLinksFromText(text string) []string {
	var links []string
	
	// 提取HTTP/HTTPS链接
	re := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	matches := re.FindAllString(text, -1)
	links = append(links, matches...)
	
	// 提取邮件地址
	re = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	matches = re.FindAllString(text, -1)
	for _, match := range matches {
		links = append(links, "mailto:"+match)
	}
	
	return links
}

// GetAttachmentMetadata 获取附件元数据
func GetAttachmentMetadata(emailContent *EmailContent) []map[string]interface{} {
	metadata := make([]map[string]interface{}, 0, len(emailContent.Attachments))
	
	for _, attachment := range emailContent.Attachments {
		meta := map[string]interface{}{
			"filename":     attachment.Filename,
			"content_type": attachment.ContentType,
			"size":         attachment.Size,
			"disposition":  attachment.Disposition,
			"is_inline":    attachment.Disposition == "inline",
			"content_id":   attachment.CID,
		}
		
		// 添加文件扩展名
		if attachment.Filename != "" {
			if dotIndex := strings.LastIndex(attachment.Filename, "."); dotIndex != -1 {
				meta["extension"] = strings.ToLower(attachment.Filename[dotIndex+1:])
			}
		}
		
		// 添加MIME类型信息
		if attachment.ContentType != "" {
			mainType := strings.Split(attachment.ContentType, "/")[0]
			meta["main_type"] = mainType
			meta["is_image"] = mainType == "image"
			meta["is_document"] = mainType == "application" || mainType == "text"
		}
		
		metadata = append(metadata, meta)
	}
	
	return metadata
}