package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	configpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/config"
)

// Environment tracks shared state for CLI commands.
type Environment struct {
	ConfigPath string
	Config     *configpkg.Config
}

// Save persists the currently loaded configuration to disk.
func (e *Environment) Save() error {
	if e == nil {
		return errors.New("cli environment is nil")
	}
	if e.Config == nil {
		return errors.New("configuration not loaded")
	}
	path := e.ConfigPath
	if path == "" {
		var err error
		path, err = configpkg.DefaultPath()
		if err != nil {
			return err
		}
		e.ConfigPath = path
	}
	return e.Config.Save(path)
}

type envKey struct{}

func withEnvironment(ctx context.Context, env *Environment) context.Context {
	return context.WithValue(ctx, envKey{}, env)
}

// EnvironmentFrom extracts the CLI environment from the command context.
func EnvironmentFrom(cmd *cobra.Command) (*Environment, error) {
	if cmd == nil {
		return nil, errors.New("command is nil")
	}
	env, _ := cmd.Context().Value(envKey{}).(*Environment)
	if env == nil {
		return nil, errors.New("cli environment not initialized")
	}
	return env, nil
}

// MustEnvironment panics if the CLI environment cannot be resolved. Useful within tests.
func MustEnvironment(cmd *cobra.Command) *Environment {
	env, err := EnvironmentFrom(cmd)
	if err != nil {
		panic(err)
	}
	return env
}
