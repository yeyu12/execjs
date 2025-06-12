# go-exec-js
- [go-exec-js](#go-exec-js)
  - [Introduction](#introduction)
  - [Requirement](#requirement)
  - [Intall](#intall)
  - [Usage](#usage)
  - [Thanks to](#thanks-to)
## Introduction
参考PyExecJS，提供了在go语言里调用javascript的能力。由于go是强类型语言，所以在迁移的时候使用`interface{}`作为替代传递任意类型的参数。

## Requirement
需要本地环境变量中, 有node路径, PATH 环境变量中, 最好添加到第一行中
最好也添加上 node_modules 路径, NODE_PATH 环境变量

## Install
```
go get -u github.com/yeyu2/execjs
```
## Usage
可以使用Eval方法获取表达式的值，这将输出`12`
```go
execjs.Register(execjs.Node, execjs.NodeCommand())
output, err := execjs.Eval(`1+"2"`)
if err != nil {
    log.Fatal(err)
}
fmt.Println(output)
```
可以使用Compile方法编译一个Context，然后调用。这将输出`3`
```go
execjs.Register(execjs.Node, execjs.NodeCommand())
c, err := execjs.Compile(`function add(x, y) {
    return x + y;
    }`)
if err != nil {
    log.Fatal(err)
}
output, err := c.Call("add", 1,2)
if err != nil {
    log.Fatal(err)
}
fmt.Println(output)
```
更多用法参见测试文件`execjs_test`
> 注意：因为返回的是`interface{}`类型的变量，使用时要进行类型断言，例如
```go
output.(string) //这将输出值变为string类型
output.([]interface{}) //这将输出值变为slice类型
```
## Thanks to
[PyExecJS](https://github.com/doloopwhile/PyExecJS)
