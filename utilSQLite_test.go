// Copyright (c) 2025 minrag Authors.
//
// This file is part of minrag.
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
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"testing"

	"gitee.com/chunanyong/zorm"
)

func TestVecCreate(t *testing.T) {
	ctx := context.Background()
	// 查询sqlite_vec版本
	var vecVersion string
	finder := zorm.NewFinder().Append("select vec_version()")
	_, err := zorm.QueryRow(ctx, finder, &vecVersion)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("vec_version:" + vecVersion)
	_, err = zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewFinder().Append("CREATE VIRTUAL TABLE vec_items USING vec0(embedding float[4])")
		//return的error如果不为nil,事务就会回滚
		zorm.UpdateFinder(ctx, finder)
		return nil, nil
	})

	if err != nil {
		t.Fatal(err)
	}
	items := map[int][]float32{
		1: {0.1, 0.1, 0.1, 0.1},
		2: {0.2, 0.2, 0.2, 0.2},
		3: {0.3, 0.3, 0.3, 0.3},
		4: {0.4, 0.4, 0.4, 0.4},
		5: {0.5, 0.5, 0.5, 0.5},
	}
	for id, values := range items {
		v, err := vecSerializeFloat32(values)
		if err != nil {
			t.Fatal(err)
		}
		_, err = zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
			finder := zorm.NewFinder().Append("INSERT INTO vec_items(rowid, embedding) VALUES (?, ?)", id, v)
			return zorm.UpdateFinder(ctx, finder)
		})

		if err != nil {
			t.Fatal(err)
		}
	}

}

func TestVecQuery(t *testing.T) {

	q := []float32{0.3, 0.3, 0.3, 0.3}

	query, err := vecSerializeFloat32(q)
	if err != nil {
		t.Fatal(err)
	}

	finder := zorm.NewFinder().Append(`
	SELECT
		rowid,
		distance
	FROM vec_items
	WHERE embedding MATCH ?
	ORDER BY distance
	LIMIT 3
`, query)

	datas, err := zorm.QueryMap(context.Background(), finder, nil)
	if err != nil {
		t.Fatal(err)
	}
	for rowid, distance := range datas {
		fmt.Printf("rowid=%v, distance=%v\n", rowid, distance)
	}

}
