package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConnectionConfig struct {
	ConnMaxLifetime   *int
	ConnMaxIdleTime   *int
	MaxOpenConns      *int
	HealthCheckPeriod *int
}

func NewPostgresDB(dbUrl string, dbConnCfg DBConnectionConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := DBConfig(dbUrl, dbConnCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create a config, error: %v", err)
	}
	connPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("error while creating connection to the database!! : %v", err)
	}
	connection, err := connPool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while acquiring connection from the database pool!! : %v", err)
	}
	defer connection.Release()

	err = connection.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not ping database : %v", err)
	}
	return connPool, nil
}

func DBConfig(dbUrl string, dbConnCfg DBConnectionConfig) (*pgxpool.Config, error) {
	dbConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create a config, error: %v", err)
	}

	dbConfig.ConnConfig.RuntimeParams["timezone"] = "Asia/Bangkok"
	dbConfig.MaxConns = int32(*dbConnCfg.MaxOpenConns)
	dbConfig.MaxConnLifetime = time.Duration(*dbConnCfg.ConnMaxLifetime) * time.Minute
	dbConfig.MaxConnIdleTime = time.Duration(*dbConnCfg.ConnMaxIdleTime) * time.Minute
	dbConfig.HealthCheckPeriod = time.Duration(*dbConnCfg.HealthCheckPeriod) * time.Minute

	return dbConfig, nil
}
