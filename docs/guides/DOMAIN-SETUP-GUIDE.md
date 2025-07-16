# 域名设置指南

要使您的 YgoCard Mail 服务器能够正确接收来自互联网的邮件，必须正确配置您域名的 DNS 记录。本文档提供了必要的配置步骤和验证方法。

---

## 1. 核心 DNS 记录

您需要在您的域名注册商（如 GoDaddy, Cloudflare, 阿里云等）的管理面板中，为您的域名（本文以 `ygocard.org` 为例）添加以下四条核心 DNS 记录。

| 类型  | 名称 (Host/Name)           | 值 / 目标 (Value/Target)     | 优先级 (Priority) |
| :---- | :------------------------- | :--------------------------- | :---------------- |
| **A** | `mail`                     | `[你的服务器公网IP]`         | -                 |
| **MX**| `@` 或 `ygocard.org`       | `mail.ygocard.org`           | 10                |
| **TXT**| `@` 或 `ygocard.org`       | `"v=spf1 a mx ~all"`         | -                 |
| **TXT**| `default._domainkey`       | `"v=DKIM1; k=rsa; p=[你的公钥]"` | -                 |

### **记录说明:**

-   **A 记录:** 将 `mail.ygocard.org` 这个子域名指向您服务器的公网 IP 地址。这是邮件服务器的地址。
-   **MX 记录:** 告诉其他邮件服务器，所有发送到 `@ygocard.org` 后缀的邮件��都应该被投递到 `mail.ygocard.org` 这台服务器上。优先级 `10` 是一个标准值。
-   **TXT (SPF) 记录:** 发件人策略框架 (SPF) 是一种邮件认证标准，用于防止他人伪造您的域名发送垃圾邮件。此记录声明了只有您的服务器 (`a` 和 `mx` 记录指向的地址)才有权为您的域名发送邮件。
-   **TXT (DKIM) 记录:** 域名密钥识别邮件 (DKIM) 为您的外发邮件添加数字签名，收件方服务器可以通过查询此 DNS 记录中的公钥来验证邮件的真实性和完整性，确保邮件未被篡改。
    -   `[你的公钥]` 部分需要替换为您在 YgoCard Mail 中生成的 DKIM 公钥。

> 📚 **关于 SPF 和 DKIM 的更详细信息，请参阅 [邮件认证指南](./DNS-EMAIL-AUTH-GUIDE.md)。**

---

## 2. 如何验证 DNS 配置

在您添加或修改 DNS 记录后，通常需要一些时间才能全球生效（从几分钟到几小时不等）。您可以使用以下命令来验证您的配置是否正确。

### **验证 A 记录**
```bash
# 在 Linux/macOS/Windows WSL 中运行
dig A mail.ygocard.org +short
```
**预期输出:**
```
[你的服务器公网IP]
```

### **验证 MX 记录**
```bash
dig MX ygocard.org +short
```
**预期输出:**
```
10 mail.ygocard.org.
```

### **验证 SPF (TXT) 记录**
```bash
dig TXT ygocard.org +short
```
**预期输出:**
```
"v=spf1 a mx ~all"
```

### **验证 DKIM (TXT) 记录**
```bash
dig TXT default._domainkey.ygocard.org +short
```
**预期输出:**
```
"v=DKIM1; k=rsa; p=[你的公钥]"
```

---

## 3. 常见问题

-   **为什么我收不到邮件?**
    -   最常见的原因是 **MX 记录** 配置错误或尚未生效。请使用 `dig` 命令仔细检查。
    -   确保您的服务器防火墙已放行 SMTP 端口 (通常是 25, 465, 587)。

-   **为什么我发送的邮件被标记为垃圾邮件?**
    -   检查您的 **SPF** 和 **DKIM** 记录是否正确配置。这两条记录对于建立域名信誉至关重要。
    -   确保您的服务器 IP 没有被列入常见的垃圾邮件黑名单 (RBL)。

-   **`@` 和 `mail` 代表什么?**
    -   在 DNS 配置中, `@` 通常是您根域名 (`ygocard.org`) 的简写。
    -   `mail` 是一个子域名，所以完整的地址是 `mail.ygocard.org`。