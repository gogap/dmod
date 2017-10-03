DMOD
====

Dynamic model Lib

动态模型类库

-----------------------

# 样例讲解

## 基于 Gorm，零代码建立报表查询


### 创建模型管理

```go
models, err := dmod.NewModels()
```

### 注册额外的类型

```go
models.StructBuilder().RegisterTypes(
	dmod.NameType{"sql.NullString", &sql.NullString{}},
	dmod.NameType{"sql.NullInt64", &sql.NullInt64{}},
)
```

### 自动组合

```go
type Table struct {
	TabName string `json:"-" gorm:"-"`
}

func (p Table) TableName() string {
	return p.TabName
}
```

```go
models.CombineMapper().Register("user",
		func(id string, field []dmod.Field) map[string]interface{} {
			m := map[string]interface{}{
				".":                Table{TabName: "users"},
				".BillingAddress":  Table{TabName: "addresses"},
				".ShippingAddress": Table{TabName: "addresses"},
			}
			return m
		},
	)
```

通过 `CombineMapper` 可以在生成对象时，自动将`Table`注入到`user`这个模型里进行组合，其中 `.BillingAddress` 代表往 `user.BillingAddress` 这个字段里注入，因为这个字段本身就是一个结构体

这里组合 `TableName` 可以让`gorm`读到表名称

### 加载模块

#### 从目录加载

```go
dir := "/gopath/src/git/zeal/playgo/dmod_models"
models.LoadFromDir(dir)
```

以下为 `dmod_models` 目录下的模块配置

```bash
> tree dmod_models
dmod_models
├── address.json
├── email.json
├── gorm.model.json
├── language.json
└── user.json
```

`gorm.model.json`

```json
{
    "name": "gorm.model",
    "fields": [{
        "name": "ID",
        "type": "uint",
        "tag": "gorm:\"primary_key\""
    }, {
        "name": "CreatedAt",
        "type": "time.Time"
    }, {
        "name": "UpdateAt",
        "type": "time.Time"
    }, {
        "name": "DeletedAt",
        "type": "*time.Time"
    }]
}
```

`language.json`

```json
{
    "name": "language",
    "fields": [{
        "name": "ID",
        "type": "int",
        "tag": "gorm:\"primary_key;AssociationForeignColumn:LanguageID\""
    }, {
        "name": "Name",
        "type": "string"
    }, {
        "name": "Code",
        "type": "string"
    }]
}
```

`email.json`

```json
{
    "name": "email",
    "fields": [{
        "name": "ID",
        "type": "int"
    }, {
        "name": "UserID",
        "type": "int",
        "tag": "gorm:\"index\""
    }, {
        "name": "Email",
        "type": "string",
        "tag": "gorm:\"type:varchar(100);unique_index\""
    }, {
        "name": "Subscribed",
        "type": "bool"
    }]
}
```
`address.json`

```
{
    "name": "address",
    "fields": [{
        "name": "ID",
        "type": "int"
    }, {
        "name": "Address1",
        "type": "string",
        "tag": "gorm:\"not null;unique\""
    }, {
        "name": "Address2",
        "type": "string",
        "tag": "gorm:\"type:varchar(100);unique\""
    }, {
        "name": "Post",
        "type": "sql.NullString",
        "tag":"gorm:\"not null\""
    }]
}
```

`user.json`

```
{
    "name": "user",
    "extends":["gorm.model"],
    "fields": [{
        "name": "Birthday",
        "type": "time.Time"
    }, {
        "name": "Age",
        "type": "int"
    }, {
        "name": "Name",
        "type": "string",
        "tag": "gorm:\"size:255\""
    }, {
        "name": "Num",
        "type": "int",
        "tag": "gorm:\"AUTO_INCREMENT\""
    }, {
        "name": "BillingAddressID",
        "type": "sql.NullInt64"
    },{
        "name": "BillingAddress",
        "Ref":"address",
        "tag": "gorm:\"ForeignKey:BillingAddressID\""
    }, {
        "name": "ShippingAddressID",
        "type": "sql.NullInt64"
    },{
        "name": "ShippingAddress",
        "Ref":"address",
        "tag": "gorm:\"ForeignKey:ShippingAddressID\""
    }, {
        "name": "Emails",
        "ref": "email",
        "array": true,
        "tag": "gorm:\"ForeignKey:UserID\""
    }, {
        "name": "Languages",
        "ref": "language",
        "array": true,
        "tag": "gorm:\"many2many:user_languages;ForeignColumn:UserID\""
    }]
}
```

#### 从文件加载

```
models.LoadFromFiles(
	"/gopath/src/git/zeal/playgo/gorm.model.json",
	"/gopath/src/git/zeal/playgo/email.json")
```

随时可以动态加载，会覆盖之前的Model，新生成出来的实例会被替换掉

### 数据填充

以下代码为Demo，`db` 对象可以放到`html/template`里执行和渲染，因此当框架完成后，不需要写一行代码，就可以完成基本的简易报表的查询


> 使用DMOD结合`gorm`的方法:

```go
u := db.Find(user, "id = ?", 2)

// 获取 BillingAddress 数据
u.Association("BillingAddress").Find(userModel.Field(user, ".BillingAddress").Interface())

// 获取 ShippingAddress 数据
u.Association("ShippingAddress").Find(userModel.Field(user, ".ShippingAddress").Interface())

// 获取 Emails 数据
u.Association("Emails").Find(userModel.Field(user, ".Emails").Interface())

// 获取 Languages 数据
u.Association("Languages").Find(userModel.Field(user, ".Languages").Interface())
```

> 使用普通模式结合`gorm`的方法

```
user := User{} // 普通模式必须定义结构体

u := db.Find(&user, "id = ?", 2)

u.Association("BillingAddress").Find(&user.BillingAddress)
u.Association("ShippingAddress").Find(&user.ShippingAddress)
u.Association("Emails").Find(&user.Emails)
u.Association("Languages").Find(&user.Languages)
```


输出结果:

```json
{
    "Birthday": "0001-01-01T00:00:00Z",
    "Age": 0,
    "Name": "jinzhu",
    "Num": 0,
    "BillingAddressID": {
        "Int64": 5,
        "Valid": true
    },
    "BillingAddress": {
        "ID": 5,
        "Address1": "Billing Address - Address 1",
        "Address2": "",
        "Post": {
            "String": "",
            "Valid": false
        }
    },
    "ShippingAddressID": {
        "Int64": 6,
        "Valid": true
    },
    "ShippingAddress": {
        "ID": 6,
        "Address1": "Shipping Address - Address 1",
        "Address2": "",
        "Post": {
            "String": "",
            "Valid": false
        }
    },
    "Emails": [
        {
            "ID": 3,
            "UserID": 2,
            "Email": "jinzhu@example.com",
            "Subscribed": false
        },
        {
            "ID": 4,
            "UserID": 2,
            "Email": "jinzhu-2@example@example.com",
            "Subscribed": false
        }
    ],
    "Languages": [
        {
            "ID": 2,
            "Name": "EN",
            "Code": ""
        },
        {
            "ID": 1,
            "Name": "ZH",
            "Code": ""
        }
    ],
    "ID": 2,
    "CreatedAt": "2017-09-04T10:20:53+08:00",
    "UpdateAt": "0001-01-01T00:00:00Z",
    "DeletedAt": null
}
```