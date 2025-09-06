# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| > 1.0.x | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability within dee, please send an email to security@example.com. All security vulnerabilities will be promptly addressed.

**Please do not report security vulnerabilities through public GitHub issues.**

### What to include

When reporting a vulnerability, please include:

- A description of the vulnerability
- Steps to reproduce the issue
- Possible impact of the vulnerability
- Any suggested fixes (if you have them)

### Response Timeline

- We will acknowledge receipt of your vulnerability report within 48 hours
- We will provide a more detailed response within 7 days
- We will work on a fix and release schedule based on the severity of the issue

## Security Considerations

### Environment Variable Storage

- dee stores environment variables in plain text YAML files in `~/.dee/`
- These files should be protected with appropriate file system permissions (0600)
- Consider using encrypted storage solutions for highly sensitive environments

### Schema Validation

- Always validate schemas before using them in production
- Be cautious with regex patterns that could lead to ReDoS attacks
- Test inheritance chains to prevent circular dependencies

### Network Security

- The install script downloads binaries over HTTPS
- Always verify checksums when downloading manually
- Consider using internal mirrors for corporate environments

### Best Practices

1. **File Permissions**: Ensure your `~/.dee/` directory has restricted permissions
2. **Backup Security**: If backing up configurations, ensure backups are encrypted
3. **Access Control**: Limit access to configuration files containing sensitive data
4. **Regular Updates**: Keep dee updated to the latest version
5. **Environment Separation**: Use separate schemas for different security contexts

## Vulnerability Disclosure Policy

We follow responsible disclosure principles:

1. Report vulnerabilities privately first
2. We will work with you to understand and address the issue
3. We will credit you in the security advisory (unless you prefer to remain anonymous)
4. We may request that you delay public disclosure until a fix is available