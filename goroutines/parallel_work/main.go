package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

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

func (r *Repository) query(ctx context.Context, query string, args ...any) ([]dto, error) {
	// чтобы завершить дочерние горутины по первой ошибке
	ctx, cancel := context.WithCancelCause(ctx)

	// чтобы дождаться всех работяг
	var wg sync.WaitGroup
	var errOnce sync.Once

	records := make([][]dto, 0, len(r.shards))

	i := 0
	for id, db := range r.shards {
		records = append(records, make([]dto, 0))

		wg.Add(1)
		go func(ctx context.Context, idx int) {
			defer wg.Done()

			if err := run(ctx, id, db, &records[idx], query, args); err != nil {
				errOnce.Do(func() {
					cancel(err)
				})
			}
		}(ctx, i)

		i++
	}

	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, fmt.Errorf("error while query execution: %w", err)
	}

	var gatheredRecords []dto
	for _, shardRecords := range records {
		gatheredRecords = append(gatheredRecords, shardRecords...)
	}

	return gatheredRecords, nil
}

func run(ctx context.Context, shardID int, db *sql.DB, dest *[]dto, query string, args ...any) error {
	if dest == nil {
		return fmt.Errorf("provided nil pointer dest receiver, dest must be a valid address")
	}

	rows, err := db.QueryContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("failed to query query %s on shard %d: %w", query, shardID, err)
	}
	defer rows.Close()

	for i := 0; rows.Next(); i++ {
		var record dto
		if err = rows.Scan(record.receivers()); err != nil {
			return fmt.Errorf("failed to scan value into dto receivers: %w", err)
		}

		*dest = append(*dest, record)
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error encountered during rows iteration: %w", err)
	}

	return nil
}

type dto struct {
	ID   int
	Name string
}

func (d *dto) receivers() []*any {
	id := any(d.ID)
	name := any(d.Name)
	return []*any{&id, &name}
}

// TODO: разобрать вывод go:noinline
// v escapes to heap: ...
// moved to heap: v
// d.ID escapes to heap
// d.Name escapes to heap
// parameter d leaks to {storage for d.Name} with derefs=1
// []*any{...} escapes to heap
// leaking param content: d
// []*any{...} escapes to heap
// d.ID escapes to heap
// d.Name escapes to heap
// ...
//func pointer(v any) *any {
//	return &v
//}
