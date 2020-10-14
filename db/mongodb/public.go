package mongodb

import (
	"context"
	"encoding/hex"
	"errors"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// ErrMustSlice result argument must be a slice address
	ErrMustSlice = errors.New("result argument must be a slice address")
)

// ObjectID type
type ObjectID primitive.ObjectID

// MDoc mongo bson doc
type MDoc primitive.M

// DDoc mongo bson doc
type DDoc primitive.D

// ADoc mongo bson doc
type ADoc primitive.A

// Hex
func Hex(s ObjectID) string {
	return hex.EncodeToString([]byte(string(s[:])))
}

// NewObjectID 生成ObjectID
func NewObjectID() ObjectID {
	return ObjectID(primitive.NewObjectID())
}

// NewObjectIDFromTimestamp 根据时间戳生成ObjectID
func NewObjectIDFromTimestamp(t time.Time) ObjectID {
	return ObjectID(primitive.NewObjectIDFromTimestamp(t))
}

// ObjectIdHex returns an ObjectId from the provided hex representation.
// Calling this function with an invalid hex representation will
// cause a runtime panic. See the IsObjectIdHex function.
func ObjectIDFromHex(s string) ObjectID {
	oid, _ := primitive.ObjectIDFromHex(s)
	return ObjectID(oid)
}

// IsObjectIdHex returns whether s is a valid hex representation of
// an ObjectId. See the ObjectIdHex function.
func IsObjectIdHex(s string) bool {
	b, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	if len(b) != 12 {
		return false
	}
	return true
}

// Collection
type Collection struct {
	CollectionName string // 集合名称
}

// ICollection
type ICollection interface {
	GetCollection() *mongo.Collection
	FindOne(ctx context.Context, filter, result interface{}) error
	FindAll(ctx context.Context, findOptions *options.FindOptions, filter interface{}, resSlice interface{}) error
	InsertOne(ctx context.Context, data interface{}) (interface{}, error)
	InsertMany(ctx context.Context, data []interface{}) ([]interface{}, error)
	UpdateOne(ctx context.Context, filter, updateData interface{}) (int64, error)
	UpdateMany(ctx context.Context, filter, updateData interface{}) (int64, error)
	Delete(ctx context.Context, filter interface{}) (int64, error)
}

// NewCollection
func NewCollection(name string) ICollection {
	return &Collection{
		CollectionName: name,
	}
}

// GetCollection 获取文档对象
func (m *Collection) GetCollection() *mongo.Collection {
	return DB().Collection(m.CollectionName)
}

// FindOne https://docs.mongodb.com/manual/reference/command/find/.
func (m *Collection) FindOne(ctx context.Context, filter, result interface{}) error {
	err := m.GetCollection().FindOne(ctx, filter).Decode(result)
	if err != nil {
		return err
	}
	return nil
}

// FindAll 查询多个文档
// findOptions := options.Find()
// findOptions.SetLimit(limit)
// findOptions.SetSkip(offset)
// findOptions.SetProjection(selector)
// findOptions.SetSort(sort)
// resSlice slice  此处需要通过反射把文档解析到切片,
// 参考mgo  https://github.com/go-mgo/mgo/blob/v2-unstable/session.go
func (m *Collection) FindAll(ctx context.Context, findOptions *options.FindOptions, filter interface{}, resSlice interface{}) error {
	// 必须是切片
	resultV := reflect.ValueOf(resSlice)
	if resultV.Kind() != reflect.Ptr || resultV.Elem().Kind() != reflect.Slice {
		return ErrMustSlice
	}

	cur, err := m.GetCollection().Find(context.TODO(), filter, findOptions)
	if err != nil {
		return err
	}

	// Close the cursor once finished
	defer cur.Close(ctx)

	i := 0
	sliceVal := resultV.Elem()
	elemType := sliceVal.Type().Elem()

	// Finding multiple documents returns a cursor  返回游标
	// Iterating through the cursor allows us to decode documents one at a time
	for cur.Next(context.TODO()) {
		if sliceVal.Len() == i {
			newElem := reflect.New(elemType)
			sliceVal = reflect.Append(sliceVal, newElem.Elem())
			sliceVal = sliceVal.Slice(0, sliceVal.Cap())
		}
		currElem := sliceVal.Index(i).Addr().Interface()
		if err = cur.Decode(currElem); err != nil {
			return err
		}
		i++
	}

	if err := cur.Err(); err != nil {
		return err
	}
	resultV.Elem().Set(sliceVal.Slice(0, i))
	return nil
}

//preInsertData 插入数据 增加ID 创建时间 和更新时间
func preInsertData(obj interface{}) {
	curTime := time.Now()

	insertPreField := map[string]interface{}{
		"ID":         primitive.NewObjectID(),
		"CreateTime": curTime,
		"UpdateTime": curTime,
	}

	for key, val := range insertPreField {
		setStructValue(obj, key, val)
	}
}

// SetStructValue 通过反射给指定field赋值
func setStructValue(data interface{}, field string, value interface{}) {
	v := reflect.ValueOf(data)
	v = v.Elem() //实际取得的对象
	resV := v.FieldByName(field)

	if resV.IsValid() {
		val := reflect.ValueOf(value)
		resV.Set(val)
	}
}

// InsertOne 插入文档 返回插入id
func (m *Collection) InsertOne(ctx context.Context, data interface{}) (interface{}, error) {
	preInsertData(data)
	insertResult, err := m.GetCollection().InsertOne(ctx, data)
	if err != nil {
		return "", err
	}
	return insertResult.InsertedID, nil
}

// InsertMany 批量插入
func (m *Collection) InsertMany(ctx context.Context, data []interface{}) ([]interface{}, error) {
	for i := 0; i < len(data); i++ {
		preInsertData(data[i])
	}
	insertResult, err := m.GetCollection().InsertMany(ctx, data)
	if err != nil {
		return nil, err
	}
	return insertResult.InsertedIDs, nil
}

// UpdateOne 更新文档
func (m *Collection) UpdateOne(ctx context.Context, filter, updateData interface{}) (int64, error) {
	update := bson.M{
		"$set": updateData,
	}
	result, err := m.GetCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.UpsertedCount, nil
}

// UpdateMany 更新多个文档
func (m *Collection) UpdateMany(ctx context.Context, filter, updateData interface{}) (int64, error) {
	update := bson.M{
		"$set": updateData,
	}
	result, err := m.GetCollection().UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.UpsertedCount, nil
}

// DeleteMany 批量删除文档
func (m *Collection) Delete(ctx context.Context, filter interface{}) (int64, error) {
	result, err := m.GetCollection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}