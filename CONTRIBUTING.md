# 🤝 Contributing to Foodie

First off, thank you for considering contributing to Foodie! It's people like you that make Foodie such a great tool.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How Can I Contribute?](#how-can-i-contribute)
- [Style Guidelines](#style-guidelines)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)

---

## 📜 Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

### Our Standards

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

---

## 🚀 Getting Started

### Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Node.js 18+ (for Cloudflare deployment)
- Git

### Setup Development Environment

```bash
# 1. Fork the repository on GitHub
# 2. Clone your fork
git clone https://github.com/medissaoui711/Foodie-Delivery-Clone.git
cd foodie

# 3. Add upstream remote
git remote add upstream https://github.com/medissaoui711/Foodie-Delivery-Clone.git

# 4. Create a branch
git checkout -b feature/my-feature

# 5. Setup environment
cp .env.example .env

# 6. Start services
docker-compose up -d postgres redis

# 7. Run tests
cd backend
go test ./...
```

---

## 💡 How Can I Contribute?

### 🐛 Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates.

**How to submit a good bug report:**

1. **Use the GitHub issue tracker**
2. **Describe the bug**
   - Clear and descriptive title
   - Steps to reproduce
   - Expected behavior vs actual behavior
   - Screenshots if applicable
3. **Provide context**
   - Go version: `go version`
   - Operating system
   - Database version
   - Any relevant logs

**Bug report template:**

```markdown
**Description:**
Clear description of the bug

**Steps to Reproduce:**
1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

**Expected behavior:**
What you expected to happen

**Actual behavior:**
What actually happened

**Environment:**
- Go version: 1.23
- OS: Ubuntu 22.04
- Database: PostgreSQL 16
```

### ✨ Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues.

**How to suggest an enhancement:**

1. Use a clear and descriptive title
2. Provide a step-by-step description of the suggested enhancement
3. Provide specific examples to demonstrate the enhancement
4. Explain why this enhancement would be useful

### 📝 Pull Requests

1. Update the README.md with details of changes if applicable
2. Update the documentation
3. Add tests for your changes
4. Ensure all tests pass
5. Reference the issue number in your PR description

---

## 🎨 Style Guidelines

### Go Code Style

We follow the standard Go conventions:

```go
// Good
package models

import (
    "time"
    
    "github.com/gofiber/fiber/v2"
)

// User represents a user in the system
type User struct {
    ID        string    `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// GetByID retrieves a user by their ID
func (u *User) GetByID(id string) (*User, error) {
    // implementation
}
```

**Key points:**

- Use `gofmt` for formatting
- Use meaningful variable names
- Add comments for exported functions
- Keep functions small and focused
- Handle errors explicitly
- Use interfaces for testability

### Frontend Code Style

```javascript
// Good
const API_URL = 'http://localhost:8080/api/v1';

/**
 * Fetch restaurants from the API
 * @param {string} category - Restaurant category
 * @returns {Promise<Array>} - Array of restaurants
 */
async function fetchRestaurants(category = 'all') {
    try {
        const response = await fetch(`${API_URL}/restaurants?category=${category}`);
        if (!response.ok) {
            throw new Error('Failed to fetch restaurants');
        }
        return await response.json();
    } catch (error) {
        console.error('Error fetching restaurants:', error);
        throw error;
    }
}
```

---

## 📝 Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, missing semi colons, etc)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Build process or auxiliary tool changes

### Examples

```bash
# Good commit messages
git commit -m "feat(auth): add JWT token refresh endpoint"
git commit -m "fix(orders): resolve order cancellation bug"
git commit -m "docs(readme): update API documentation"
git commit -m "test(auth): add unit tests for login handler"
git commit -m "refactor(database): optimize connection pooling"
```

---

## 🔄 Pull Request Process

1. **Update your fork**

   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Make your changes**
   - Write clean, documented code
   - Add tests
   - Update documentation

3. **Run tests locally**

   ```bash
   cd backend
   go test ./...
   go vet ./...
   golangci-lint run
   ```

4. **Push to your fork**

   ```bash
   git push origin feature/my-feature
   ```

5. **Create Pull Request**
   - Use a clear title
   - Reference related issues
   - Describe what changed and why
   - Add screenshots for UI changes

6. **Address review comments**
   - Be responsive to feedback
   - Make requested changes
   - Push additional commits

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## How Has This Been Tested?
- [ ] Unit tests
- [ ] Integration tests
- [ ] Manual testing

## Checklist:
- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
```

---

## 🏷️ Issue Labels

We use labels to organize issues:

- `bug`: Something isn't working
- `enhancement`: New feature or request
- `documentation`: Documentation related
- `good first issue`: Good for newcomers
- `help wanted`: Extra attention is needed
- `priority/high`: High priority issue
- `priority/low`: Low priority issue

---

## 🙏 Recognition

Contributors will be:

- Listed in the README.md
- Mentioned in release notes
- Added to our contributors page

---

## ❓ Questions?

If you have questions, feel free to:

- Open an issue
- Join our Discord community
- Email us at: <contacteinfo71@gmail.com>

Thank you for contributing! 🎉
