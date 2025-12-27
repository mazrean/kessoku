# Task Completion Rules

## Mandatory Quality Checks
After making any Go code changes, ALWAYS run:
1. **Lint check**: `go tool tools lint ./...` - Fix all linting errors
2. **Test suite**: `go test -v ./...` - Ensure all tests pass

These checks are mandatory for maintaining code quality standards.

## Git Commit Guidelines
- Always create git commits at appropriate granular units for code changes
- Each commit should represent a logical, atomic change
- Write clear, descriptive commit messages that explain the purpose of the change

## Documentation Maintenance
- **ALWAYS update documentation when making code or feature changes**
- Update CLAUDE.md when architecture, commands, or development workflows change
- Update README.md when user-facing features or installation instructions change
- Keep documentation in sync with actual implementation
- Documentation updates should be part of the same commit as related code changes

## File Creation Policy
- **NEVER create files unless absolutely necessary for achieving your goal**
- **ALWAYS prefer editing an existing file to creating a new one**
- **NEVER proactively create documentation files (*.md) or README files unless explicitly requested**