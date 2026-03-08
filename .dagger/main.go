// A generated module for Botreplyer functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/botreplyer/internal/dagger"
	"fmt"

	"golang.org/x/sync/errgroup"
)

type Botreplyer struct{}

// Returns a container that echoes whatever string argument is provided
func (m *Botreplyer) ContainerEcho(stringArg string) *dagger.Container {
	return dag.Container().From("alpine:latest").WithExec([]string{"echo", stringArg})
}

// Returns lines that match a pattern in the files of the provided Directory
func (m *Botreplyer) GrepDir(ctx context.Context, directoryArg *dagger.Directory, pattern string) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grep", "-R", pattern, "."}).
		Stdout(ctx)
}

func (m *Botreplyer) RunAllChecks(ctx context.Context, source *dagger.Directory) error {
	goCache := dag.CacheVolume("go-build-cache")
	modCache := dag.CacheVolume("go-mod-cache")
	lintCache := dag.CacheVolume("golangci-lint-cache")

	// 1. 定義基礎環境 (鎖定 Go 1.24)
	toolBase := dag.Container().
		From("golang:1.25-bookworm").
		WithMountedCache("/go/pkg/mod", modCache).
		WithMountedCache("/root/.cache/go-build", goCache).
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"})

	goBase := toolBase.
		WithDirectory("/src", source).
		WithWorkdir("/src")

	g, ctx := errgroup.WithContext(ctx)
	// 2. 執行 go mod tidy 檢查
	// 如果 tidy 後有變動，這步會失敗，達到 check-mod-tidy 的效果
	safeGo(g, func() error {
		_, err := goBase.
			WithExec([]string{"go", "mod", "tidy"}).
			WithExec([]string{"git", "diff", "--exit-code", "go.mod", "go.sum"}).
			Sync(ctx)
		if err != nil {
			return err
		}
		return nil
	})

	// 3. 執行 golangci-lint (包含你設定的 5m timeout)
	safeGo(g, func() error {
		_, err := dag.Container().
			From("golangci/golangci-lint:v2.8-alpine").
			WithMountedCache("/root/.cache/golangci-lint", lintCache).
			WithMountedCache("/go/pkg/mod", modCache). // Lint 也需要下載依賴，掛載它可以加速
			WithDirectory("/src", source).
			WithWorkdir("/src").
			WithExec([]string{"golangci-lint", "run", "--timeout", "5m"}).
			Sync(ctx)
		if err != nil {
			return err
		}
		return nil
	})

	safeGo(g, func() error {
		// 4. 執行 govulncheck
		_, err := goBase.
			WithExec([]string{"govulncheck", "./..."}).
			Sync(ctx)
		if err != nil {
			return err
		}
		return nil
	})

	return g.Wait()
}

func safeGo(g *errgroup.Group, fn func() error) {
	g.Go(func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in goroutine: %v", r)
			}
		}()
		return fn()
	})
}
