// This file and its contents are licensed under the Apache License 2.0.
// Please see the included NOTICE for copyright information and
// LICENSE for a copy of the license.

package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v4"
	"io"
	"os"
)

func main() {
	ctx := context.Background()

	// connect to the database
	con, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer func(con *pgx.Conn, ctx context.Context) {
		err := con.Close(ctx)
		if err != nil {
			panic(err)
		}
	}(con, ctx)

	// start a transaction
	tx, err := con.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && err != pgx.ErrTxClosed {
			panic(err)
		}
	}(tx, ctx)

	dec := json.NewDecoder(os.Stdin)
	trace := make(map[string]interface{})
	for {
		// decode json objects from stdin
		if err = dec.Decode(&trace); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		// insert the json object into the trace_stg table
		_, err := tx.Exec(ctx, "insert into trace_stg (trace) values ($1)", trace)
		if err != nil {
			panic(err)
		}
	}

	// commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		panic(err)
	}
}
