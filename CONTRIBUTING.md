# Contributing to terraform-provider-groundcover

We love your input! We want to make contributing to the groundcover Terraform provider as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code follows the existing style
6. Issue that pull request!

## Local Development Setup

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.24
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- groundcover account with API access

### Setting Up Your Development Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/groundcover-com/terraform-provider-groundcover.git
   cd terraform-provider-groundcover
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the provider:
   ```bash
   make build
   ```

4. Set up local provider override (see README.md for detailed instructions)

### Running Tests

Before submitting a pull request, ensure all tests pass:

```bash
# Run unit tests
go test ./internal/provider -v

# Run acceptance tests (requires groundcover environment)
export GROUNDCOVER_API_KEY="your-api-key"
export GROUNDCOVER_API_URL="https://api.main.groundcover.com/"
export GROUNDCOVER_BACKEND_ID="your-backend-id"
export GROUNDCOVER_INCLOUD_BACKEND_ID="your-in-cloud-backend-id"  # For ingestion key tests
TF_ACC=1 go test ./internal/provider -v
```

### Code Style

- Follow Go best practices and idioms
- Use `gofmt` to format your code:
  ```bash
  make fmt
  ```
- Run the linter:
  ```bash
  make lint
  ```

## Adding New Resources

When adding a new resource to the provider:

1. Create the resource file in `internal/provider/resource_<name>.go`
2. Create the corresponding client file in `internal/provider/client_<name>.go`
3. Add comprehensive acceptance tests in `internal/provider/resource_<name>_test.go`
4. Add an example in `examples/resources/groundcover_<name>/resource.tf`
5. Update documentation by running:
   ```bash
   cd tools && go generate ./...
   ```

### Resource Implementation Checklist

- [ ] Implements all CRUD operations (Create, Read, Update, Delete)
- [ ] Supports resource import
- [ ] Handles API errors gracefully
- [ ] Includes retry logic for transient failures
- [ ] Has comprehensive acceptance tests
- [ ] Includes "disappears" test for external deletion handling
- [ ] Has clear, helpful error messages
- [ ] Documentation includes all attributes and examples

## Pull Request Process

1. Update the CHANGELOG.md with details of changes
2. Update the README.md with details of changes to the interface, if applicable
3. Ensure all tests pass and coverage is maintained
4. The PR will be merged once you have the sign-off of at least one maintainer

### PR Title Convention

Please use clear, descriptive titles:
- `feat: Add support for X resource`
- `fix: Handle empty response in Y resource`
- `docs: Update configuration examples`
- `chore: Update dependencies`

## Reporting Bugs

### Security Issues

If you discover a security vulnerability, please email security@groundcover.com instead of using the issue tracker.

### Bug Reports

When reporting bugs, please include:

1. Your Terraform version (`terraform version`)
2. Provider version
3. Relevant Terraform configuration (sanitized of sensitive data)
4. Debug output (`TF_LOG=DEBUG terraform apply`)
5. Expected behavior
6. Actual behavior
7. Steps to reproduce

## Feature Requests

We love feature requests! When submitting a feature request:

1. Explain the use case
2. Provide example Terraform configuration showing how you'd like to use it
3. Explain why existing resources/attributes don't meet your needs
4. If possible, link to groundcover API documentation for the feature

## Documentation

- Keep resource documentation up to date
- Include practical examples
- Document any breaking changes clearly
- Use proper groundcover styling (lowercase "groundcover")

## Community

- Be respectful and inclusive
- Help others when you can
- Follow the [Go Community Code of Conduct](https://go.dev/conduct)

## License

By contributing, you agree that your contributions will be licensed under the Mozilla Public License Version 2.0 (MPL-2.0).

## Questions?

Feel free to open an issue for any questions about contributing!