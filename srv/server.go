package srv

import (
	"context"
	"fmt"
	"time"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/json"
	"github.com/ability-sh/abi-micro-user/pb"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	G "google.golang.org/grpc"
)

type server struct {
}

func setUser(a *pb.User, rs bson.M) {
	a.Id = dynamic.StringValue(rs["_id"], "")
	a.Ctime = int32(dynamic.IntValue(rs["ctime"], 0))
	a.Name = dynamic.StringValue(rs["name"], "")
	a.Nick = dynamic.StringValue(rs["nick"], "")
}

func toUserItems(rs []bson.M) []*pb.User {
	vs := []*pb.User{}
	for _, r := range rs {
		v := &pb.User{}
		setUser(v, r)
		vs = append(vs, v)
	}
	return vs
}

func setInfo(a *pb.Info, rs bson.M) {
	b, _ := json.Marshal(rs["info"])
	a.Info = string(b)
}

func (s *server) UserCreate(c context.Context, task *pb.UserCreateTask) (*pb.UserResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Name == "" {
		return &pb.UserResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param name"}, nil
	}

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	db_user := db.Collection(config.GetUserDB())

	id := config.NewID()
	if task.Password == "" {
		task.Password = config.NewPassword()
	}
	passowrd := config.SecPassword(task.Password)
	ctime := int32(time.Now().Unix())

	d := bson.D{bson.E{"_id", id},
		bson.E{"name", task.Name},
		bson.E{"passowrd", passowrd},
		bson.E{"ctime", ctime}}

	if task.Nick != "" {
		d = append(d, bson.E{"nick", task.Nick})
	}

	_, err = db_user.InsertOne(c, d)

	if err != nil {
		if err == mongo.ErrInvalidIndexValue {
			return &pb.UserResult{Errno: ERRNO_INDEX_VALUE, Errmsg: err.Error()}, nil
		}
		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.User{Id: id, Nick: task.Nick, Name: task.Name, Ctime: ctime}

	return &pb.UserResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) UserSet(c context.Context, task *pb.UserSetTask) (*pb.UserResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Uid == "" {
		return &pb.UserResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param uid"}, nil
	}

	if task.Password != nil && task.Password.Value == "" {
		return &pb.UserResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param password"}, nil
	}

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	db_user := db.Collection(config.GetUserDB())

	set := bson.D{}

	if task.Name != nil {
		set = append(set, bson.E{"name", task.Name})
	}

	if task.Nick != nil {
		if task.Nick.Value == "" {
			set = append(set, bson.E{"nick", nil})
		} else {
			set = append(set, bson.E{"nick", task.Nick.Value})
		}
	}

	if task.Password != nil {
		set = append(set, bson.E{"password", config.SecPassword(task.Password.Value)})
	}

	a := &pb.User{}

	if len(set) == 0 {

		var rs bson.M

		err = db_user.FindOne(c,
			bson.D{bson.E{"_id", task.Uid}}).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.UserResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found user"}, nil
			}
			return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		setUser(a, rs)

	} else {

		opts := options.FindOneAndUpdate().SetUpsert(true)

		var rs bson.M

		err = db_user.FindOneAndUpdate(c,
			bson.D{bson.E{"_id", task.Uid}}, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

		if err != nil {
			return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		if task.Nick != nil {
			rs["nick"] = task.Nick.Value
		}

		if task.Name != nil {
			rs["name"] = task.Name.Value
		}

		setUser(a, rs)

	}

	return &pb.UserResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) UserGet(c context.Context, task *pb.UserGetTask) (*pb.UserResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Uid == "" && task.Name == "" && task.Nick == "" {
		return &pb.UserResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param uid/name/nick"}, nil
	}

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	db_user := db.Collection(config.GetUserDB())

	a := &pb.User{}

	d := bson.D{}

	if task.Uid != "" {
		d = append(d, bson.E{"_id", task.Uid})
	}

	if task.Name != "" {
		d = append(d, bson.E{"name", task.Name})
	}

	if task.Nick != "" {
		d = append(d, bson.E{"nick", task.Nick})
	}

	var rs bson.M

	err = db_user.FindOne(c, d).Decode(&rs)

	if err != nil {

		if err == mongo.ErrNoDocuments {

			if task.AutoCreated && task.Name != "" {

				opts := options.FindOneAndUpdate().SetUpsert(true)

				var rs bson.M
				id := config.NewID()
				passowrd := config.SecPassword(config.NewPassword())
				ctime := int32(time.Now().Unix())

				set := bson.D{bson.E{"name", task.Name}, bson.E{"_id", id}, bson.E{"passowrd", passowrd}, bson.E{"ctime", ctime}}

				err = db_user.FindOneAndUpdate(c,
					d, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

				if err == mongo.ErrNoDocuments {
					rs = bson.M{"_id": id, "ctime": ctime, "name": task.Name}
				} else if err != nil {
					return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
				}

				setUser(a, rs)

				return &pb.UserResult{Errno: ERRNO_OK, Data: a}, nil
			}

			return &pb.UserResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found user"}, nil
		}

		return &pb.UserResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	setUser(a, rs)

	return &pb.UserResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) UserQuery(c context.Context, task *pb.UserQueryTask) (*pb.UserQueryResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.UserQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.UserQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	n := task.N
	p := task.P

	if n < 1 {
		n = 20
	}

	rs := &pb.UserQueryResult{}

	db := conn.Database(config.Db)

	db_user := db.Collection(config.GetUserDB())

	opts := options.Find().SetSort(bson.D{bson.E{"ctime", -1}}).SetLimit(int64(n))

	filter := bson.D{}

	if task.Q != "" {
		filter = append(filter, bson.E{"title", bson.E{"$regex", task.Q}})
	}

	if p > 0 {

		opts = opts.SetSkip(int64(n * (p - 1)))

		totalCount, err := db_user.CountDocuments(c, filter)

		if err != nil {
			return &pb.UserQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		count := int32(totalCount) / n

		if int32(totalCount)%n != 0 {
			count = count + 1
		}

		rs.Page = &pb.Page{P: p, N: n, TotalCount: int32(totalCount), Count: count}

	}

	cursor, err := db_user.Find(c, filter, opts)

	if err != nil {
		return &pb.UserQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return &pb.UserQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	rs.Items = toUserItems(items)

	return rs, nil
}

func (s *server) InfoSet(c context.Context, task *pb.InfoSetTask) (*pb.InfoResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Uid == "" {
		return &pb.InfoResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param uid"}, nil
	}

	if task.Key == "" {
		return &pb.InfoResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param key"}, nil
	}

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.InfoResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.InfoResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	db_info := db.Collection(config.GetInfoDB(task.Uid))

	opts := options.FindOneAndUpdate().SetUpsert(true)

	id := fmt.Sprintf("%s-%s", task.Uid, task.Key)

	set := bson.D{}

	var info interface{} = nil

	if task.Info != "" {
		err = json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.InfoResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
		dynamic.Each(info, func(key interface{}, value interface{}) bool {
			set = append(set, bson.E{fmt.Sprintf("info.%s", dynamic.StringValue(key, "")), value})
			return true
		})
	}

	var rs bson.M

	err = db_info.FindOneAndUpdate(c,
		bson.D{bson.E{"_id", id}}, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

	if err == mongo.ErrNoDocuments {
		rs = bson.M{"info": info, "_id": id}
	} else if err != nil {
		return &pb.InfoResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	{
		var i = rs["info"]
		if i == nil {
			i = map[string]interface{}{}
			rs["info"] = i
		}
		dynamic.Each(info, func(key interface{}, value interface{}) bool {
			dynamic.Set(i, dynamic.StringValue(key, ""), value)
			return true
		})
	}

	a := &pb.Info{}

	setInfo(a, rs)

	return &pb.InfoResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) InfoGet(c context.Context, task *pb.InfoGetTask) (*pb.InfoResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Uid == "" {
		return &pb.InfoResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param uid"}, nil
	}

	if task.Key == "" {
		return &pb.InfoResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param key"}, nil
	}

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.InfoResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.InfoResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	db_info := db.Collection(config.GetInfoDB(task.Uid))

	var rs bson.M

	id := fmt.Sprintf("%s-%s", task.Uid, task.Key)

	err = db_info.FindOne(c, bson.D{bson.E{"_id", id}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.InfoResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found info"}, nil
		}
		return &pb.InfoResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.Info{}

	setInfo(a, rs)

	return &pb.InfoResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) Login(c context.Context, task *pb.LoginTask) (*pb.LoginResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Name == "" {
		return &pb.LoginResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param name"}, nil
	}

	if task.Password == "" {
		return &pb.LoginResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param password"}, nil
	}

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.LoginResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.LoginResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	db_user := db.Collection(config.GetUserDB())

	var rs bson.M

	err = db_user.FindOne(c, bson.D{bson.E{"name", task.Name}, bson.E{"password", task.Password}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.LoginResult{Errno: ERRNO_LOGIN, Errmsg: "not found user / password"}, nil
		}
		return &pb.LoginResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	return &pb.LoginResult{Errno: ERRNO_OK}, nil
}

func (s *server) UserBatchGet(c context.Context, task *pb.UserBatchGetTask) (*pb.UserBatchGetResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.UserBatchGetResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.UserBatchGetResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	db_user := db.Collection(config.GetUserDB())

	cursor, err := db_user.Find(c, bson.D{bson.E{"_id", bson.E{"$in", task.Uid}}})

	if err != nil {
		return &pb.UserBatchGetResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return &pb.UserBatchGetResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	vs := toUserItems(items)

	userSet := map[string]*pb.User{}

	for _, v := range vs {
		userSet[v.Id] = v
	}

	ret := []*pb.User{}

	for _, uid := range task.Uid {
		ret = append(ret, userSet[uid])
	}

	return &pb.UserBatchGetResult{Errno: ERRNO_OK, Items: ret}, nil
}

func (s *server) InfoBatchGet(c context.Context, task *pb.InfoBatchGetTask) (*pb.InfoBatchGetResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	config, err := GetUserService(ctx, SERVICE_USER)

	if err != nil {
		return &pb.InfoBatchGetResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.InfoBatchGetResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(config.Db)

	vs := []*pb.Info{}

	for _, uid := range task.Uid {

		id := fmt.Sprintf("%s-%s", task.Uid, task.Key)

		db_info := db.Collection(config.GetInfoDB(uid))

		var rs bson.M

		err = db_info.FindOne(c, bson.D{bson.E{"_id", id}}).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				vs = append(vs, nil)
				continue
			} else {
				return &pb.InfoBatchGetResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
			}
		}

		a := &pb.Info{}
		setInfo(a, rs)
		vs = append(vs, a)
	}

	return &pb.InfoBatchGetResult{Errno: ERRNO_OK, Items: vs}, nil
}

func Reg(s *G.Server) {
	pb.RegisterServiceServer(s, &server{})
}
