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
	"encoding/json"
)

var functionCallingMap = make(map[string]IToolFunctionCalling, 0)

func init() {
	fcWeather := FCWeather{}
	fc, err := fcWeather.Initialization(context.TODO(), get_weather_json)
	if err == nil {
		functionCallingMap["get_weather"] = fc
	}
}

// IToolFunctionCalling 函数调用接口
type IToolFunctionCalling interface {
	// Initialization 初始化方法
	Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error)
	//获取描述的Map
	Description(ctx context.Context) interface{}
	// Run 执行方法
	Run(ctx context.Context, arguments string) (string, error)
}

var get_weather_json = `{
	"type": "function",
	"function": {
		"name": "get_weather",
		"description": "Get weather of an location, the user shoud supply a location first",
		"parameters": {
			"type": "object",
			"properties": {
				"location": {
					"type": "string",
					"description": "The city and state, e.g. San Francisco, CA"
				}
			},
			"required": ["location"]
		}
	}
}`

// FCWeather 天气函数
type FCWeather struct {
	//接受模型返回的 arguments
	Location       string                 `json:"location,omitempty"`
	DescriptionMap map[string]interface{} `json:"-"`
}

func (fc FCWeather) Initialization(ctx context.Context, descriptionJson string) (IToolFunctionCalling, error) {
	dm := make(map[string]interface{})
	if descriptionJson == "" {
		return fc, nil
	}
	err := json.Unmarshal([]byte(descriptionJson), &dm)
	if err != nil {
		return fc, err
	}
	fc.DescriptionMap = dm
	return fc, nil
}

// 获取描述的Map
func (fc FCWeather) Description(ctx context.Context) interface{} {
	return fc.DescriptionMap
}

// Run 执行方法
func (fc FCWeather) Run(ctx context.Context, arguments string) (string, error) {
	if arguments != "" {
		err := json.Unmarshal([]byte(arguments), &fc)
		if err != nil {
			return "", nil
		}
	}
	return fc.Location + "的气温是25度", nil
}
