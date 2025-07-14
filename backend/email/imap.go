package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type IMAPServer struct {
	mailServer *MailServer
}

func NewIMAPServer(ms *MailServer) *IMAPServer {
	return &IMAPServer{mailServer: ms}
}

func (imap *IMAPServer) StartIMAPServer(port string) {
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal("Failed to start IMAP server:", err)
	}
	defer listener.Close()
	
	log.Printf("IMAP server listening on 0.0.0.0:%s", port)
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept IMAP connection:", err)
			continue
		}
		
		log.Printf("New IMAP connection from: %s", conn.RemoteAddr())
		go imap.HandleIMAPConnection(conn)
	}
}

func (imap *IMAPServer) HandleIMAPConnection(conn net.Conn) {
	defer conn.Close()
	
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	
	// IMAP greeting
	writer.WriteString("* OK [CAPABILITY IMAP4rev1] YgoCard IMAP Server ready\r\n")
	writer.Flush()
	
	var authenticated bool
	var selectedMailbox string
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		
		tag := parts[0]
		command := strings.ToUpper(parts[1])
		
		switch command {
		case "CAPABILITY":
			writer.WriteString("* CAPABILITY IMAP4rev1 LOGIN\r\n")
			writer.WriteString(tag + " OK CAPABILITY completed\r\n")
			
		case "LOGIN":
			if len(parts) >= 4 {
				username := strings.Trim(parts[2], "\"")
				password := strings.Trim(parts[3], "\"")
				
				// 简单认证 - 任何用户名密码都接受
				if imap.authenticate(username, password) {
					authenticated = true
					writer.WriteString(tag + " OK LOGIN completed\r\n")
				} else {
					writer.WriteString(tag + " NO LOGIN failed\r\n")
				}
			} else {
				writer.WriteString(tag + " BAD LOGIN command incomplete\r\n")
			}
			
		case "LIST":
			if !authenticated {
				writer.WriteString(tag + " NO Please login first\r\n")
			} else {
				// 列出所有邮箱
				mailboxes := imap.mailServer.GetAllMailboxes()
				for _, mailbox := range mailboxes {
					writer.WriteString(fmt.Sprintf("* LIST () \".\" \"%s\"\r\n", mailbox))
				}
				writer.WriteString(tag + " OK LIST completed\r\n")
			}
			
		case "SELECT":
			if !authenticated {
				writer.WriteString(tag + " NO Please login first\r\n")
			} else if len(parts) >= 3 {
				mailbox := strings.Trim(parts[2], "\"")
				emails := imap.mailServer.GetEmails(mailbox)
				selectedMailbox = mailbox
				
				writer.WriteString(fmt.Sprintf("* %d EXISTS\r\n", len(emails)))
				writer.WriteString("* 0 RECENT\r\n")
				writer.WriteString("* OK [UIDVALIDITY 1] UIDs valid\r\n")
				writer.WriteString("* FLAGS (\\Seen \\Answered \\Flagged \\Deleted \\Draft)\r\n")
				writer.WriteString(tag + " OK [READ-WRITE] SELECT completed\r\n")
			} else {
				writer.WriteString(tag + " BAD SELECT command incomplete\r\n")
			}
			
		case "FETCH":
			if !authenticated {
				writer.WriteString(tag + " NO Please login first\r\n")
			} else if selectedMailbox == "" {
				writer.WriteString(tag + " NO No mailbox selected\r\n")
			} else {
				imap.handleFetch(writer, tag, parts, selectedMailbox)
			}
			
		case "LOGOUT":
			writer.WriteString("* BYE YgoCard IMAP Server logging out\r\n")
			writer.WriteString(tag + " OK LOGOUT completed\r\n")
			writer.Flush()
			return
			
		default:
			writer.WriteString(tag + " BAD Command not recognized\r\n")
		}
		
		writer.Flush()
	}
}

func (imap *IMAPServer) authenticate(username, password string) bool {
	// 简单认证策略：
	// 1. 接受任何包含@domain的用户名
	// 2. 密码简单验证或接受默认密码
	domain := imap.mailServer.domain
	return strings.Contains(username, "@"+domain) || username == "admin"
}

func (imap *IMAPServer) handleFetch(writer *bufio.Writer, tag string, parts []string, mailbox string) {
	if len(parts) < 4 {
		writer.WriteString(tag + " BAD FETCH command incomplete\r\n")
		return
	}
	
	// 解析消息序列号
	seqStr := parts[2]
	attrs := strings.Join(parts[3:], " ")
	
	emails := imap.mailServer.GetEmails(mailbox)
	
	if seqStr == "*" || seqStr == "1:*" {
		// 获取所有邮件
		for i, email := range emails {
			msgNum := i + 1
			imap.writeFetchResponse(writer, msgNum, email, attrs)
		}
	} else {
		// 解析具体的序列号
		if seq, err := strconv.Atoi(seqStr); err == nil && seq > 0 && seq <= len(emails) {
			imap.writeFetchResponse(writer, seq, emails[seq-1], attrs)
		}
	}
	
	writer.WriteString(tag + " OK FETCH completed\r\n")
}

func (imap *IMAPServer) writeFetchResponse(writer *bufio.Writer, msgNum int, email Email, attrs string) {
	attrs = strings.ToUpper(attrs)
	
	response := fmt.Sprintf("* %d FETCH (", msgNum)
	
	if strings.Contains(attrs, "ENVELOPE") {
		date := email.Date
		if date == "" {
			date = time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
		}
		
		envelope := fmt.Sprintf("ENVELOPE (\"%s\" \"%s\" ((\"%s\" NIL \"%s\" \"%s\")) ((\"%s\" NIL \"%s\" \"%s\")) NIL NIL NIL NIL)",
			date, email.Subject,
			email.From, getLocalPart(email.From), getDomainPart(email.From),
			email.To, getLocalPart(email.To), getDomainPart(email.To))
		response += envelope
	}
	
	if strings.Contains(attrs, "BODY") || strings.Contains(attrs, "RFC822") {
		if strings.Contains(response, "ENVELOPE") {
			response += " "
		}
		bodySize := len(email.Body)
		response += fmt.Sprintf("BODY {%d}\r\n%s", bodySize, email.Body)
	}
	
	if strings.Contains(attrs, "FLAGS") {
		if !strings.HasSuffix(response, "(") {
			response += " "
		}
		response += "FLAGS (\\Seen)"
	}
	
	response += ")\r\n"
	writer.WriteString(response)
}

func getLocalPart(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}

func getDomainPart(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}