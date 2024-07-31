package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

var pool = newGoroutinesPool(20)

func main() {
}

type Person struct {
	ID   int
	Name string
}

type Repository struct {
	shards map[int]*sql.DB
}

func (r *Repository) FindByName(ctx context.Context, name string) ([]*Person, error) {
	// Our database has many shards. We use Person.ID as the sharding key.
	// Now we want to find all records containing this name.
	// Implement this function.

	dtos, err := r.query(ctx, "SELECT id, name FROM person WHERE name = %s", name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch query: %w", err)
	}

	persons := make([]*Person, 0, len(dtos))
	for _, dto := range dtos {
		persons = append(persons, &Person{ID: dto.ID, Name: dto.Name})
	}

	return persons, nil
}

type dto struct {
	ID   int
	Name string
}

type queryResult struct {
	records []dto
	err     error
}

func (r *Repository) query(ctx context.Context, query string, args ...any) ([]dto, error) {
	var mu sync.Mutex
	records := make([]dto, 0)

	// чтобы завершить дочерние горутины по первой ошибке
	ctx, cancel := context.WithCancelCause(ctx)

	// чтобы дождаться всех работяг
	var wg sync.WaitGroup

	for _, db := range r.shards {
		shardResultChan := queryAsync(ctx, db, query, args)

		wg.Add(1)
		pool.Go(ctx, func(ctx context.Context) {
			defer wg.Done()

			select {
			case <-ctx.Done():
			case shardResult := <-shardResultChan:
				if shardResult.err != nil {
					cancel(shardResult.err)
					return
				}
				mu.Lock()
				records = append(records, shardResult.records...)
				mu.Unlock()
			}
		})
	}

	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, fmt.Errorf("error while query execution: %w", err)
	}

	return records, nil
}

func queryAsync(ctx context.Context, db *sql.DB, query string, args ...any) <-chan queryResult {
	resultChan := make(chan queryResult)

	pool.Go(ctx, func(ctx context.Context) {
		defer close(resultChan)

		records, err := exec(ctx, db, query, args)
		result := queryResult{
			records: records,
			err:     err,
		}

		select {
		case <-ctx.Done():
		case resultChan <- result:
		}
	})

	return resultChan
}

func exec(ctx context.Context, db *sql.DB, query string, args ...any) ([]dto, error) {
	rows, err := db.QueryContext(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to run query %s on shard: %w", query, err)
	}
	defer rows.Close()

	records := make([]dto, 0)
	for i := 0; rows.Next(); i++ {
		var record dto
		if err = rows.Scan(&record.ID, &record.Name); err != nil {
			return nil, fmt.Errorf("failed to scan value into dto: %w", err)
		}
		records = append(records, record)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error encountered during rows iteration: %w", err)
	}

	return records, nil
}

type goroutinesPool struct {
	capacity  int64
	semaphore chan struct{}
}

func newGoroutinesPool(capacity int64) *goroutinesPool {
	return &goroutinesPool{
		capacity:  capacity,
		semaphore: make(chan struct{}, capacity),
	}
}

func (p *goroutinesPool) Go(ctx context.Context, fn func(ctx context.Context)) {
	select {
	case <-ctx.Done():
	case p.semaphore <- struct{}{}:
		go func() {
			fn(ctx)
			<-p.semaphore
		}()
	}
}
