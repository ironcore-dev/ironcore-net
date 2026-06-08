// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migration Suite")
}

type fakeMigration struct {
	err     error
	called  atomic.Bool
	started chan struct{}
	block   chan struct{}
}

func newFakeMigration(err error) *fakeMigration {
	return &fakeMigration{
		err:     err,
		started: make(chan struct{}),
		block:   make(chan struct{}),
	}
}

func newUnblockedMigration(err error) *fakeMigration {
	fm := newFakeMigration(err)
	fm.unblock()
	return fm
}

func (f *fakeMigration) Migrate(_ context.Context) error {
	f.called.Store(true)
	close(f.started)
	<-f.block
	return f.err
}

func (f *fakeMigration) unblock() {
	close(f.block)
}

// captureManager is a minimal fake that satisfies the manager.Manager interface.
// Only the Add method is overridden to capture the runnable; all other methods
// delegate to the embedded nil interface and will panic if called.
type captureManager struct {
	captured manager.Runnable
	manager.Manager
}

func (c *captureManager) Add(r manager.Runnable) error {
	c.captured = r
	return nil
}

var _ = Describe("Migrator", func() {
	It("should create a new migrator with open Done channel and nil Err", func() {
		m := NewMigrator()
		Expect(m).NotTo(BeNil())
		Expect(m.Done()).NotTo(BeClosed())
		Expect(m.Err()).NotTo(HaveOccurred())
	})

	It("should allow adding migrations before start", func() {
		m := NewMigrator()
		Expect(m.Add(newUnblockedMigration(nil))).To(Succeed())
	})

	It("should reject adding migrations after start", func(ctx SpecContext) {
		m := NewMigrator()
		Expect(m.Start(ctx)).To(Succeed())

		Expect(m.Add(newUnblockedMigration(nil))).To(MatchError(ContainSubstring("cannot add migrations once started")))
	})

	It("should run all registered migrations", func(ctx SpecContext) {
		m := NewMigrator()
		fm1 := newUnblockedMigration(nil)
		fm2 := newUnblockedMigration(nil)

		Expect(m.Add(fm1)).To(Succeed())
		Expect(m.Add(fm2)).To(Succeed())
		Expect(m.Start(ctx)).To(Succeed())

		Expect(fm1.called.Load()).To(BeTrue())
		Expect(fm2.called.Load()).To(BeTrue())
	})

	It("should succeed with no migrations", func(ctx SpecContext) {
		m := NewMigrator()
		Expect(m.Start(ctx)).To(Succeed())
		Expect(m.Done()).To(BeClosed())
	})

	It("should collect errors from failing migrations", func(ctx SpecContext) {
		m := NewMigrator()
		Expect(m.Add(newUnblockedMigration(fmt.Errorf("migration-1 failed")))).To(Succeed())
		Expect(m.Add(newUnblockedMigration(fmt.Errorf("migration-2 failed")))).To(Succeed())
		Expect(m.Add(newUnblockedMigration(nil))).To(Succeed())

		err := m.Start(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("migration-1 failed"))
		Expect(err.Error()).To(ContainSubstring("migration-2 failed"))
		Expect(m.Err()).To(Equal(err))
	})

	It("should reject starting twice", func(ctx SpecContext) {
		m := NewMigrator()
		Expect(m.Start(ctx)).To(Succeed())
		Expect(m.Start(ctx)).To(MatchError(ContainSubstring("already started")))
	})

	It("should close Done only after migrations complete", func(ctx SpecContext) {
		m := NewMigrator()
		fm := newFakeMigration(nil)
		Expect(m.Add(fm)).To(Succeed())

		go func() {
			defer GinkgoRecover()
			_ = m.Start(ctx)
		}()

		Eventually(fm.started).Should(BeClosed())
		Expect(m.Done()).NotTo(BeClosed())

		fm.unblock()
		Eventually(m.Done()).Should(BeClosed())
	})

	It("should run migrations concurrently", func(ctx SpecContext) {
		m := NewMigrator()
		fm1 := newFakeMigration(nil)
		fm2 := newFakeMigration(nil)
		Expect(m.Add(fm1)).To(Succeed())
		Expect(m.Add(fm2)).To(Succeed())

		go func() {
			defer GinkgoRecover()
			_ = m.Start(ctx)
		}()

		Eventually(fm1.started).Should(BeClosed())
		Eventually(fm2.started).Should(BeClosed())

		fm1.unblock()
		fm2.unblock()
		Eventually(m.Done()).Should(BeClosed())
	})
})

var _ = Describe("WrapManager", func() {
	It("should block runnables until migration completes, then run them", func(ctx SpecContext) {
		m := NewMigrator()
		fm := newFakeMigration(nil)
		Expect(m.Add(fm)).To(Succeed())

		mgr := &captureManager{}
		wrapped := WrapManager(m, mgr)

		runnableStarted := make(chan struct{})
		Expect(wrapped.Add(manager.RunnableFunc(func(ctx context.Context) error {
			close(runnableStarted)
			return nil
		}))).To(Succeed())

		runnableDone := make(chan error, 1)
		go func() {
			runnableDone <- mgr.captured.Start(ctx)
		}()

		Consistently(runnableStarted, 50*time.Millisecond).ShouldNot(BeClosed())

		fm.unblock()
		go func() {
			defer GinkgoRecover()
			_ = m.Start(ctx)
		}()

		Eventually(runnableStarted).Should(BeClosed())
		Eventually(runnableDone).Should(Receive(Succeed()))
	})

	It("should prevent runnables from starting when migration fails", func(ctx SpecContext) {
		m := NewMigrator()
		Expect(m.Add(newUnblockedMigration(fmt.Errorf("boom")))).To(Succeed())
		_ = m.Start(ctx)

		mgr := &captureManager{}
		wrapped := WrapManager(m, mgr)

		called := false
		Expect(wrapped.Add(manager.RunnableFunc(func(ctx context.Context) error {
			called = true
			return nil
		}))).To(Succeed())

		err := mgr.captured.Start(ctx)
		Expect(err).To(MatchError(ContainSubstring("boom")))
		Expect(called).To(BeFalse())
	})

	It("should exit cleanly when context is canceled before migration finishes", func(ctx SpecContext) {
		m := NewMigrator()
		fm := newFakeMigration(nil)
		Expect(m.Add(fm)).To(Succeed())

		mgr := &captureManager{}
		wrapped := WrapManager(m, mgr)

		called := false
		Expect(wrapped.Add(manager.RunnableFunc(func(ctx context.Context) error {
			called = true
			return nil
		}))).To(Succeed())

		runnableCtx, cancel := context.WithCancel(ctx)

		runnableDone := make(chan error, 1)
		go func() {
			runnableDone <- mgr.captured.Start(runnableCtx)
		}()

		cancel()

		Eventually(runnableDone).Should(Receive(Succeed()))
		Expect(called).To(BeFalse())

		fm.unblock()
		go func() { _ = m.Start(ctx) }()
		Eventually(m.Done()).Should(BeClosed())
	})
})
