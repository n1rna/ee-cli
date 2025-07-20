---
name: Bug report
about: Create a report to help us improve
title: '[BUG] '
labels: ['bug']
assignees: ''

---

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Run command '...'
2. With config '...'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Screenshots**
If applicable, add screenshots to help explain your problem.

**Environment (please complete the following information):**
 - OS: [e.g. Ubuntu 20.04, macOS 12.0, Windows 11]
 - menv Version: [e.g. v1.0.0] (run `menv version`)
 - Go Version: [e.g. 1.21.0] (if building from source)

**Configuration Files**
If relevant, please provide your schema and config files:

```yaml
# schema.yaml
name: example-schema
variables:
  - name: API_KEY
    type: string
    required: true
```

```yaml
# development.yaml
project_name: myproject
env_name: development
schema: example-schema
values:
  API_KEY: "secret"
```

**Additional context**
Add any other context about the problem here.