BINARY_NAME=macautostartupeditor

build:
	mkdir -p out/bin
	GOARCH=arm64 GOOS=darwin go build -o out/bin/${BINARY_NAME} .

buildapp:
	mkdir -p out/app
	make build
	@echo 'set appPath to path to me' > out/bin/script.applescript
	@echo 'set binaryPath to POSIX path of appPath & "Contents/MacOS/${BINARY_NAME}"' >> out/bin/script.applescript
	@echo 'tell application "Terminal"' >> out/bin/script.applescript
	@echo '    do script quoted form of binaryPath & "; kill -9 $$(ps -p $$PPID -o ppid=)"' >> out/bin/script.applescript
	@echo '    activate' >> out/bin/script.applescript
	@echo '    set bounds of front window to {100, 100, 650, 480}' >> out/bin/script.applescript
	@echo 'end tell' >> out/bin/script.applescript
	osacompile -o out/app/Mase.app out/bin/script.applescript
	cp out/bin/${BINARY_NAME} out/app/Mase.app/Contents/MacOS/${BINARY_NAME}
	chmod +x out/app/Mase.app/Contents/MacOS/${BINARY_NAME}
	cp assets/icns/icon.icns out/app/Mase.app/Contents/Resources/applet.icns
clean:
	go clean
	rm -rf out/*