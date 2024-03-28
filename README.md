# GQL - GraphQL CLI

This is a small golang project designed to provide a command-line interface (CLI) for executing GraphQL queries, mutations, and subscriptions. The project utilizes various packages and libraries to facilitate GraphQL operations.

## Setup

1. **Clone Repository**: Clone this repository to your local machine.

2. **Install Dependencies**: Make sure you have Go installed. Then, navigate to the project directory and run:

   ```
   go mod tidy
   ```

## Usage

### GraphQL Endpoint

You can configure GraphQL server endpoint through environment variable: `GQL_URL`. Default is `http://_gql._tcp.local/query`

### Configuration

The project expects a configuration file in the GraphQL format. By default, it looks for a file named `.gql`. You can specify a custom configuration file using the environment variable `GQL_CONF`.

### Logging

You can configure logging behavior using environment variables:

- `GQL_LOG_LVL`: Set the log level (debug, info, warn, error).
- `GQL_LOG_FMT`: Set the log format (txt, json).
- `GQL_LOG_OUT`: Set the log output (stdout, stderr, or a custom file path).

### Running the CLI

To run the CLI, execute the following command:

```
go run .
```

### Command Structure

The CLI organizes commands based on the operations defined in the GraphQL configuration file. Each operation corresponds to a CLI command, with flags representing the operation's variables.

## Dependencies

The project relies on the following external packages:

- `github.com/hasura/go-graphql-client`: GraphQL client library for Go.
- `github.com/urfave/cli/v3`: Library for building command-line applications.
- `github.com/vektah/gqlparser/v2`: Library for parsing GraphQL queries.

## Contributing

Contributions are welcome! If you encounter any issues or have suggestions for improvements, feel free to open an issue or submit a pull request.

## License

This project is licensed under the [MIT License](LICENSE).
