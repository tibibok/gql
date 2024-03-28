package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"

	gql "github.com/hasura/go-graphql-client"
	"github.com/urfave/cli/v3"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const (
	configEnvName    = "GQL_CONF"
	logEnvName       = "GQL_LOG_LVL"
	logFormatEnvName = "GQL_LOG_FMT"
	logOutputEnvName = "GQL_LOG_OUT"
)

func main() {
	err := setSlog(
		LogFormat(getEnv(logFormatEnvName, LogTXT.String())),
		LogLevel(getEnv(logEnvName, LogInfo.String())),
		LogOutput(getEnv(logOutputEnvName, "stderr")),
	)
	if err != nil {
		slog.Error(
			"failed to set slog",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
		return
	}
	conf := getEnv(configEnvName, ".gql")
	slog.Debug(
		"reading config",
		slog.String("config", conf),
	)
	content, err := os.ReadFile(conf)
	if err != nil {
		slog.Error(
			"failed to read config file",
			slog.String("config", conf),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
		return
	}
	slog.Debug(
		"parsing config",
		slog.String("config", conf),
	)
	doc, err := parser.ParseQuery(&ast.Source{
		Name:  conf,
		Input: string(content),
	})
	if err != nil {
		slog.Error(
			"failed to parse query",
			slog.String("config", conf),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
		return
	}
	slog.Debug(
		"building cli",
		slog.String("config", conf),
	)
	cmd := &cli.Command{
		Name:     strings.Trim(conf, ".gql"),
		Flags:    []cli.Flag{},
		Commands: []*cli.Command{},
	}
	if doc.Comment != nil {
		cmd.Description = commentGroupText(doc.Comment)
	}
	if err := cliOperations(cmd, doc, string(content)); err != nil {
		slog.Error(
			"failed to parse query",
			slog.String("config", conf),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
		return
	}
	slog.Debug(
		"executing cli",
		slog.String("config", conf),
	)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.Error(
			"failed to run command",
			slog.String("config", conf),
			slog.String("error", err.Error()),
		)
		cancel()
		os.Exit(1)
		return
	}
}

func cliOperations(cmd *cli.Command, doc *ast.QueryDocument, raw string) error {
	for _, op := range doc.Operations {
		slog.Debug(
			"parsing operation",
			slog.String("operation", op.Name),
		)
		if op.Name == "" {
			if op.Comment != nil {
				cmd.Description = commentGroupText(op.Comment)
			}
			flags, err := cliVariables(op, cmd)
			if err != nil {
				return err
			}
			cmd.Flags = append(cmd.Flags, flags...)
			// TODO: add action to root cmd
			continue
		}
		c := &cli.Command{
			Name:     op.Name,
			Category: string(op.Operation),
			Flags:    []cli.Flag{},
			Action: func(ctx context.Context, c *cli.Command) error {
				vars := map[string]any{}
				for _, fn := range c.FlagNames() {
					vars[fn] = c.Value(fn)
				}
				args := c.Args().Slice()
				vars["_"] = args
				for n, arg := range args {
					vars[fmt.Sprintf("_%d", n+1)] = arg
				}
				cl := gql.NewClient(getEnv("GQL_URL", "http://_gql._tcp.local/query"), nil)
				switch op.Operation {
				case ast.Query, ast.Mutation:
					out, err := cl.ExecRaw(ctx, raw, vars, gql.OperationName(op.Name))
					if err != nil {
						return err
					}
					os.Stdout.Write(out)
					os.Stdout.Write([]byte("\n"))
				case ast.Subscription:
					return fmt.Errorf("subscriptions not implemented yet")
				}
				return nil
			},
		}
		flags, err := cliVariables(op, c)
		if err != nil {
			return err
		}
		c.Flags = append(c.Flags, flags...)
		if op.Comment != nil {
			c.Usage = commentGroupText(op.Comment)
		}

		cmd.Commands = append(cmd.Commands, c)
	}
	return nil
}

func commentGroupText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	buff := &strings.Builder{}
	for _, c := range cg.List {
		buff.WriteString(c.Text())
	}
	return buff.String()
}

var argRegexp = regexp.MustCompile(`^_[0-9]*$`)

func cliVariables(op *ast.OperationDefinition, cmd *cli.Command) ([]cli.Flag, error) {
	out := []cli.Flag{}
	for _, v := range op.VariableDefinitions {
		if argRegexp.MatchString(v.Variable) {
			slog.Debug(
				"skip arg variable",
				slog.String("variable", v.Variable),
			)
			cmd.ArgsUsage = cmd.ArgsUsage + commentGroupText(v.Comment)
			continue
		}
		switch v.Type.NamedType {
		case "String":
			f := &cli.StringFlag{
				Name:     v.Variable,
				Usage:    commentGroupText(v.Comment),
				Required: v.Type.NonNull,
			}
			if v.DefaultValue != nil {
				f.Value = v.DefaultValue.Raw
				f.Required = false
			}
			out = append(out, f)
		case "Int", "Int64":
			f := &cli.IntFlag{
				Name:     v.Variable,
				Usage:    commentGroupText(v.Comment),
				Required: v.Type.NonNull,
			}
			if v.DefaultValue != nil {
				i, err := strconv.Atoi(v.DefaultValue.Raw)
				if err != nil {
					slog.Warn(
						"failed to parse default value",
						slog.String("value", v.DefaultValue.Raw),
						slog.String("error", err.Error()),
					)
				}
				f.Value = int64(i)
				f.Required = false
			}
			out = append(out, f)
		case "Boolean":
			f := &cli.StringFlag{
				Name:     v.Variable,
				Usage:    commentGroupText(v.Comment),
				Required: v.Type.NonNull,
			}
			if v.DefaultValue != nil {
				f.Value = v.DefaultValue.Raw
				f.Required = false
			}
			out = append(out, f)
		}
	}
	return out, nil
}

func getEnv(name string, def string) string {
	value := os.Getenv(name)
	if value == "" {
		return def
	}
	return value
}

type LogFormat string

func (f LogFormat) String() string { return string(f) }

const (
	LogTXT  LogFormat = "txt"
	LogJSON LogFormat = "json"
)

type LogLevel string

func (l LogLevel) String() string { return string(l) }

func (l LogLevel) Level() slog.Level {
	switch l {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

type LogOutput string

func (o LogOutput) String() string { return string(o) }

func (o LogOutput) Output() (io.WriteCloser, error) {
	switch o {
	case "stdout":
		return os.Stdout, nil
	case "stderr", "":
		return os.Stderr, nil

	default:
		return os.OpenFile(string(o), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}
}

func setSlog(format LogFormat, level LogLevel, output LogOutput) error {
	out, err := output.Output()
	if err != nil {
		return err
	}
	switch format {
	case LogTXT:
		slog.SetDefault(slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{
			Level: level.Level(),
		})))
		return nil
	case LogJSON:
		slog.SetDefault(slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
			Level: level.Level(),
		})))
		return nil
	default:
		return fmt.Errorf("unknown log format: %s", format)
	}
}
