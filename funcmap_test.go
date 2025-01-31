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
	"reflect"
	"testing"
)

func TestDataSliceKnowledgeBase2Tree(t *testing.T) {
	type args struct {
		categories []*KnowledgeBase
	}
	tests := []struct {
		name string
		args args
		want []*KnowledgeBase
	}{
		{
			name: "无节点",
			args: args{
				categories: nil,
			},
			want: []*KnowledgeBase{},
		},
		{
			name: "两级节点",
			args: args{
				categories: []*KnowledgeBase{
					{Id: "1", Name: "KnowledgeBase 1", Pid: ""},
					{Id: "2", Name: "KnowledgeBase 2", Pid: "1"},
					{Id: "3", Name: "KnowledgeBase 3", Pid: ""},
					{Id: "4", Name: "KnowledgeBase 4", Pid: "3"},
				},
			},
			want: []*KnowledgeBase{
				{
					Id:   "1",
					Name: "KnowledgeBase 1",
					Leaf: []*KnowledgeBase{{Id: "2", Name: "KnowledgeBase 2", Pid: "1"}},
				},
				{
					Id:   "3",
					Name: "KnowledgeBase 3",
					Leaf: []*KnowledgeBase{{Id: "4", Name: "KnowledgeBase 4", Pid: "3"}},
				},
			},
		},
		{
			name: "多级节点",
			args: args{
				categories: []*KnowledgeBase{
					{Id: "1", Name: "KnowledgeBase 1", Pid: ""},
					{Id: "2", Name: "KnowledgeBase 2", Pid: "1"},
					{Id: "3", Name: "KnowledgeBase 3", Pid: "1"},
					{Id: "4", Name: "KnowledgeBase 4", Pid: "2"},
					{Id: "5", Name: "KnowledgeBase 5", Pid: "2"},
					{Id: "6", Name: "KnowledgeBase 6", Pid: "3"},
				},
			},
			want: []*KnowledgeBase{
				{
					Id:   "1",
					Name: "KnowledgeBase 1",
					Leaf: []*KnowledgeBase{
						{
							Id:   "2",
							Pid:  "1",
							Name: "KnowledgeBase 2",
							Leaf: []*KnowledgeBase{
								{
									Id:   "4",
									Pid:  "2",
									Name: "KnowledgeBase 4",
								},
								{
									Id:   "5",
									Pid:  "2",
									Name: "KnowledgeBase 5",
								},
							},
						},
						{
							Id:   "3",
							Pid:  "1",
							Name: "KnowledgeBase 3",
							Leaf: []*KnowledgeBase{
								{
									Id:   "6",
									Pid:  "3",
									Name: "KnowledgeBase 6",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "多颗树",
			args: args{
				categories: []*KnowledgeBase{
					{Id: "1", Name: "KnowledgeBase 1", Pid: ""},
					{Id: "2", Name: "KnowledgeBase 2", Pid: "1"},
					{Id: "3", Name: "KnowledgeBase 3", Pid: ""},
					{Id: "4", Name: "KnowledgeBase 4", Pid: "3"},
					{Id: "5", Name: "KnowledgeBase 5", Pid: ""},
					{Id: "6", Name: "KnowledgeBase 6", Pid: "5"},
				},
			},
			want: []*KnowledgeBase{
				{
					Id:   "1",
					Name: "KnowledgeBase 1",
					Leaf: []*KnowledgeBase{
						{
							Id:   "2",
							Pid:  "1",
							Name: "KnowledgeBase 2",
						},
					},
				},
				{
					Id:   "3",
					Name: "KnowledgeBase 3",
					Leaf: []*KnowledgeBase{
						{
							Id:   "4",
							Pid:  "3",
							Name: "KnowledgeBase 4",
						},
					},
				},
				{
					Id:   "5",
					Name: "KnowledgeBase 5",
					Leaf: []*KnowledgeBase{
						{
							Id:   "6",
							Pid:  "5",
							Name: "KnowledgeBase 6",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sliceKnowledgeBase2Tree(tt.args.categories); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sliceKnowledgeBase2Tree() = %v, want %v", got, tt.want)
			}
		})
	}
}
