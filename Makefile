default: c b i

c:
		rm -rf ./down ./bin ~/.local/bin/down

b: c
		go build  -o "bin/down" ./main.go
		ln -s bin/down down

r: b
		exec ./bin/down


i: b
		go install
		cp -r ./bin/down ~/.local/bin/down


#vim:ft=bash
