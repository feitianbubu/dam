## set bucket public read only
    docker compose exec -it minio sh
    mc alias set myminio http://localhost:9000 $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD
    mc anonymous set public myminio/dam-cn