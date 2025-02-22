// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2019 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/jaegertracing/jaeger/pkg/config"
	"github.com/jaegertracing/jaeger/pkg/metrics"
	"github.com/jaegertracing/jaeger/pkg/testutils"
	"github.com/jaegertracing/jaeger/plugin/storage/cassandra"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
)

var errInitializeCassandraDependencyWriter = errors.New("failed to initialize cassandra dependency writer")

type CassandraStorageIntegration struct {
	StorageIntegration

	logger *zap.Logger
}

func newCassandraStorageIntegration() *CassandraStorageIntegration {
	return &CassandraStorageIntegration{
		StorageIntegration: StorageIntegration{
			GetDependenciesReturnsSource: true,

			Refresh: func() error { return nil },
			CleanUp: func() error { return nil },
		},
	}
}

func (s *CassandraStorageIntegration) initializeCassandraFactory(flags []string) (*cassandra.Factory, error) {
	s.logger, _ = testutils.NewLogger()
	f := cassandra.NewFactory()
	v, command := config.Viperize(f.AddFlags)
	if err := command.ParseFlags(flags); err != nil {
		return nil, fmt.Errorf("unable to parse flags: %w", err)
	}
	f.InitFromViper(v, zap.NewNop())
	if err := f.Initialize(metrics.NullFactory, s.logger); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *CassandraStorageIntegration) initializeCassandra() error {
	f, err := s.initializeCassandraFactory([]string{
		"--cassandra.keyspace=jaeger_v1_dc1",
	})
	if err != nil {
		return err
	}
	if s.SpanWriter, err = f.CreateSpanWriter(); err != nil {
		return err
	}
	if s.SpanReader, err = f.CreateSpanReader(); err != nil {
		return err
	}
	if err = s.initializeDependencyReaderAndWriter(f); err != nil {
		return err
	}
	return nil
}

func (s *CassandraStorageIntegration) initializeDependencyReaderAndWriter(f *cassandra.Factory) error {
	var (
		err error
		ok  bool
	)
	if s.DependencyReader, err = f.CreateDependencyReader(); err != nil {
		return err
	}
	// TODO: Update this when the factory interface has CreateDependencyWriter
	if s.DependencyWriter, ok = s.DependencyReader.(dependencystore.Writer); !ok {
		return errInitializeCassandraDependencyWriter
	}
	return nil
}

func TestCassandraStorage(t *testing.T) {
	if os.Getenv("STORAGE") != "cassandra" {
		t.Skip("Integration test against Cassandra skipped; set STORAGE env var to cassandra to run this")
	}
	s := newCassandraStorageIntegration()
	require.NoError(t, s.initializeCassandra())
	s.IntegrationTestAll(t)
}
