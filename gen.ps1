goctl rpc protoc trade_id_mgr.proto --go_out=. --go-grpc_out=. --zrpc_out=.

pushd
cd model/mysql
# 生成mysql代码
goctl model mysql ddl -src id_generator.sql -dir .
popd