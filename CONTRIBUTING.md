# Contributing to otserver

First off, thank you for considering contributing to `otserver`! We welcome contributions from everyone.

## Getting Started

1. Fork the repository on GitHub.
2. Clone your forked repository to your local machine.
3. Add the upstream repository as a remote:

   ```bash
   git remote add upstream https://github.com/MutterPedro/otserver.git
   ```

## Development Workflow

This project strictly adheres to idiomatic Go conventions. Please ensure your development workflow includes the following steps:

1. Create a branch for your feature or bug fix:

   ```bash
   git checkout -b feature/my-new-feature
   ```

2. Write your code, ensuring it follows the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
3. Write sufficient unit tests to verify your code. Run the tests using the race detector:

   ```bash
   make test-race
   ```

4. Format your code and run the linters:

   ```bash
   make fmt
   make lint
   ```

5. Commit your changes. Use clear, descriptive commit messages. e.g. `network: fix race condition in packet dispatcher`

## Pull Request Process

1. Prior to submitting a PR, ensure that your branch is rebased against the `upstream/main` branch.
2. Open a Pull Request with a clear title and description.
3. Ensure the GitHub Actions CI pipeline passes successfully.
4. The maintainers will review your code. You might be asked to make changes, which is a normal part of the process!

## Code Conventions

- Avoid stuttering (`game.Game` -> `game.Engine`).
- Interfaces are defined where they are used (consumer-owned), not where they are implemented.
- We use `gofumpt` instead of `gofmt` to enforce stricter formatting rules.
- Run `make vuln` to check for known vulnerabilities in dependencies.
