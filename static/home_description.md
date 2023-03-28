这段代码是一个使用Vue.js框架实现的Todo List应用程序的前端代码。它使用了Bootstrap和Font Awesome库来实现样式和图标。以下是代码的一些关键点：

- v-model指令用于将输入框的值绑定到Vue实例中的todo.title属性。
- v-on:keyup指令用于在用户按下回车键时调用checkForEnter方法。
- v-for指令用于在待办事项数组上循环，以便将每个待办事项渲染为列表项。
- v-on:click指令用于在用户单击列表项时调用toggleTodo、editTodo或deleteTodo方法。
- :class指令用于根据待办事项的状态动态设置列表项的类。
- :class指令还用于根据showError属性动态设置输入框的类。
# 首页模板

这是一个简单的待办事项应用程序的首页模板。它包含一个简单的表格，用于显示待办事项列表。用户可以通过单击“添加任务”按钮添加新任务。

## 模板变量

- `Title`：页面标题
- `Todos`：待办事项列表，每个待办事项包含以下属性：
    - `ID`：唯一标识符
    - `Title`：任务标题
    - `Completed`：任务是否已完成

## 模板函数

- `urlFor`：生成指定路由名称的URL。例如，`urlFor("home")`将生成首页的URL。
- `form`：生成HTML表单。它接受一个表单对象作为参数，该对象包含以下属性：
    - `Action`：表单提交的URL
    - `Method`：表单提交的HTTP方法（例如，GET或POST）
    - `Fields`：表单字段列表，每个字段包含以下属性：
        - `Name`：字段名称
        - `Type`：字段类型（例如，text或checkbox）
        - `Value`：字段值
        - `Label`：字段标签
        - `Required`：字段是否必填

## 示例

```go
// 在控制器中渲染home.tpl模板
func (c *Controller) Home(ctx *web.Context) {
    // 获取待办事项列表
    todos := c.TodoService.GetAll()

    // 渲染模板
    ctx.Data["Title"] = "待办事项"
    ctx.Data["Todos"] = todos
    ctx.Data["Form"] = form{
        Action: "/todos",
        Method: "POST",
        Fields: []field{
            {
                Name:     "title",
                Type:     "text",
                Label:    "任务标题",
                Required: true,
            },
            {
                Name:  "completed",
                Type:  "checkbox",
                Label: "已完成",
            },
        },
    }
    ctx.HTML(200, "home")
}
