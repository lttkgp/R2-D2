package swagger

//go:generate rm -rf server
//go:generate mkdir -p server
//go:generate swagger generate server --quiet --target server --name r2d2 --spec swagger.yml --exclude-main
