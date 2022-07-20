package srv

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/iid"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/mongodb"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserService struct {
	config   interface{} `json:"-"`
	name     string      `json:"-"`
	Prefix   string      `json:"prefix"`
	BasePath string      `json:"basePath"`
	Aid      int64       `json:"aid"`     //区域ID
	Nid      int64       `json:"nid"`     //节点ID
	Expires  int64       `json:"expires"` //过期秒数
	Db       string      `json:"db"`      // mongodb db
	Secret   string      `json:"secret"`  // secret
	IID      *iid.IID    `json:"-"`
}

func newUserService(name string, config interface{}) *UserService {
	return &UserService{name: name, config: config}
}

/**
* 服务名称
**/
func (s *UserService) Name() string {
	return s.name
}

/**
* 服务配置
**/
func (s *UserService) Config() interface{} {
	return s.config
}

/**
* 初始化服务
**/
func (s *UserService) OnInit(ctx micro.Context) error {

	dynamic.SetValue(s, s.config)

	s.IID = iid.NewIID(s.Aid, s.Nid)

	ctx.Printf("db init ...")

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return err
	}

	db := conn.Database(s.Db)

	c := context.Background()

	db_user := db.Collection(s.GetUserDB())

	{
		indexes := db_user.Indexes()
		_, err = indexes.CreateMany(c, []mongo.IndexModel{
			{
				Keys: bson.D{bson.E{"ctime", -1}},
			},
			{
				Keys:    bson.D{bson.E{"nick", 1}},
				Options: options.Index().SetUnique(true).SetSparse(true),
			},
			{
				Keys:    bson.D{bson.E{"name", 1}},
				Options: options.Index().SetUnique(true),
			},
		})
		if err != nil {
			return err
		}
	}

	ctx.Printf("db init done")

	return nil
}

/**
* 校验服务是否可用
**/
func (s *UserService) OnValid(ctx micro.Context) error {
	return nil
}

func (s *UserService) Recycle() {

}

func (s *UserService) NewID() string {
	return strconv.FormatInt(s.IID.NewID(), 36)
}

func (s *UserService) SecPassword(p string) string {
	m := md5.New()
	m.Write([]byte(p))
	m.Write([]byte(s.Secret))
	return hex.EncodeToString(m.Sum(nil))
}

func (s *UserService) NewPassword() string {
	return uuid.New().String()
}

func (s *UserService) GetInfoDB(uid string) string {
	n := len(uid)
	if n > 4 {
		return fmt.Sprintf("%sinfo_%s", s.Prefix, uid[0:4])
	}
	return fmt.Sprintf("%sinfo_%s", s.Prefix, uid)
}

func (s *UserService) GetUserDB() string {
	return fmt.Sprintf("%suser", s.Prefix)
}

func GetUserService(ctx micro.Context, name string) (*UserService, error) {
	s, err := ctx.GetService(name)
	if err != nil {
		return nil, err
	}
	ss, ok := s.(*UserService)
	if ok {
		return ss, nil
	}
	return nil, fmt.Errorf("service %s not instanceof *UserService", name)
}
