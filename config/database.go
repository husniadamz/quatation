package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	sqlmysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConnectDatabase(cfg Config) (*gorm.DB, error) {
	driverConfig := sqlmysql.NewConfig()
	driverConfig.User = cfg.DBUser
	driverConfig.Passwd = cfg.DBPass
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(cfg.DBHost, cfg.DBPort)
	driverConfig.DBName = cfg.DBName
	driverConfig.ParseTime = true
	params, err := url.ParseQuery(cfg.DBParams)
	if err != nil {
		return nil, fmt.Errorf("parse DB_PARAMS: %w", err)
	}
	driverConfig.Params = make(map[string]string, len(params))
	for key, values := range params {
		if len(values) == 0 {
			continue
		}
		switch key {
		case "parseTime":
			driverConfig.ParseTime, err = strconv.ParseBool(values[len(values)-1])
			if err != nil {
				return nil, fmt.Errorf("invalid DB_PARAMS parseTime: %w", err)
			}
		case "loc":
			driverConfig.Loc, err = time.LoadLocation(values[len(values)-1])
			if err != nil {
				return nil, fmt.Errorf("invalid DB_PARAMS loc: %w", err)
			}
		default:
			driverConfig.Params[key] = values[len(values)-1]
		}
	}
	if _, ok := driverConfig.Params["charset"]; !ok {
		driverConfig.Params["charset"] = "utf8mb4"
	}
	dsn := driverConfig.FormatDSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open MySQL connection: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get MySQL connection pool: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return db, nil
}
