package main

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// ParseEmailContent 解析原始邮件内容，提取主题、正文等
func ParseEmailContent(rawContent string) (subject, body, date string) {
	// 使用Go标准库解析邮件
	msg, err := mail.ReadMessage(strings.NewReader(rawContent))
	if err != nil {
		// 如果标准库解析失败，使用备用解析器
		return parseEmailFallback(rawContent)
	}
	
	// 解析并解码主题
	if subjectHeader := msg.Header.Get("Subject"); subjectHeader != "" {
		subject = decodeHeader(subjectHeader)
	}
	
	// 解析日期
	if dateHeader := msg.Header.Get("Date"); dateHeader != "" {
		date = dateHeader
	} else {
		date = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	}
	
	// 解析邮件正文
	body = parseEmailBody(msg)
	
	return subject, body, date
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

// parseEmailBody 解析邮件正文
func parseEmailBody(msg *mail.Message) string {
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
		return strings.TrimSpace(bodyContent)
	}
	
	// 解析multipart邮件
	if strings.Contains(strings.ToLower(contentType), "multipart") {
		return extractTextFromMultipart(bodyContent)
	}
	
	// 检查Content-Transfer-Encoding
	encoding := msg.Header.Get("Content-Transfer-Encoding")
	if encoding != "" {
		bodyContent = decodeTransferEncoding(bodyContent, encoding)
	}
	
	// 无论是否有编码头，都检查内容是否为Base64并尝试解码
	trimmedContent := strings.TrimSpace(bodyContent)
	if isLikelyBase64(trimmedContent) {
		if decoded, err := base64.StdEncoding.DecodeString(trimmedContent); err == nil {
			if utf8.Valid(decoded) {
				// 只有解码后的内容比原内容更"可读"才使用解码结果
				decodedStr := string(decoded)
				if len(strings.TrimSpace(decodedStr)) > 0 && !isLikelyBase64(decodedStr) {
					bodyContent = decodedStr
				}
			}
		}
	}
	
	return strings.TrimSpace(bodyContent)
}

// decodeTransferEncoding 解码传输编码
func decodeTransferEncoding(content, encoding string) string {
	content = strings.TrimSpace(content)
	
	switch strings.ToLower(encoding) {
	case "base64":
		// 尝试标准base64解码
		if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
			if utf8.Valid(decoded) {
				return string(decoded)
			}
		}
		
		// 如果标准解码失败，尝试添加填充
		if len(content)%4 != 0 {
			padding := 4 - (len(content) % 4)
			paddedContent := content + strings.Repeat("=", padding)
			if decoded, err := base64.StdEncoding.DecodeString(paddedContent); err == nil {
				if utf8.Valid(decoded) {
					return string(decoded)
				}
			}
		}
	case "quoted-printable":
		return decodeQuotedPrintable(content)
	}
	return content
}

// parseEmailFallback 备用邮件解析器
func parseEmailFallback(rawContent string) (subject, body, date string) {
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
				subject = decodeHeader(strings.TrimSpace(line[8:]))
			} else if strings.HasPrefix(strings.ToLower(line), "date:") {
				date = strings.TrimSpace(line[5:])
			}
		} else {
			// 处理邮件正文
			bodyLines = append(bodyLines, line)
		}
	}
	
	// 处理multipart邮件
	body = strings.Join(bodyLines, "\n")
	body = extractTextFromMultipart(body)
	
	// 如果没有解析到日期，使用当前时间
	if date == "" {
		date = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	}
	
	return subject, body, date
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