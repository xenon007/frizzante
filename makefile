update: www/package.json go.mod
	cd www && bun update
	cd www && bunx vite build --ssr render.server.js --outDir dist/server
	cd www && ./node_modules/.bin/esbuild dist/server/render.server.js --bundle --outfile=dist/server/render.server.js --format=esm --allow-overwrite
	cd www && bunx vite build --outDir dist/client
	go mod tidy

clean:
	go clean
	rm out -fr
	rm .temp -fr
	rm www/dist/server -fr
	rm www/dist/client -fr
	rm www/node_modules -fr

test: clean update
	go test

certificate-interactive:
	openssl req -newkey rsa:2048 -new -nodes -x509 -days 3650 -keyout key.pem -out certificate.pem

certificate:
	openssl req -newkey rsa:2048 -new -nodes -x509 -days 3650 -keyout key.pem -out certificate.pem -nodes -subj \
	"/C=XX/ST=Test/L=Test/O=Test/OU=Test/CN=Test"


hooks:
	printf "#!/usr/bin/env bash\n" > .git/hooks/pre-commit
	printf "make test" >> .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit