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
	"os"
	"testing"
)

func Test_unzip(t *testing.T) {
	type args struct {
		zipPath string
		destDir string
	}
	tests := []struct {
		name         string
		args         args
		wantExistDir bool
		wantDir      string
	}{
		{
			name: "解压成功 -> 创建或者覆盖目录和文件成功",
			args: args{
				zipPath: "./minragdatadir/minrag-geekdoc-master.zip",
				destDir: "minragdatadir/",
			},
			wantExistDir: true,
			wantDir:      "./minragdatadir/minrag-geekdoc-master/",
		},
		{
			name: "解压失败-原目录存在 -> 不进行任何处理",
			args: args{
				zipPath: "./minragdatadir/minrag-geekdoc-master2.zip",
				destDir: "minragdatadir/",
			},
			wantExistDir: true,
			wantDir:      "./minragdatadir/minrag-geekdoc-master/",
		},
		{
			name: "解压失败-原目录不存在 -> 删除原目录",
			args: args{
				zipPath: "./minragdatadir/minrag-geekdoc-master_副本.zip", // 这个副本 把里面的一个.html后缀的改成.zip 然后压缩
				destDir: "tmp/",
			},
			wantExistDir: false,
			wantDir:      "./tmp/minrag-geekdoc-master/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = unzip(tt.args.zipPath, tt.args.destDir)
			if _, err := os.Stat(tt.wantDir); (err == nil) != tt.wantExistDir {
				t.Errorf("unzip() error = %v, wantExistDir %v", err, tt.wantExistDir)
			}
		})
	}
}
