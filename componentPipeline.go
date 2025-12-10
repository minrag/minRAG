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
	"sync"
	"text/template"

	"gitee.com/chunanyong/zorm"
)

/**

flowchart TD
    A[流水线启动] --> B[遍历pipelineComponentMap<br>初始化所有组件实例]
    B --> C[为每个组件实例<br>加载其下游节点完整对象]
    C --> D[从根节点开始<br>（UpStream为空的组件）]
    D --> E{检查UpStreamCondition<br>是否全部满足？}
    E -- 是 --> F[执行当前组件核心逻辑]
    E -- 否 --> G[标记当前组件状态为“阻塞”]
    F --> H[更新组件状态为“完成”]
    G --> I[等待条件满足或超时]
    I -.-> E
    H --> J[通知所有下游节点<br>（DownStream中的组件）]
    J --> K{是否所有下游节点<br>均已执行完毕？}
    K -- 否 --> D
    K -- 是 --> L[整条流水线执行结束]

**/

/**
流水线组件有ID和基础组件ID,基础组件ID,对应组件表的ID,为空时,默认为ID.比如一个基础组件被多次引用,ID不同,BaseComponentId相同
所有组件都放到了pipelineComponentMap map[流水线组件id]*PipelineComponent,每个流水线组件的所有组件都是互相隔离的,都是新的实例

上游节点UpStream和下游节点DownStream,都是[]*PipelineComponent类型,上游节点默认为空,当有多个节点时,要全部完成才能进行下游节点.有upStreamCondition时,UpStream必须有值
从UpStreamCondition map[upStreamID]Condition 获取上游节点进入的表达式,组件运行前先验证表达式是否通过,可以为空. 例如 "{{.size}}>100"

UpStream和DownStream的只有节点的ID,完整对象从pipelineComponentMap获取,例如:  "downStream":[{"id":"FtsKeywordRetriever"}]

有参数Parameter,json格式字符串.如果有值,必须是完整的参数,为空可用只保留id,如果流水线里有多个相同基础组件的组件,必须指定BaseComponentId,使用ID来区分不同的组件实例

**/

// PipelineComponent 流水线组件的结构体
type PipelineComponent struct {
	// Id 流水线组件的ID,唯一标识
	Id string `json:"id,omitempty"`
	// BaseComponentId 基础组件ID,对应组件表的ID,为空时,默认为ID.比如一个基础组件被多次引用,ID不同,BaseComponentId相同
	BaseComponentId string `json:"baseComponentId,omitempty"`

	// Parameter 参数,json格式字符串.如果有值,必须是完整的参数,为空可用只保留id,从map中获取
	Parameter string `json:"parameter,omitempty"`

	// UpStream 上游组件,必须上游组件都执行完成后,才会执行当前组件.默认为空,只有一个上游时,可以为空.有upStreamCondition时,UpStream必须有值
	// 流水线组件都在pipelineComponentMap[Id]*PipelineComponent,每个流水线的所有组件都是互相隔离的,都是新的实例.
	// UpStream和DownStream的只有节点的ID,完整对象从pipelineComponentMap获取,例如:  "upStream":[{"id":"FtsKeywordRetriever"}]
	UpStream []*PipelineComponent `json:"upStream,omitempty"`
	// DownStreamCondition  map[downStreamId]Condition  下游组件条件表达式,先验证表达式是否通过,可以为空. 例如 "{{.size}}>100"
	DownStreamCondition map[string]string `json:"downStreamCondition,omitempty"`

	// DownStream 下游组件,多个节点时,一般指定runCondition,同时执行多个下游节点
	// UpStream和DownStream的只有节点的ID,完整对象从pipelineComponentMap获取,例如:  "downStream":[{"id":"FtsKeywordRetriever"}]
	DownStream []*PipelineComponent `json:"downStream,omitempty"`

	// Component 组件实例对象,运行时使用
	Component IComponent `json:"-"`

	// Status 组件状态,0未开始,1进行中,2阻塞,3完成,4失败
	Status int `json:"-"`
}

// Pipeline 流水线,也是IComponent实现
type Pipeline struct {

	// 引入流水线组件
	PipelineComponent

	// pipelineComponentMap map[流水线组件id]*PipelineComponent
	pipelineComponentMap map[string]*PipelineComponent `json:"-"`
}

func (pipeline *Pipeline) Initialization(ctx context.Context, input map[string]any) error {
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
func (pipeline *Pipeline) initPipelineComponentMap(ctx context.Context, input map[string]any) error {
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

		// 记录到map中
		pipeline.pipelineComponentMap[pipelineComponent.Id] = pipelineComponent

	}

	return nil
}

func (pipeline *Pipeline) Run(ctx context.Context, input map[string]any) error {
	// 流水线的第一个组件,作为开始的组件
	return runComponent(ctx, input, pipeline.DownStream[0], pipeline.pipelineComponentMap)
}

// runComponent 运行组件
func runComponent(ctx context.Context, input map[string]any, currPipelineComponent *PipelineComponent, pipelineComponentMap map[string]*PipelineComponent) error {
	isRun := true
	for _, upStream := range currPipelineComponent.UpStream {
		status := upStream.Status
		if status != 3 { //没有完成
			isRun = false
			break
		}
	}

	if isRun { // 可以执行
		currPipelineComponent.Status = 0 //重置为未开始
	} else {
		currPipelineComponent.Status = 2 //阻塞
		return nil                       // 还有上游组件没有执行完,跳过
	}

	currPipelineComponent.Status = 1 //进行中
	err := currPipelineComponent.Component.Run(ctx, input)
	if err != nil {
		currPipelineComponent.Status = 4 //失败
		FuncLogError(ctx, err)
		input[errorKey] = err
		return err
	}
	if input[errorKey] != nil {
		return input[errorKey].(error)
	}
	if input[endKey] != nil {
		return nil
	}
	currPipelineComponent.Status = 3 //完成
	// 所有的下游节点
	downStream := currPipelineComponent.DownStream
	downStreamCondition := currPipelineComponent.DownStreamCondition
	// 使用WaitGroup异步方案
	var wg sync.WaitGroup
	for i := range downStream {
		id := downStream[i].Id //组件id
		if pipelineComponentMap[id] == nil {
			return fmt.Errorf(funcT("The %s component of the pipeline does not exist"), id)
		}
		downPipelineComponent := pipelineComponentMap[id]
		if downPipelineComponent.Status != 0 {
			continue
		}
		// @TODO if elseif 这样的逻辑关系  有 条件组件 完成,这里只处理简单的表达式逻辑
		// 验证下游的表达式
		condition, has := downStreamCondition[id]
		if has && condition != "" {
			tmpl := template.New("pipelineComponentMap-" + id + "-" + downPipelineComponent.Id)
			t, err := tmpl.Parse(condition)
			if err != nil {
				FuncLogError(ctx, err)
				input[errorKey] = err
				return err
			}
			// 使用text/template进行表达式计算
			// 创建一个 bytes.Buffer 用于存储渲染后的 text 内容
			var buf bytes.Buffer
			// 执行模板并将结果写入到 bytes.Buffer
			if err := t.Execute(&buf, input); err != nil {
				FuncLogError(ctx, err)
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

		//异步并行执行downStream的组件
		//@TODO 组件如果有输出,会有乱序,组件需要增加参数控制是否输出
		wg.Go(func() {
			runComponent(ctx, input, downPipelineComponent, pipelineComponentMap)
		})

	}
	wg.Wait() // 等待所有goroutine完成
	return nil
}

// findPipelineById 根据ID查找流水线组件
func findPipelineById(ctx context.Context, pipelineId string, input map[string]any) (*Pipeline, error) {
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
