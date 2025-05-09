// Copyright (c) 2025 minRAG Authors.
//
// This file is part of minRAG.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses>.

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"os"

	"gitee.com/chunanyong/zorm"

	// 00.引入数据库驱动
	"github.com/mattn/go-sqlite3"
)

var dbDao *zorm.DBDao

var dbDaoConfig = zorm.DataSourceConfig{
	DSN:                   sqliteDBfile,
	DriverName:            "sqlite3_simple", // 使用simple分词器会注册这个驱动名
	Dialect:               "sqlite",
	MaxOpenConns:          1,
	MaxIdleConns:          1,
	ConnMaxLifetimeSecond: 600,
	SlowSQLMillis:         -1,
}

// checkSQLiteStatus 初始化sqlite数据库,并检查是否成功
func checkSQLiteStatus() bool {
	const failSuffix = ".fail"
	if failDB := datadir + "minrag.db" + failSuffix; pathExist(failDB) {
		FuncLogError(nil, fmt.Errorf(funcT("Please confirm if [%s] needs to be manually renamed to [minrag.db]. If not, please manually delete [%s]"), failDB, failDB))
		return false
	}

	fts5File := datadir + "extensions/libsimple"
	vecFile := datadir + "extensions/vec0"
	//注册fts5的simple分词器,建议使用jieba分词
	//需要  --tags "fts5"
	sql.Register("sqlite3_simple", &sqlite3.SQLiteDriver{
		Extensions: []string{
			fts5File, //不要加后缀,它会自己处理,这样代码也统一
			vecFile,
		},
	})

	var err error
	dbDao, err = zorm.NewDBDao(&dbDaoConfig)
	if dbDao == nil || err != nil { //数据库初始化失败
		if db := datadir + "minrag.db"; pathExist(db) {
			_ = os.Rename(db, db+failSuffix)
		}
		return false
	}

	//初始化结巴分词的字典
	finder := zorm.NewFinder().Append("SELECT jieba_dict(?)", datadir+"dict")
	fts5jieba := ""
	_, err = zorm.QueryRow(context.Background(), finder, &fts5jieba)
	if err != nil {
		return false
	}

	finder = zorm.NewFinder().Append("select jieba_query(?)", "让数据自由一点点,让世界美好一点点")
	_, err = zorm.QueryRow(context.Background(), finder, &fts5jieba)
	if err != nil {
		return false
	}

	// 查询sqlite_vec版本
	var vecVersion string
	finder = zorm.NewFinder().Append("select vec_version()")
	_, err = zorm.QueryRow(context.Background(), finder, &vecVersion)
	if err != nil {
		panic(err)
	}
	//fmt.Println("vec_version:" + vecVersion)
	isInit := pathExist(datadir + "minrag.db")
	if !isInit { //需要初始化数据库
		return isInit
	}

	if tableExist(tableDocumentName) {
		return true
	}

	sqlByte, err := os.ReadFile("minragdatadir/minrag.sql")
	if err != nil {
		panic(err)
	}
	createTableSQL := string(sqlByte)
	if createTableSQL == "" {
		panic("minragdatadir/minrag.sql " + funcT("File anomaly"))
	}

	ctx := context.Background()
	_, err = execNativeSQL(ctx, createTableSQL)
	if err != nil {
		panic(err)
	}

	return true
}

// tableExist 数据表是否存在
func tableExist(tableName string) bool {
	finder := zorm.NewSelectFinder("sqlite_master", "count(*)").Append("WHERE type=? and name=?", "table", tableName)
	count := 0
	zorm.QueryRow(context.Background(), finder, &count)
	return count > 0
}

// deleteById 根据Id删除数据
func deleteById(ctx context.Context, tableName string, id string) error {
	finder := zorm.NewDeleteFinder(tableName).Append(" WHERE id=?", id)
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		_, err := zorm.UpdateFinder(ctx, finder)
		return nil, err
	})

	return err
}

// deleteAll 删除所有数据
func deleteAll(ctx context.Context, tableName string) error {
	finder := zorm.NewDeleteFinder(tableName)
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		_, err := zorm.UpdateFinder(ctx, finder)
		return nil, err
	})

	return err
}

// execNativeSQL 执行SQL语句
func execNativeSQL(ctx context.Context, nativeSQL string) (bool, error) {
	finder := zorm.NewFinder().Append(nativeSQL)
	finder.InjectionCheck = false
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		_, err := zorm.UpdateFinder(ctx, finder)
		return nil, err
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// vecSerializeFloat64 Serializes a float64 list into a vector BLOB that sqlite-vec accepts
func vecSerializeFloat64(vector []float64) ([]byte, error) {
	vector32 := make([]float32, len(vector))
	for i, v := range vector {
		vector32[i] = float32(v)
	}
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, vector32)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
