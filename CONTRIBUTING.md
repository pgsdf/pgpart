# Contributing to PGPart

Thank you for your interest in contributing to PGPart! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Set up the development environment (see README.md)
4. Create a new branch for your feature or bugfix

## Development Setup

### Prerequisites

- FreeBSD 12.0+ or GhostBSD
- Go 1.18 or later
- Required packages: `pkg install go gcc git pkgconf mesa-libs libglvnd`

### Building from Source

```bash
git clone https://github.com/pgsdf/pgpart.git
cd pgpart
make deps
make build
```

## Code Style

- Follow standard Go conventions
- Run `gofmt` on your code before committing
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and single-purpose

## Testing

- Write tests for new features
- Ensure all existing tests pass before submitting
- Test on real hardware when possible (use VMs for destructive operations)
- Run `make test` before committing

## Pull Request Process

1. Update the README.md if you're adding new features
2. Ensure your code builds and runs without errors
3. Update documentation as needed
4. Create a pull request with a clear description of changes
5. Reference any related issues in your PR description

## Commit Messages

- Use clear, descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Keep the first line under 72 characters
- Add detailed description if needed after a blank line

Example:
```
Add support for ext4 filesystem detection

- Implement ext4 detection in partition scanner
- Add ext4 formatting option in UI
- Update documentation with ext4 support
```

## Reporting Bugs

When reporting bugs, please include:

- FreeBSD/GhostBSD version
- PGPart version
- Steps to reproduce the issue
- Expected vs actual behavior
- Any error messages or logs
- Hardware configuration (if relevant)

## Feature Requests

We welcome feature requests! Please:

- Check existing issues first to avoid duplicates
- Clearly describe the feature and its use case
- Explain why it would be valuable to users
- Be open to discussion and feedback

## Code of Conduct

- Be respectful and professional
- Welcome newcomers and help them get started
- Focus on constructive feedback
- Respect differing opinions and experiences

## Areas for Contribution

We especially welcome contributions in these areas:

- Additional filesystem support (ext2/3/4, NTFS, etc.)
- Improved error handling and user feedback
- Unit and integration tests
- Documentation improvements
- UI/UX enhancements
- Performance optimizations
- Bug fixes

## Security Issues

If you discover a security vulnerability:

- Do NOT open a public issue
- Email the maintainers directly
- Include detailed information about the vulnerability
- Allow time for the issue to be addressed before public disclosure

## License

By contributing to PGPart, you agree that your contributions will be licensed under the MIT License.

## Questions?

If you have questions about contributing:

- Check the README.md and documentation
- Search existing issues
- Open a new issue with the "question" label
- Join our community discussions

Thank you for helping make PGPart better!
