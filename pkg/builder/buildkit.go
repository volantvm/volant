// Copyright (c) 2025 HYPR. PTE. LTD.
//go:build linux

package builder

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/moby/buildkit/cache/remotecache"
	inlineremotecache "github.com/moby/buildkit/cache/remotecache/inline"
	localremotecache "github.com/moby/buildkit/cache/remotecache/local"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/frontend"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	fgateway "github.com/moby/buildkit/frontend/gateway"
	"github.com/moby/buildkit/frontend/gateway/forwarder"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/solver"
	"github.com/moby/buildkit/solver/bboltcachestorage"
	"github.com/moby/buildkit/util/resolver"
	"github.com/moby/buildkit/worker"
	"github.com/moby/buildkit/worker/base"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufConnSize = 32 << 20 // 32MB buffer for in-memory gRPC
)

// EmbeddedBuildKit provides embedded BuildKit functionality without Docker daemon
type EmbeddedBuildKit struct {
	stateDir string
	reporter ProgressReporter
}

// NewEmbeddedBuildKit creates a new embedded BuildKit builder
func NewEmbeddedBuildKit(stateDir string, reporter ProgressReporter) *EmbeddedBuildKit {
	if stateDir == "" {
		stateDir = ensureStateDir()
	}
	if reporter == nil {
		reporter = NewConsoleReporter(os.Stdout)
	}

	return &EmbeddedBuildKit{
		stateDir: stateDir,
		reporter: reporter,
	}
}

// BuildToOCI builds a Dockerfile and exports to OCI tar format
// This is MUCH faster than exporting to local directory
func (b *EmbeddedBuildKit) BuildToOCI(ctx context.Context, opts BuildOptions) (string, error) {
	startTime := time.Now()
	b.reporter.StartStage("Initializing embedded BuildKit")

	// Create temp directory for OCI output
	ociDir, err := os.MkdirTemp("", "volant-buildkit-oci-*")
	if err != nil {
		return "", fmt.Errorf("create oci temp dir: %w", err)
	}

	ociTarPath := filepath.Join(ociDir, "image.tar")

	// Create embedded BuildKit client
	client, cleanup, err := b.newEmbeddedClient(ctx)
	if err != nil {
		b.reporter.FailStage("Initialize BuildKit", err)
		return "", err
	}
	defer cleanup()

	b.reporter.CompleteStage("Initialize embedded BuildKit", time.Since(startTime))

	// Prepare build
	buildStart := time.Now()
	b.reporter.StartStage(fmt.Sprintf("Building %s:%s", opts.ImageName, opts.ImageVersion))

	dfDir := filepath.Dir(opts.Dockerfile)
	dfBase := filepath.Base(opts.Dockerfile)

	frontendAttrs := map[string]string{
		"filename": dfBase,
	}
	if opts.Target != "" {
		frontendAttrs["target"] = opts.Target
		b.reporter.StartStep(fmt.Sprintf("Target stage: %s", opts.Target))
	}
	for k, v := range opts.BuildArgs {
		frontendAttrs["build-arg:"+k] = v
	}

	// Export to OCI tar (fast!)
	solveOpt := bkclient.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,
		LocalDirs: map[string]string{
			"context":    opts.Context,
			"dockerfile": dfDir,
		},
		Exports: []bkclient.ExportEntry{
			{
				Type: bkclient.ExporterOCI,
				Output: func(_ map[string]string) (io.WriteCloser, error) {
					return os.Create(ociTarPath)
				},
			},
		},
	}

	// Run build with progress tracking
	statusCh := make(chan *bkclient.SolveStatus, 16)
	var progressWG sync.WaitGroup
	progressWG.Add(1)

	go func() {
		defer progressWG.Done()
		b.trackBuildProgress(statusCh)
	}()

	_, err = client.Solve(ctx, nil, solveOpt, statusCh)
	progressWG.Wait()

	if err != nil {
		b.reporter.FailStage("Build Dockerfile", err)
		return "", fmt.Errorf("buildkit solve: %w", err)
	}

	b.reporter.CompleteStage(fmt.Sprintf("Build %s:%s", opts.ImageName, opts.ImageVersion), time.Since(buildStart))

	return ociTarPath, nil
}

// trackBuildProgress monitors BuildKit progress and reports it beautifully
func (b *EmbeddedBuildKit) trackBuildProgress(statusCh <-chan *bkclient.SolveStatus) {
	seenVertices := make(map[string]bool)
	stageTimes := make(map[string]time.Time)

	for st := range statusCh {
		// Track vertex (stage) progress
		for _, v := range st.Vertexes {
			if v == nil {
				continue
			}

			vertexKey := v.Digest.String()

			// New vertex started
			if _, seen := seenVertices[vertexKey]; !seen {
				seenVertices[vertexKey] = true
				stageTimes[vertexKey] = time.Now()

				// Check if this is a cached layer
				if v.Cached {
					b.reporter.CacheHit(v.Name)
				} else {
					b.reporter.CacheMiss(v.Name)
				}
			}

			// Vertex completed
			if v.Completed != nil && v.Started != nil {
				duration := v.Completed.Sub(*v.Started)
				if v.Error != "" {
					b.reporter.FailStage(v.Name, fmt.Errorf("%s", v.Error))
				}
				// Don't report completion for cached items (already reported as cache hit)
				if !v.Cached {
					b.reporter.CompleteStep(v.Name)
				}
			}

			// Vertex error
			if v.Error != "" && v.Completed == nil {
				b.reporter.FailStage(v.Name, fmt.Errorf("%s", v.Error))
			}
		}

		// Track download/upload progress
		for _, s := range st.Statuses {
			if s == nil {
				continue
			}

			name := s.Name
			if name == "" {
				name = s.ID
			}
			if name == "" {
				continue
			}

			if s.Total > 0 && s.Current > 0 {
				b.reporter.UpdateProgress(name, s.Current)
			}
		}
	}
}

// newEmbeddedClient creates an in-process BuildKit client with in-memory gRPC
func (b *EmbeddedBuildKit) newEmbeddedClient(ctx context.Context) (*bkclient.Client, func(), error) {
	// Create session manager
	sm, err := session.NewManager()
	if err != nil {
		return nil, nil, fmt.Errorf("session manager: %w", err)
	}

	// Setup worker
	workerRoot := filepath.Join(b.stateDir, "worker")
	registryHosts := resolver.NewRegistryConfig(nil)

	wk, err := base.NewWorker(ctx, base.WorkerOpt{
		ID:            "volant-builder",
		MetadataStore: filepath.Join(workerRoot, "metadata"),
		Executor:      nil, // Will use default executor
		Platforms:     nil, // Auto-detect
		GCPolicy:      nil, // Default GC policy
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create worker: %w", err)
	}

	wc := &worker.Controller{}
	if err := wc.Add(wk); err != nil {
		wk.Close()
		return nil, nil, fmt.Errorf("add worker: %w", err)
	}

	// Setup cache storage
	cachePath := filepath.Join(b.stateDir, "cache.db")
	cacheStorage, err := bboltcachestorage.NewStore(cachePath)
	if err != nil {
		wc.Close()
		return nil, nil, fmt.Errorf("cache store: %w", err)
	}

	// Setup history database
	historyPath := filepath.Join(b.stateDir, "history.db")
	historyDB, err := bbolt.Open(historyPath, 0o600, nil)
	if err != nil {
		cacheStorage.Close()
		wc.Close()
		return nil, nil, fmt.Errorf("history db: %w", err)
	}

	defaultWorker, err := wc.GetDefault()
	if err != nil {
		historyDB.Close()
		cacheStorage.Close()
		wc.Close()
		return nil, nil, fmt.Errorf("get default worker: %w", err)
	}

	contentStore := defaultWorker.ContentStore()
	if contentStore == nil {
		historyDB.Close()
		cacheStorage.Close()
		wc.Close()
		return nil, nil, fmt.Errorf("content store unavailable")
	}

	leaseManager := defaultWorker.LeaseManager()
	if leaseManager == nil {
		historyDB.Close()
		cacheStorage.Close()
		wc.Close()
		return nil, nil, fmt.Errorf("lease manager unavailable")
	}

	// Setup frontends
	frontends := map[string]frontend.Frontend{
		"dockerfile.v0": forwarder.NewGatewayForwarder(wc.Infos(), builder.Build),
		"gateway.v0":    fgateway.NewGatewayFrontend(wc.Infos()),
	}

	// Setup cache manager
	cacheMgr := solver.NewCacheManager(ctx, identity.NewID(), cacheStorage, worker.NewCacheResultStorage(wc))

	// Setup cache exporters/importers
	cacheExporters := map[string]remotecache.ResolveCacheExporterFunc{
		"local":  localremotecache.ResolveCacheExporterFunc(sm),
		"inline": inlineremotecache.ResolveCacheExporterFunc(),
	}

	cacheImporters := map[string]remotecache.ResolveCacheImporterFunc{
		"local": localremotecache.ResolveCacheImporterFunc(sm),
	}

	// Create controller
	controller, err := control.NewController(control.Opt{
		SessionManager:            sm,
		WorkerController:          wc,
		Frontends:                 frontends,
		CacheManager:              cacheMgr,
		ResolveCacheExporterFuncs: cacheExporters,
		ResolveCacheImporterFuncs: cacheImporters,
		Entitlements:              nil,
		HistoryDB:                 historyDB,
		CacheStore:                cacheStorage,
		LeaseManager:              leaseManager,
		ContentStore:              contentStore,
		HistoryConfig:             nil,
	})
	if err != nil {
		historyDB.Close()
		cacheStorage.Close()
		wc.Close()
		return nil, nil, fmt.Errorf("create controller: %w", err)
	}

	// Setup in-memory gRPC (no network sockets!)
	listener := bufconn.Listen(bufConnSize)
	server := grpc.NewServer()
	controller.Register(server)

	serverErr := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			serverErr <- err
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	client, err := bkclient.New(ctx, "", bkclient.WithContextDialer(dialer))
	if err != nil {
		server.Stop()
		listener.Close()
		controller.Close()
		historyDB.Close()
		cacheStorage.Close()
		wc.Close()
		return nil, nil, fmt.Errorf("create client: %w", err)
	}

	cleanup := func() {
		if client != nil {
			client.Close()
		}
		server.Stop()
		listener.Close()
		if err := controller.Close(); err != nil {
			log.Printf("controller close error: %v", err)
		}
		select {
		case err := <-serverErr:
			if err != nil {
				log.Printf("server error: %v", err)
			}
		default:
		}
		historyDB.Close()
		cacheStorage.Close()
		wc.Close()
	}

	return client, cleanup, nil
}
