$MODULE_NAME = "trade_id_mgr"
$VERSION = "v1.0.0"
$IMAGE_NAME = "${MODULE_NAME}:${VERSION}"

docker rm -f $MODULE_NAME
docker rmi -f $IMAGE_NAME
docker build -t $IMAGE_NAME .
docker run -d --name $MODULE_NAME --network dev_pay_net -p 30884:8080 $IMAGE_NAME
# docker run -d --name trade_id_mgr --network dev_pay_net -p 30884:8080 trade_id_mgr:v1.0.0
docker ps
docker logs $MODULE_NAME
