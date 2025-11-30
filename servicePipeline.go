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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"gitee.com/chunanyong/zorm"
)

/**

flowchart TD
    A[流水线启动] --> B[遍历pipelineComponentMap<br>初始化组件实例]
    B --> C{运行表达式<br>RunExpression是否为空?}
    C -- 是 --> D[表达式验证通过]
    C -- 否 --> E[执行表达式引擎计算]
    E --> F{表达式结果是否为真?}
    F -- 否 --> G[标记当前组件状态为“跳过”]
    F -- 是 --> D
    G --> H
    D --> H{检查上游依赖<br>UpStream是否为空?}
    H -- 是 --> I[所有上游节点均已完成]
    H -- 否 --> J[等待所有UpStream节点<br>执行完成]
    J --> I
    I --> K[执行当前组件核心逻辑]
    K --> L[更新组件状态为“完成”<br>并通知下游DownStream节点]

**/

/**
流水线组件有ID和基础组件ID,基础组件ID,对应组件表的ID,为空时,默认为ID.比如一个基础组件被多次引用,ID不同,BaseComponentId相同
所有组件都放到了pipelineComponentMap map[流水线组件id]*PipelineComponent,每个流水线组件的所有组件都是互相隔离的,都是新的实例

有运行表达式RunExpression,组件运行前先验证表达式是否通过,可以为空. 例如 "{{.size}}>100"

有上游节点UpStream和下游节点DownStream,都是[]*PipelineComponent类型,上游节点默认为空,当有多个节点时,要全部完成才能进行下游节点
下游节点DownStream的JSON中只写下游节点的ID,完整对象从pipelineComponentMap获取,例如:  "downStream":[{"id":"FtsKeywordRetriever"}]

有参数Parameter,json格式字符串.如果有值,必须是完整的参数,为空可用只保留id,如果流水线里有多个相同基础组件的组件,必须使用BaseComponentId,使用ID来区分不同的组件实例

**/

// PipelineComponent 流水线组件的结构体
type PipelineComponent struct {
	// Id 流水线组件的ID,唯一标识
	Id string `json:"id,omitempty"`
	// BaseComponentId 基础组件ID,对应组件表的ID,为空时,默认为ID.比如一个基础组件被多次引用,ID不同,BaseComponentId相同
	BaseComponentId string `json:"baseComponentId,omitempty"`

	// Parameter 参数,json格式字符串.如果有值,必须是完整的参数,为空可用只保留id,从map中获取
	Parameter string `json:"parameter,omitempty"`

	// RunExpression 运行表达式,组件运行时先验证表达式是否通过,可以为空. 例如 "{{.size}}>100"
	RunExpression string             `json:"runExpression,omitempty"`
	t             *template.Template `json:"-"`

	// 流水线里的所有组件都放到一个map<Id,PipelineComponent>,可以根据ID获取单例,避免使用指针,因为每个流水线的组件要互相隔离
	// UpStream 上游组件,必须上游组件都执行完成后,才会执行当前组件.默认为空,只有一个上游时,可以为空
	UpStream []*PipelineComponent `json:"upstream,omitempty"`

	// DownStream 下游组件,多个节点时,一般指定runExpression,同时执行多个下游节点
	// JSON中只写下游节点的ID,完整对象从pipelineComponentMap获取,例如:  "downStream":[{"id":"FtsKeywordRetriever"}]
	DownStream []*PipelineComponent `json:"downstream,omitempty"`

	// Component 组件实例对象,运行时使用
	Component IComponent `json:"-"`
}

// Pipeline 流水线,也是IComponent实现
type Pipeline struct {

	// 引入组件Struct
	PipelineComponent

	// pipelineComponentMap map[流水线组件id]*PipelineComponent
	pipelineComponentMap map[string]*PipelineComponent `json:"-"`
}

func (pipeline *Pipeline) Initialization(ctx context.Context, input map[string]interface{}) error {
	// 初始化流水线的组件map
	pipeline.pipelineComponentMap = make(map[string]*PipelineComponent, 0)
	// 获取上游组件
	for _, up := range pipeline.UpStream {
		baseComponentId := up.BaseComponentId
		if baseComponentId == "" {
			baseComponentId = up.Id
		}
		up.Component = baseComponentMap[baseComponentId]
		if up.Component == nil { //查找流水线
			pipeline, err := findPipelineById(ctx, baseComponentId, input)
			if err != nil && pipeline != nil {
				up.Component = pipeline
			}
		}
		pipeline.pipelineComponentMap[up.Id] = up
	}
	// 获取下游组件,并初始化pipelineComponentMap
	pipeline.initPipelineComponentMap(ctx, input)

	return nil
}

// initPipelineComponentMap 初始化流水线的组件map,递归处理
func (pipeline *Pipeline) initPipelineComponentMap(ctx context.Context, input map[string]interface{}) error {
	for i := 0; i < len(pipeline.DownStream); i++ {
		pipelineComponent := pipeline.DownStream[i]
		baseComponentId := pipelineComponent.BaseComponentId //基础组件id
		if baseComponentId == "" {                           // 没有设置基础组件id,默认使用当前组件id
			baseComponentId = pipelineComponent.Id
		}
		if pipelineComponent.Parameter == "" { // 没有参数,直接从公共map获取
			pipelineComponent.Component = baseComponentMap[baseComponentId]
		} else {
			baseComponent := baseComponentMap[baseComponentId]
			// 使用反射动态创建一个结构体的指针实例
			cType := reflect.TypeOf(baseComponent).Elem()
			cPtr := reflect.New(cType)
			// 将反射对象转换为接口类型
			pipelineComponent.Component = cPtr.Interface().(IComponent)
			//有参数,进行实例化
			err := json.Unmarshal([]byte(pipelineComponent.Parameter), pipelineComponent.Component)
			if err != nil {
				FuncLogError(ctx, err)
				continue
			}
			//初始化组件
			pipelineComponent.Component.Initialization(ctx, input)
		}
		if pipelineComponent.RunExpression != "" {
			tmpl := template.New("pipelineComponentMap-" + pipelineComponent.Id)
			var err error
			pipelineComponent.t, err = tmpl.Parse(pipelineComponent.RunExpression)
			if err != nil {
				FuncLogError(ctx, err)
				continue
			}
		}
		pipeline.pipelineComponentMap[pipelineComponent.Id] = pipelineComponent
		/*
			//按照顺序包装所有的组件,等于两层处理,不再递归处理
			cs := make([]*PipelineComponent, 0)
			if pipelineComponent.UpStream != nil {
				cs = append(cs, pipelineComponent.UpStream...)
			}
			if pipelineComponent.DownStream != nil {
				cs = append(cs, pipelineComponent.DownStream...)
			}

			// 递归处理
			if len(cs) > 0 {
				initPipelineComponentMap(ctx, input, cs, pipelineComponentMap)
			}
		*/

	}

	return nil
}

func (pipeline *Pipeline) Run(ctx context.Context, input map[string]interface{}) error {
	// 流水线的第一个组件,作为开始的组件
	downStream := make([]*PipelineComponent, 0)
	downStream = append(downStream, pipeline.DownStream[0])
	return runProcess(ctx, input, &pipeline.PipelineComponent, downStream, pipeline.pipelineComponentMap)
}
func runProcess(ctx context.Context, input map[string]interface{}, upStream *PipelineComponent, downStream []*PipelineComponent, pipelineComponentMap map[string]*PipelineComponent) error {
	if len(downStream) < 1 {
		return nil
	}
	for i := 0; i < len(downStream); i++ {
		id := downStream[i].Id //组件id
		if pipelineComponentMap[id] == nil {
			return fmt.Errorf(funcT("The %s component of the pipeline does not exist"), id)
		}
		pipelineComponent := pipelineComponentMap[id]
		// 使用text/template进行表达式计算
		if pipelineComponent.t != nil {
			// 创建一个 bytes.Buffer 用于存储渲染后的 text 内容
			var buf bytes.Buffer
			// 执行模板并将结果写入到 bytes.Buffer
			if err := pipelineComponent.t.Execute(&buf, input); err != nil {
				input[errorKey] = err
				return err
			}
			// 获取编译后的内容
			result := strings.TrimSpace(buf.String())
			// 如果结果不是 true,则跳过该组件的执行
			if strings.ToLower(result) != "true" {
				continue
			}

		}
		if len(pipelineComponent.UpStream) > 0 { // 有上游组件,需要把上游组件传递过来,从数组里删除
			//remove(component.UpStream, upStreamId)
			upId := upStream.Id
			index := -1
			for j := 0; j < len(pipelineComponent.UpStream); j++ {
				if pipelineComponent.UpStream[index].Id == upId {
					index = j
					break
				}
			}
			if index >= 0 {
				pipelineComponent.UpStream = append(pipelineComponent.UpStream[:index], pipelineComponent.UpStream[index+1:]...)
			}
			if len(pipelineComponent.UpStream) > 0 {
				continue // 还有上游组件没有执行完,跳过
			}

		}

		err := pipelineComponent.Component.Run(ctx, input)
		if err != nil {
			input[errorKey] = err
			return err
		}
		if input[errorKey] != nil {
			return input[errorKey].(error)
		}
		if input[endKey] != nil {
			return nil
		}
		nextComponens := pipelineComponent.DownStream
		nextComponentObj, has := input[nextComponentKey]
		if has && nextComponentObj.(string) != "" {
			nextComponens = make([]*PipelineComponent, 0)
			nextComponentId := nextComponentObj.(string)
			nextComponent := pipelineComponentMap[nextComponentId]
			if nextComponent == nil {
				return fmt.Errorf(funcT("The %s component of the pipeline does not exist"), nextComponentId)
			}
			nextComponens = append(nextComponens, nextComponent)
		}

		if len(nextComponens) > 0 {
			err := runProcess(ctx, input, pipelineComponent, nextComponens, pipelineComponentMap)
			if err != nil {
				FuncLogError(ctx, err)
				return err
			}
		}
	}
	return nil
}

// findPipelineById 根据ID查找流水线组件
func findPipelineById(ctx context.Context, pipelineId string, input map[string]interface{}) (*Pipeline, error) {
	// 流水线组件,以后有可以单独初始化一个,不用启动时全部初始化
	finderPipeline := zorm.NewSelectFinder(tableComponentName).Append("WHERE status=1 and componentType=? and id=? ", "Pipeline", pipelineId)
	finderPipeline.SelectTotalCount = false
	pipeline := &Pipeline{}
	component := &Component{}
	has, err := zorm.QueryRow(ctx, finderPipeline, component)
	if err != nil || !has {
		return pipeline, err
	}
	// 没有初始化参数,直接返回
	if component.Parameter == "" {
		return pipeline, err
	}
	//初始化组件实例
	err = json.Unmarshal([]byte(component.Parameter), pipeline)
	if err != nil {
		FuncLogError(ctx, err)
		return pipeline, err
	}
	// 设置一个默认的ID
	if pipeline.Id == "" {
		pipeline.Id = component.Id
	}
	err = pipeline.Initialization(ctx, input)
	return pipeline, err

}
